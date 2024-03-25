package order

import (
	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/core"
	"github.com/zly-app/zapp/handler"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/client"
	"github.com/zlyuancn/order/conf"
)

func init() {
	zapp.AddHandler(zapp.AfterInitializeHandler, func(app core.IApp, handlerType handler.HandlerType) {
		err := app.GetConfig().Parse(conf.OrderConfigKey, &conf.Conf, true)
		if err != nil {
			app.Fatal("parse order config err", zap.Error(err))
		}
		conf.Conf.Check()
		client.Init(app)
	})
	zapp.AddHandler(zapp.BeforeExitHandler, func(app core.IApp, handlerType handler.HandlerType) {
		client.Close(app)
	})
}
