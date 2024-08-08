package order

import (
	pulsar_consume "github.com/zly-app/service/pulsar-consume"
	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/core"

	"github.com/zlyuancn/order/conf"
)

func WithService() zapp.Option {
	return zapp.WithCustomEnableService(func(app core.IApp, services []core.ServiceType) []core.ServiceType {
		if !conf.Conf.AllowMqCompensation {
			return services
		}

		switch conf.Conf.MQType {
		case conf.MQType_Pulsar:
			return addService(services, pulsar_consume.DefaultServiceType)
		}
		return services
	})
}

func addService(services []core.ServiceType, t core.ServiceType) []core.ServiceType {
	for i := range services {
		if services[i] == t {
			return services
		}
	}
	services = append(services, t)
	return services
}
