package order

import (
	"errors"
)

var (
	// 订单不存在
	OrderNotFoundErr = errors.New("order not found")
	// 订单业务取消推进
	OrderBusinessCancelForwardErr = errors.New("order business cancel forward")
)
