package order

import (
	"fmt"

	"github.com/zlyuancn/order/order_model"
)

var orderBusiness = map[order_model.OrderType]order_model.OrderBusiness{}

// 注册业务, 重复注册会panic
func (orderCli) RegistryOrderBusiness(t order_model.OrderType, ob order_model.OrderBusiness) {
	_, ok := orderBusiness[t]
	if ok {
		panic(fmt.Errorf("RegistryOrderBusiness repetition OrderType=%v", t))
	}
	orderBusiness[t] = ob
}

// 获取业务
func (orderCli) GetOrderBusiness(t order_model.OrderType) (order_model.OrderBusiness, bool) {
	ob, ok := orderBusiness[t]
	return ob, ok
}

// 注册业务, 重复注册会panic
func RegistryOrderBusiness(t order_model.OrderType, ob order_model.OrderBusiness) {
	orderApi.RegistryOrderBusiness(t, ob)
}

// 获取业务
func GetOrderBusiness(t order_model.OrderType) (order_model.OrderBusiness, bool) {
	return orderApi.GetOrderBusiness(t)
}
