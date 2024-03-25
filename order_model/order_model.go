package order_model

import (
	"context"
)

// 订单类型
type OrderType int

// 订单状态
type OrderStatus int

const (
	OrderStatus_Forwarding OrderStatus = 1 // 已创建/推进中
	OrderStatus_Finish     OrderStatus = 2 // 完成

	OrderStatus_BusinessCancelForward OrderStatus = 3 // 业务层取消推进
	OrderStatus_InsufficientBalance   OrderStatus = 4 // 余额不足
	OrderStatus_ReturnedBalance       OrderStatus = 5 // 已退回余额
	OrderStatus_UnableToAdvance       OrderStatus = 6 // 无法向前推进业务, 需要人工介入
)

// 订单使用的支付类型
type OrderPayType int

const (
	OrderPayType_None   OrderPayType = 0 // 无需支付
	OrderPayType_WeChat OrderPayType = 1 // 支付宝
	OrderPayType_Alipay OrderPayType = 2 // 微信
)

// 订单支付状态
type OrderPayStatus int

const (
	OrderPayStatus_None    OrderPayStatus = 0 // 未支付
	OrderPayStatus_Success OrderPayStatus = 0 // 支付完成
)

// 订单数据
type Order struct {
	OrderID   string    // 订单id
	OrderType OrderType // 订单类型

	PayType         OrderPayType   // 支付类型
	PayStatus       OrderPayStatus // 支付状态
	PayAmount       uint32         // 付费金额, 单位分
	ThirdPayOrderID string         // 第三方支付订单id

	Uid string // 用户唯一标识
}

// 订单在mq中的数据
type OrderMqMsg struct {
	OrderID string // 订单id
	Uid     string // uid, 主要用于确定订单数据在db哪个表上
}

// -----------------
//   callback
// -----------------

var _ OrderBusiness = (*OrderBusinessWrap)(nil)

type OrderBusinessWrap struct {
	// 返回扩展数据的结构
	OrderNewExtendStruct func(ctx context.Context) interface{}
	// 是否能推进, 订单系统会在 Forward 前调用这个方法, 返回err会让mq重试, 设置 cause 表示被业务层取消推进
	OrderCanForward func(ctx context.Context, order *Order, extend interface{}) (cause string, err error)
	// 交付, 订单系统会在付款成功后调用这个方法, 返回err会让mq重试
	OrderDelivery func(ctx context.Context, order *Order, extend interface{}) error
	// 推进订单异常结束状态回调, 订单无法继续推进(重试也不能推进)时会调用这个方法, 返回err会让mq重试
	OrderForwardAbnormalCallback func(ctx context.Context, order *Order, extend interface{}, status OrderStatus) error
	// 推进订单完成回调
	OrderForwardFinishCallback func(ctx context.Context, order *Order, extend interface{}) error
}

func (o *OrderBusinessWrap) NewExtendStruct(ctx context.Context) interface{} {
	if o.OrderNewExtendStruct != nil {
		return o.OrderNewExtendStruct(ctx)
	}
	return nil
}
func (o *OrderBusinessWrap) CanForward(ctx context.Context, order *Order, extend interface{}) (cause string, err error) {
	if o.OrderCanForward != nil {
		return o.OrderCanForward(ctx, order, extend)
	}
	return "", nil
}
func (o *OrderBusinessWrap) Delivery(ctx context.Context, order *Order, extend interface{}) error {
	if o.OrderDelivery != nil {
		return o.OrderDelivery(ctx, order, extend)
	}
	return nil
}
func (o *OrderBusinessWrap) ForwardAbnormalCallback(ctx context.Context, order *Order, extend interface{}, status OrderStatus) error {
	if o.OrderForwardAbnormalCallback != nil {
		return o.OrderForwardAbnormalCallback(ctx, order, extend, status)
	}
	return nil
}
func (o *OrderBusinessWrap) ForwardFinishCallback(ctx context.Context, order *Order, extend interface{}) error {
	if o.OrderForwardFinishCallback != nil {
		return o.OrderForwardFinishCallback(ctx, order, extend)
	}
	return nil
}

// 订单业务层
type OrderBusiness interface {
	// 返回扩展数据的结构
	NewExtendStruct(ctx context.Context) interface{}
	// 是否能推进, 订单系统会在 Forward 前调用这个方法, 返回err会让mq重试, 设置 cause 表示被业务层取消推进
	CanForward(ctx context.Context, order *Order, extend interface{}) (cause string, err error)
	// 交付, 订单系统会在付款成功后调用这个方法, 返回err会让mq重试
	Delivery(ctx context.Context, order *Order, extend interface{}) error
	// 推进订单异常结束状态回调, 订单无法继续推进(重试也不能推进)时会调用这个方法, 返回err会让mq重试
	ForwardAbnormalCallback(ctx context.Context, order *Order, extend interface{}, status OrderStatus) error
	// 推进订单完成回调
	ForwardFinishCallback(ctx context.Context, order *Order, extend interface{}) error
}
