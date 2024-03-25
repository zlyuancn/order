package client

import (
	pulsar_producer "github.com/zly-app/component/pulsar-producer"
	"github.com/zly-app/component/redis"
	"github.com/zly-app/component/sqlx"
	"github.com/zly-app/zapp/core"

	"github.com/zlyuancn/order/conf"
)

var (
	sqlxCreator sqlx.ISqlx
	SqlxClient  sqlx.Client

	redisCreator redis.IRedisCreator
	RedisClient  redis.UniversalClient

	pulsarProducerCreator pulsar_producer.IPulsarProducerCreator
	PulsarProducer        pulsar_producer.IPulsarProducer
)

func Init(app core.IApp) {
	sqlxCreator = sqlx.NewSqlx(app)
	SqlxClient = sqlxCreator.GetSqlx(conf.Conf.DBName)

	redisCreator = redis.NewRedisCreator(app)
	RedisClient = redisCreator.GetRedis(conf.Conf.RedisName)

	if conf.Conf.AllowMqCompensation {
		switch conf.Conf.MQType {
		case conf.MQType_Pulsar:
			pulsarProducerCreator = pulsar_producer.NewProducerCreator(app)
			PulsarProducer = pulsarProducerCreator.GetPulsarProducer(conf.Conf.MQProducerName)
		}
	}
}
func Close(app core.IApp) {
	sqlxCreator.Close()
	redisCreator.Close()

	if conf.Conf.AllowMqCompensation {
		switch conf.Conf.MQType {
		case conf.MQType_Pulsar:
			pulsarProducerCreator.Close()
		}
	}
}
