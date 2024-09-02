package order

import (
	"context"

	"github.com/zly-app/zapp/filter"

	"github.com/zlyuancn/order/order_model"
)

const (
	clientType = "order"
	clientName = "sdk"
)

type coReq struct {
	Order              *order_model.Order `json:"Order"`
	Extend             interface{}        `json:"Extend,omitempty"`
	EnableCompensation bool               `json:"EnableCompensation,omitempty"`
}

/*
创建订单, orderID重复会报错

	order 订单相关数据
	extend 扩展数据
	enableCompensation 是否启用后置补偿, 会提交一个mq消息到队列中
	compensationDelayTime 开始补偿延迟时间. 秒
*/
func CreateOrder(ctx context.Context, order *order_model.Order, extend interface{},
	enableCompensation bool) error {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "CreateOrder")
	r := &coReq{
		Order:              order,
		Extend:             extend,
		EnableCompensation: enableCompensation,
	}
	_, err := chain.Handle(ctx, r, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		r := req.(*coReq)
		return nil, orderApi.CreateOrder(ctx, r.Order, r.Extend, r.EnableCompensation)
	})
	return err
}

type scsReq struct {
	OrderID string
	UID     string
}

// 主动发送补偿信号
func SendCompensationSignal(ctx context.Context, orderID, uid string) error {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "SendCompensationSignal")
	r := &scsReq{
		OrderID: orderID,
		UID:     uid,
	}
	_, err := chain.Handle(ctx, r, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		r := req.(*scsReq)
		return nil, orderApi.SendCompensationSignal(ctx, r.OrderID, r.UID)
	})
	return err
}

type goReq struct {
	OrderID string
	UID     string
}
type goRsp struct {
	Order  *order_model.Order      `json:"Order"`
	Extend string                  `json:"Extend,omitempty"`
	Status order_model.OrderStatus `json:"Status"`
}

// 获取订单
func GetOrder(ctx context.Context, orderID, uid string) (
	*order_model.Order, string, order_model.OrderStatus, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "GetOrder")
	r := &goReq{
		OrderID: orderID,
		UID:     uid,
	}
	sp := &goRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*goReq)
		sp := rsp.(*goRsp)
		order, extend, status, err := orderApi.GetOrder(ctx, r.OrderID, r.UID)
		sp.Order = order
		sp.Extend = extend
		sp.Status = status
		return err
	})
	return sp.Order, sp.Extend, sp.Status, err
}

type fReq struct {
	Order  *order_model.Order `json:"Order"`
	Extend interface{}        `json:"Extend,omitempty"`
}
type fRsp struct {
	Order  *order_model.Order      `json:"Order"`
	Status order_model.OrderStatus `json:"Status"`
}

/*
业务推进刚创建的订单

注意, 只有在同一个线程中创建的订单才能调用这个方法, 通过 GetOrder 获取到的 order 不能调用这个方法.
否则可能会导致订单数据不一致, 因为里面涉及到锁的问题. 一旦别的线程介入了操作, 就可能导致你的 order 数据被修改了但是传入给 Forward
的 order 仍然是旧数据
*/
func Forward(ctx context.Context, order *order_model.Order, extend interface{}) (
	*order_model.Order, order_model.OrderStatus, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "Forward")
	r := &fReq{
		Order:  order,
		Extend: extend,
	}
	sp := &fRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*fReq)
		sp := rsp.(*fRsp)
		order, status, err := orderApi.Forward(ctx, r.Order, r.Extend)
		sp.Order = order
		sp.Status = status
		return err
	})
	return sp.Order, sp.Status, err
}

type foidReq struct {
	OrderID string
	UID     string
}
type foidRsp struct {
	Order  *order_model.Order      `json:"Order"`
	Status order_model.OrderStatus `json:"Status"`
}

/*
业务推进

	status 当前订单状态
*/
func ForwardOrderID(ctx context.Context, orderID, uid string) (
	*order_model.Order, order_model.OrderStatus, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "ForwardOrderID")
	r := &foidReq{
		OrderID: orderID,
		UID:     uid,
	}
	sp := &foidRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*foidReq)
		sp := rsp.(*foidRsp)
		order, status, err := orderApi.ForwardOrderID(ctx, r.OrderID, r.UID)
		sp.Order = order
		sp.Status = status
		return err
	})
	return sp.Order, sp.Status, err
}

type upsReq struct {
	OrderID   string
	UID       string
	PayStatus order_model.OrderPayStatus
	Remark    string `json:"Remark,omitempty"`
}

// 更新付费状态
func UpdatePayStatus(ctx context.Context, orderID, uid string, payStatus order_model.OrderPayStatus, remark string) error {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "UpdatePayStatus")
	r := &upsReq{
		OrderID:   orderID,
		UID:       uid,
		PayStatus: payStatus,
		Remark:    remark,
	}
	_, err := chain.Handle(ctx, r, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		r := req.(*upsReq)
		return nil, orderApi.UpdatePayStatus(ctx, r.OrderID, r.UID, r.PayStatus, r.Remark)
	})
	return err
}

type uosReq struct {
	OrderID string
	UID     string
	Extend  interface{} `json:"Extend,omitempty"`
	Status  order_model.OrderStatus
	Remark  string `json:"Remark,omitempty"`
}

// 更新订单状态和扩展数据
func UpdateOrderStatus(ctx context.Context, orderID, uid string, extend interface{}, status order_model.OrderStatus,
	remark string) error {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "UpdateOrderStatus")
	r := &uosReq{
		OrderID: orderID,
		UID:     uid,
		Extend:  extend,
		Status:  status,
		Remark:  remark,
	}
	_, err := chain.Handle(ctx, r, func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
		r := req.(*uosReq)
		return nil, orderApi.UpdateOrderStatus(ctx, r.OrderID, r.UID, r.Extend, r.Status, r.Remark)
	})
	return err
}

type genOIDReq struct {
	OrderType   order_model.OrderType
	UID         string
	UserOrderID string `json:"UserOrderID,omitempty"`
	ThirdPayOid string `json:"ThirdPayOid,omitempty"`
}
type genOIDRsp struct {
	OrderID string
}

// 根据用户订单号生成单号
func GenOIDByUserOID(ctx context.Context, orderType order_model.OrderType, uid, userOrderID string) (string, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "GenOIDByUserOID")
	r := &genOIDReq{
		OrderType:   orderType,
		UID:         uid,
		UserOrderID: userOrderID,
	}
	sp := &genOIDRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*genOIDReq)
		sp := rsp.(*genOIDRsp)
		oid, err := orderApi.GenOIDByUserOID(ctx, r.OrderType, r.UID, r.UserOrderID)
		sp.OrderID = oid
		return err
	})
	return sp.OrderID, err
}

// 根据第三方订单号生成单号
func GenOIDByThirdPayOID(ctx context.Context, orderType order_model.OrderType, uid, thirdPayOid string) (string, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "GenOIDByThirdPayOID")
	r := &genOIDReq{
		OrderType:   orderType,
		UID:         uid,
		ThirdPayOid: thirdPayOid,
	}
	sp := &genOIDRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*genOIDReq)
		sp := rsp.(*genOIDRsp)
		oid, err := orderApi.GenOIDByThirdPayOID(ctx, r.OrderType, r.UID, r.ThirdPayOid)
		sp.OrderID = oid
		return err
	})
	return sp.OrderID, err
}

// 生成订单号
func GenOID(ctx context.Context, orderType order_model.OrderType, uid string) (string, error) {
	ctx, chain := filter.GetClientFilter(ctx, clientType, clientName, "GenOID")
	r := &genOIDReq{
		OrderType: orderType,
		UID:       uid,
	}
	sp := &genOIDRsp{}
	err := chain.HandleInject(ctx, r, sp, func(ctx context.Context, req, rsp interface{}) error {
		r := req.(*genOIDReq)
		sp := rsp.(*genOIDRsp)
		oid, err := orderApi.GenOID(ctx, r.OrderType, r.UID)
		sp.OrderID = oid
		return err
	})
	return sp.OrderID, err
}
