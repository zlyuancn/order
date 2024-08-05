package client

import (
	pulsar_producer "github.com/zly-app/component/pulsar-producer"
	"github.com/zly-app/component/redis"
	"github.com/zly-app/component/sqlx"

	"github.com/zlyuancn/order/conf"
)

func GetSqlxClient() sqlx.Client {
	return sqlx.GetClient(conf.Conf.SqlxName)
}

func GetRedisClient() redis.UniversalClient {
	return redis.GetClient(conf.Conf.RedisName)
}

func GetPulsarProducer() pulsar_producer.Client {
	return pulsar_producer.GetClient(conf.Conf.MQProducerName)
}
