package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/conf"
	"github.com/zlyuancn/order/dao"
	"github.com/zlyuancn/order/mq"
	"github.com/zlyuancn/order/order_model"
)

var Order = orderCli{}

type orderCli struct{}

/*
创建订单, orderID重复会报错

	order 订单相关数据
	extend 扩展数据
	enableCompensation 是否启用后置补偿, 会提交一个mq消息到队列中
	compensationDelayTime 开始补偿延迟时间. 秒
*/
func (o orderCli) CreateOrder(ctx context.Context, order *order_model.Order, extend interface{},
	enableCompensation bool) error {
	err := o.SendCompensationSignal(ctx, order.OrderID, order.Uid, enableCompensation)
	if err != nil {
		return err
	}

	v, err := o.order2DBModel(order, extend, order_model.OrderStatus_Forwarding)
	if err != nil {
		logger.Log.Error(ctx, "CreateOrder order2DBModel err",
			zap.Any("order", order),
			zap.Error(err),
		)
		return err
	}
	v.Remark = "Created"

	_, err = dao.Dao(order.Uid).CreateOneModel(ctx, v)
	if err != nil {
		logger.Log.Error(ctx, "CreateOrder dao.CreateOneModel err",
			zap.Any("v", v),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (orderCli) order2DBModel(order *order_model.Order, extend interface{}, status order_model.OrderStatus) (*dao.Model, error) {
	v := &dao.Model{
		OrderID:         order.OrderID,
		OrderType:       byte(order.OrderType),
		OrderStatus:     byte(status),
		PayType:         byte(order.PayType),
		PayStatus:       byte(order.PayStatus),
		PayAmount:       order.PayAmount,
		ThirdPayOrderID: order.ThirdPayOrderID,
		Uid:             order.Uid,
	}
	if extend != nil {
		extendText, err := sonic.MarshalString(extend)
		if err != nil {
			return nil, err
		}
		v.Extend = extendText
	}
	return v, nil
}

func (o orderCli) SendCompensationSignal(ctx context.Context, orderID, uid string,
	enableCompensation bool) error {
	if !enableCompensation {
		logger.Log.Warn(ctx, "order create no send mq",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
		)
		return nil
	}

	if !conf.Conf.AllowMqCompensation {
		logger.Log.Warn(ctx, "order sendMq but AllowMqCompensation is false",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
		)
		return errors.New("order sendMq but AllowMqCompensation is false")
	}

	// 发送补偿mq
	orderMsg := &order_model.OrderMqMsg{
		OrderID: orderID,
		Uid:     uid,
	}

	err := mq.Send(ctx, orderMsg)
	if err != nil {
		logger.Log.Error(ctx, "CreateOrder Produce Compensation Mq msg err",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// 订单操作锁, 用于防止多线程操作订单, 比如mq重复同时消费
func (o orderCli) orderDBLock(ctx context.Context, orderID string) (
	unlock func(ctx context.Context), ok bool, err error) {
	key := order_model.RedisOrderLockOP + orderID
	expireTime := conf.Conf.OrderLockDBExpire
	un, ok, err := dao.SetRedisLock(ctx, key, expireTime)
	if !ok || err != nil {
		return nil, ok, err
	}
	return func(ctx context.Context) {
		_, _ = un(ctx, conf.Conf.OrderUnlockDBLimitProcessTime)
	}, ok, err
}

// 获取订单
func (orderCli) GetOrder(ctx context.Context, orderID, uid string) (
	*order_model.Order, string, order_model.OrderStatus, error) {
	model, err := dao.Dao(uid).GetOne(ctx, orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", 0, OrderNotFoundErr
		}
		logger.Log.Error(ctx, "GetOrder dao.GetOne err",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
			zap.Error(err),
		)
		return nil, "", 0, err
	}

	order := &order_model.Order{
		OrderID:   orderID,
		OrderType: order_model.OrderType(model.OrderType),

		PayType:         order_model.OrderPayType(model.PayType),
		PayStatus:       order_model.OrderPayStatus(model.PayStatus),
		PayAmount:       model.PayAmount,
		ThirdPayOrderID: model.ThirdPayOrderID,

		Uid: uid,
	}
	status := order_model.OrderStatus(model.OrderStatus)
	return order, model.Extend, status, nil
}

/*
业务推进

	status 当前订单状态
*/
func (o orderCli) Forward(ctx context.Context, orderID, uid string) (
	*order_model.Order, order_model.OrderStatus, error) {
	return o.forward(ctx, orderID, uid, false)
}

func (o orderCli) forward(ctx context.Context, orderID, uid string, isMq bool) (
	*order_model.Order, order_model.OrderStatus, error) {
	unlock, ok, err := o.orderDBLock(ctx, orderID)
	if err != nil {
		logger.Log.Error(ctx, "Order Forward orderDBLock err",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
			zap.Error(err),
		)
		return nil, 0, err
	}
	if !ok {
		logger.Log.Warn(ctx, "Order Forward orderDBLock is failed",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
		)
		return nil, 0, fmt.Errorf("Order Forward orderDBLock is failed")
	}
	defer unlock(ctx)

	// 获取订单数据
	order, extendText, status, err := o.GetOrder(ctx, orderID, uid)
	if err != nil {
		logger.Log.Error(ctx, "Order forward GetOrder err",
			zap.String("orderID", orderID),
			zap.String("uid", uid),
			zap.Error(err),
		)
		if err == OrderNotFoundErr && isMq { // mq重试发现订单不存在则忽略消息
			return nil, 0, nil
		}
		return nil, 0, err
	}

	// 获取业务
	ob, ok := o.GetOrderBusiness(order.OrderType)
	if !ok {
		return nil, 0, fmt.Errorf("Order forward OrderType %v not found OrderBusiness", order.OrderType)
	}

	// 解析业务扩展数据
	extend := ob.NewExtendStruct(ctx)
	if extend != nil && extendText != "" {
		err = sonic.UnmarshalString(extendText, extend)
		if err != nil {
			return nil, 0, fmt.Errorf("Order forward Unmarshal extend err. orderID=%v, err=%v", order.OrderID, err)
		}
	}

	// 检查状态
	if status != order_model.OrderStatus_Forwarding {
		logger.Log.Warn(ctx, "Order Forward order not is create status",
			zap.Any("order", order),
			zap.Any("status", status),
		)
		if status == order_model.OrderStatus_Finish {
			err := ob.ForwardFinishCallback(ctx, order, extend)
			if err != nil {
				logger.Log.Error(ctx, "Order forward call ForwardFinishCallback err",
					zap.Any("order", order),
					zap.Any("extend", extend),
					zap.Int("status", int(status)),
					zap.Error(err),
				)
				return nil, 0, err
			}
		} else {
			err := ob.ForwardAbnormalCallback(ctx, order, extend, status)
			if err != nil {
				logger.Log.Error(ctx, "Order forward call ForwardAbnormalCallback err",
					zap.Any("order", order),
					zap.Any("extend", extend),
					zap.Int("status", int(status)),
					zap.Error(err),
				)
				return nil, 0, err
			}
		}
		return order, status, nil
	}

	// 业务检查是否允许推进
	cancelCause, err := ob.CanForward(ctx, order, extend)
	if err != nil {
		logger.Log.Error(ctx, "Order forward call CanForward err",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Any("status", status),
			zap.Error(err),
		)
		return nil, 0, err
	}
	if cancelCause != "" {
		logger.Log.Error(ctx, "Order forward call CanForward got cancel forward",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Int("status", int(status)),
			zap.String("cancelCause", cancelCause),
		)
		status = order_model.OrderStatus_BusinessCancelForward
		err = o.UpdateOrderStatus(ctx, order.OrderID, order.Uid, extend, status, cancelCause)
		if err != nil {
			logger.Log.Error(ctx, "Order forward cancel set UpdateOrderStatus err",
				zap.Any("order", order),
				zap.Any("extend", extend),
				zap.Int("status", int(status)),
				zap.String("cancelCause", cancelCause),
				zap.Error(err),
			)
			return nil, 0, err
		}
		err := ob.ForwardAbnormalCallback(ctx, order, extend, order_model.OrderStatus_BusinessCancelForward)
		if err != nil {
			logger.Log.Error(ctx, "Order forward call ForwardAbnormalCallback err",
				zap.Any("order", order),
				zap.Any("extend", extend),
				zap.Int("status", int(status)),
				zap.String("cancelCause", cancelCause),
				zap.Error(err),
			)
			return nil, 0, err
		}
		return nil, 0, OrderBusinessCancelForwardErr
	}

	// 扣款
	ok, err = o.deductBalance(ctx, order, extend)
	if err != nil {
		logger.Log.Error(ctx, "Order forward DeductBalance err",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Error(err),
		)
		return nil, 0, err
	}
	if !ok {
		status = order_model.OrderStatus_InsufficientBalance
		// 余额不足, 这里 DeductBalance 已经自动更新了订单状态
		logger.Log.Warn(ctx, "Order forward DeductBalance is InsufficientBalance",
			zap.Any("order", order),
			zap.Any("extend", extend),
		)
		err := ob.ForwardAbnormalCallback(ctx, order, extend, status)
		if err != nil {
			logger.Log.Error(ctx, "Order forward deductBalance fail and call ForwardAbnormalCallback err",
				zap.Any("order", order),
				zap.Any("extend", extend),
				zap.Int("status", int(status)),
				zap.String("cancelCause", cancelCause),
				zap.Error(err),
			)
			return nil, 0, err
		}
		return order, order_model.OrderStatus_InsufficientBalance, nil
	}

	// 发货
	err = ob.Delivery(ctx, order, extend)
	if err != nil {
		logger.Log.Error(ctx, "Order forward Delivery err",
			zap.Any("order", order),
			zap.Error(err),
		)
		return nil, 0, err
	}

	status = order_model.OrderStatus_Finish
	// 更新订单状态
	err = o.UpdateOrderStatus(ctx, order.OrderID, order.Uid, extend, status, "forward finish")
	if err != nil {
		logger.Log.Error(ctx, "Order forward finish but set updateOrderStatus err",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Any("status", status),
			zap.Error(err),
		)
		return nil, 0, err
	}

	err = ob.ForwardFinishCallback(ctx, order, extend)
	if err != nil {
		logger.Log.Error(ctx, "Order forward finish but call ForwardAbnormalCallback err",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Int("status", int(status)),
			zap.Error(err),
		)
		return nil, 0, err
	}

	return order, order_model.OrderStatus_Finish, nil
}

/*
扣除余额

	order 订单相关数据

return 扣除余额是否成功, false一般为余额不足

如果是扣款发生余额不足, 会打上余额不足状态.
*/
func (o orderCli) deductBalance(ctx context.Context, order *order_model.Order, extend interface{}) (bool, error) {
	if order.PayStatus == order_model.OrderPayStatus_Success {
		return true, nil
	}

	deductOK := false
	switch order.PayType {
	case order_model.OrderPayType_None: // 无需支付
		return true, nil
	default:
		return false, fmt.Errorf("order deductBalance unrealized payType=%v", order.PayType)
	}

	if !deductOK {
		status := order_model.OrderStatus_InsufficientBalance
		err := o.UpdateOrderStatus(ctx, order.OrderID, order.Uid, extend, status, "InsufficientBalance")
		if err != nil {
			logger.Log.Error(ctx, "Order deductBalance fail and set UpdateOrderStatus err",
				zap.Any("order", order),
				zap.Any("extend", extend),
				zap.Int("status", int(status)),
				zap.Error(err),
			)
			return false, err
		}
		return false, nil
	}

	order.PayStatus = order_model.OrderPayStatus_Success
	status := order_model.OrderStatus_Forwarding
	err := dao.Dao(order.Uid).SetPayStatus(ctx, order.OrderID, "", byte(order.PayStatus), "Auto Pay")
	if err != nil {
		logger.Log.Error(ctx, "Order deductBalance finish but set PayStatus err",
			zap.Any("order", order),
			zap.Any("extend", extend),
			zap.Int("status", int(status)),
			zap.Error(err),
		)
		return false, err
	}
	return true, nil
}

// 更新付费状态
func (o orderCli) UpdatePayStatus(ctx context.Context, orderID, uid string, payStatus order_model.OrderPayStatus, remark string) error {
	err := dao.Dao(uid).SetPayStatus(ctx, orderID, "", byte(payStatus), remark)
	if err != nil {
		logger.Log.Error(ctx, "Order UpdatePayStatus call set PayStatus err",
			zap.Any("orderID", orderID),
			zap.Any("uid", uid),
			zap.Int("payStatus", int(payStatus)),
			zap.String("remark", remark),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// 更新订单状态和扩展数据
func (o orderCli) UpdateOrderStatus(ctx context.Context, orderID, uid string, extend interface{}, status order_model.OrderStatus,
	remark string) error {
	var extendText string
	if extend != nil {
		v, err := sonic.MarshalString(extend)
		if err != nil {
			logger.Log.Error(ctx, "order UpdateOrderStatus Marshal extend err",
				zap.Any("orderID", orderID),
				zap.Any("extend", extend),
				zap.Any("status", status),
				zap.Any("remark", remark),
				zap.Error(err),
			)
			return err
		}
		extendText = v
	}
	// 更新状态
	err := dao.Dao(uid).UpdateOrderStatus(ctx, orderID, extendText, status, remark)
	if err != nil {
		logger.Log.Error(ctx, "order UpdateOrderStatus err",
			zap.Any("orderID", orderID),
			zap.Any("extend", extend),
			zap.Any("status", status),
			zap.Any("remark", remark),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// 根据用户订单号生成单号
func (o orderCli) GenOIDByUserOID(orderType order_model.OrderType, uid, userOrderID string) string {
	shard := dao.GenShard(uid)
	return fmt.Sprintf("order-uoid-%d-%s-%s-%s", orderType, shard, uid, userOrderID)
}

// 根据第三方订单号生成单号
func (o orderCli) GenOIDByThirdPayOID(orderType order_model.OrderType, uid, thirdPayOid string) string {
	shard := dao.GenShard(uid)
	return fmt.Sprintf("order-third-%d-%s-%s-%s", orderType, shard, uid, thirdPayOid)
}

// 生成订单号
func (o orderCli) GenOID(ctx context.Context, orderType order_model.OrderType, uid string) (string, error) {
	shard := dao.GenShard(uid)
	key := fmt.Sprintf("%s:%d-%s", order_model.RedisOrderSeqNoIncr, orderType, shard)
	incrV, err := dao.RedisIncrBy(ctx, key, 1)
	if err != nil {
		logger.Log.Error(ctx, "order GenOrderID err",
			zap.Error(err),
		)
		return "", err
	}
	orderID := fmt.Sprintf("order-sgen-%d-%s-%d-%d", orderType, shard, incrV, time.Now().Unix())
	return orderID, nil
}
