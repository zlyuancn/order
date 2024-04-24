package order

import (
	"context"

	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/config"
	"github.com/zly-app/zapp/core"
	"github.com/zly-app/zapp/handler"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/client"
	"github.com/zlyuancn/order/conf"
	"github.com/zlyuancn/order/mq"
)

func init() {
	config.RegistryApolloNeedParseNamespace(conf.OrderConfigKey)

	zapp.AddHandler(zapp.BeforeInitializeHandler, func(app core.IApp, handlerType handler.HandlerType) {
		err := app.GetConfig().Parse(conf.OrderConfigKey, &conf.Conf, true)
		if err != nil {
			app.Fatal("parse order config err", zap.Error(err))
		}
		conf.Conf.Check()
	})
	zapp.AddHandler(zapp.AfterInitializeHandler, func(app core.IApp, handlerType handler.HandlerType) {
		client.Init(app)
		mq.Init(app, func(ctx context.Context, oid, uid string) error {
			_, _, err := Order.forwardOrderID(ctx, oid, uid, true)
			if err == OrderBusinessCancelForwardErr {
				return nil
			}
			return err
		})
	})
	zapp.AddHandler(zapp.AfterExitHandler, func(app core.IApp, handlerType handler.HandlerType) {
		mq.Close()
		client.Close()
	})
}
