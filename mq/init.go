package mq

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	pulsar_producer "github.com/zly-app/component/pulsar-producer"
	pulsar_consume "github.com/zly-app/service/pulsar-consume"
	"github.com/zly-app/zapp/core"
	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/conf"
	"github.com/zlyuancn/order/order_model"
)

var (
	pulsarProducerCreator pulsar_producer.IPulsarProducerCreator
	PulsarProducer        pulsar_producer.IPulsarProducer
)

type CompensationProcess func(ctx context.Context, oid, uid string) error

var defCompensationProcess CompensationProcess

func Init(app core.IApp, compensationProcess CompensationProcess) {
	if !conf.Conf.AllowMqCompensation {
		return
	}

	defCompensationProcess = compensationProcess
	switch conf.Conf.MQType {
	case conf.MQType_Pulsar:
		pulsarProducerCreator = pulsar_producer.NewProducerCreator(app)
		PulsarProducer = pulsarProducerCreator.GetPulsarProducer(conf.Conf.MQProducerName)
		pulsar_consume.RegistryHandler(conf.Conf.MQConsumeName, func(ctx context.Context, msg pulsar_consume.Message) error {
			return consumeProcess(ctx, msg.Payload(), msg.PublishTime())
		})
	}
}

func Close() {
	if !conf.Conf.AllowMqCompensation {
		return
	}

	switch conf.Conf.MQType {
	case conf.MQType_Pulsar:
		pulsarProducerCreator.Close()
	}
}

func Send(ctx context.Context, msg *order_model.OrderMqMsg) error {
	payload, err := sonic.Marshal(msg)
	if err != nil {
		return err
	}

	switch conf.Conf.MQType {
	case conf.MQType_Pulsar:
		msg := &pulsar_producer.ProducerMessage{
			Payload:      payload,
			DeliverAfter: time.Duration(conf.Conf.CompensationDelayTime) * time.Second,
		}
		_, err = PulsarProducer.Send(ctx, msg)
		return err
	}

	logger.Log.Error(ctx, "order config err. Unsupported MQType", zap.String("MQType", conf.Conf.MQType))
	return fmt.Errorf("order config err. Unsupported MQType: %v", conf.Conf.MQType)
}

func consumeProcess(ctx context.Context, payload []byte, msgProductionTime time.Time) error {
	orderMsg := order_model.OrderMqMsg{}
	err := sonic.Unmarshal(payload, &orderMsg)
	if err != nil {
		logger.Log.Error(ctx, "Order consumeProcess unmarshal msg err",
			zap.String("payload", string(payload)),
			zap.Error(err),
		)
		return nil // 无论如何重试也不可能成功了
	}

	if orderMsg.OrderID == "" {
		logger.Log.Error(ctx, "Order consumeProcess OrderID is empty",
			zap.Any("orderMsg", orderMsg),
		)
		return nil
	}

	// 开始推进
	err = defCompensationProcess(ctx, orderMsg.OrderID, orderMsg.Uid)
	if err == nil {
		return nil
	}

	logger.Log.Error(ctx, "Order consumeProcess err",
		zap.Any("orderMsg", orderMsg),
		zap.Error(err),
	)
	return err
}
