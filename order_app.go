package order

import (
	"fmt"

	"github.com/zlyuancn/order/order_model"
)

var orderBusiness = map[order_model.OrderType]order_model.OrderBusiness{}

// 注册业务交付Handler, 重复注册会panic
func (orderCli) RegistryOrderBusiness(t order_model.OrderType, ob order_model.OrderBusiness) {
	_, ok := orderBusiness[t]
	if ok {
		panic(fmt.Errorf("RegistryOrderBusiness repetition OrderType=%v", t))
	}
	orderBusiness[t] = ob
}

func (orderCli) GetOrderBusiness(t order_model.OrderType) (order_model.OrderBusiness, bool) {
	ob, ok := orderBusiness[t]
	return ob, ok
}
