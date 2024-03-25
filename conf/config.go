package conf

import (
	"strings"

	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"
)

const OrderConfigKey = "order"

const (
	defDBName         = "order"
	defTableShardNums = 2

	defRedisName                     = "order"
	defOrderLockDBExpire             = 30
	defOrderUnlockDBLimitProcessTime = 10
	defOrderIDGeneratorShardNums     = 1000

	defMQType                = MQType_Pulsar
	defMQProducerName        = "order"
	defAllowMqCompensation   = false
	defCompensationDelayTime = 60
)

const (
	MQType_Pulsar = "pulsar"
)

var Conf = Config{
	DBName:         defDBName,
	TableShardNums: defTableShardNums,

	RedisName:                     defRedisName,
	OrderLockDBExpire:             defOrderLockDBExpire,
	OrderUnlockDBLimitProcessTime: defOrderUnlockDBLimitProcessTime,
	OrderIDGeneratorShardNums:     defOrderIDGeneratorShardNums,

	MQType:                defMQType,
	MQProducerName:        defMQProducerName,
	AllowMqCompensation:   defAllowMqCompensation,
	CompensationDelayTime: defCompensationDelayTime,
}

type Config struct {
	DBName         string // 数据库名
	TableShardNums uint32 // 表分片数量

	RedisName                     string // redis名
	OrderLockDBExpire             int    // 订单锁有效时间, 单位秒
	OrderUnlockDBLimitProcessTime int    // 订单处理在多少时间内完成才会主动解锁, 单位秒
	OrderIDGeneratorShardNums     int32  // 订单id生成器数量

	MQType                string // mq类型. 支持 pulsar
	MQProducerName        string // mq生产者名
	AllowMqCompensation   bool   // 是否允许mq补偿, 如果为false, 将不会启动mq补偿消费进程, 代码中的提交mq补偿会报错
	CompensationDelayTime int64  // mq补偿延迟时间, 单位秒
}

func (conf *Config) Check() {
	if conf.DBName == "" {
		conf.DBName = defDBName
	}
	if conf.TableShardNums < 1 {
		conf.TableShardNums = defTableShardNums
	}

	if conf.RedisName == "" {
		conf.RedisName = defRedisName
	}
	if conf.OrderLockDBExpire < 1 {
		conf.OrderLockDBExpire = defOrderLockDBExpire
	}
	if conf.OrderUnlockDBLimitProcessTime < 1 {
		conf.OrderUnlockDBLimitProcessTime = defOrderUnlockDBLimitProcessTime
	}
	if conf.OrderIDGeneratorShardNums < 1 {
		conf.OrderIDGeneratorShardNums = defOrderIDGeneratorShardNums
	}

	if conf.MQType == "" {
		conf.MQType = defMQType
	}
	conf.MQType = strings.ToLower(conf.MQType)
	switch conf.MQType {
	case MQType_Pulsar:
	default:
		logger.Log.Fatal("order config err. Unsupported MQType", zap.String("MQType", conf.MQType))
	}
	if conf.MQProducerName == "" {
		conf.MQProducerName = defMQProducerName
	}
	if conf.CompensationDelayTime < 1 {
		conf.CompensationDelayTime = defCompensationDelayTime
	}
}
