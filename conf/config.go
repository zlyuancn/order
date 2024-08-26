package conf

import (
	"strings"

	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"
)

const OrderConfigKey = "order"

const (
	defSqlxName       = "order"
	defTableShardNums = 2

	defRedisName                     = "order"
	defOrderLockDBExpire             = 30
	defOrderUnlockDBLimitProcessTime = 10
	defOrderLockKeyFormat            = "order:lock:op:<order_id>"
	defOrderSeqNoKeyFormat           = "order:seqno:<order_type>:<shard_num>"

	defMQType                = MQType_Pulsar
	defMQProducerName        = "order"
	defAllowMqCompensation   = false
	defCompensationDelayTime = 20
	defMQConsumeName         = "order"
)

const (
	MQType_Pulsar = "pulsar"
)

var Conf = Config{
	SqlxName:       defSqlxName,
	TableShardNums: defTableShardNums,

	RedisName:                     defRedisName,
	OrderLockDBExpire:             defOrderLockDBExpire,
	OrderUnlockDBLimitProcessTime: defOrderUnlockDBLimitProcessTime,
	OrderLockKeyFormat:            defOrderLockKeyFormat,
	OrderSeqNoKeyFormat:           defOrderSeqNoKeyFormat,

	MQType:                defMQType,
	MQProducerName:        defMQProducerName,
	AllowMqCompensation:   defAllowMqCompensation,
	CompensationDelayTime: defCompensationDelayTime,
	MQConsumeName:         defMQConsumeName,
}

type Config struct {
	SqlxName       string // sqlx组件名
	TableShardNums uint32 // 表分片数量

	RedisName                     string // redis组件名
	OrderLockDBExpire             int    // 订单锁有效时间, 单位秒
	OrderUnlockDBLimitProcessTime int    // 订单处理在多少时间内完成才会主动解锁, 单位秒
	OrderLockKeyFormat            string // 订单锁key格式化字符串
	OrderSeqNoKeyFormat           string // 生成订单序列号key格式化字符串

	MQType                string // mq类型. 支持 pulsar
	MQProducerName        string // mq生产者组件名
	AllowMqCompensation   bool   // 是否允许mq补偿, 如果为false, 将不会启动mq补偿消费进程, 代码中的提交mq补偿会报错, 且不会启动mq补偿消费者
	CompensationDelayTime int64  // mq补偿延迟时间, 单位秒
	MQConsumeName         string // mq消费者组件名
}

func (conf *Config) Check() {
	if conf.SqlxName == "" {
		conf.SqlxName = defSqlxName
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
	if conf.OrderLockKeyFormat == "" {
		conf.OrderLockKeyFormat = defOrderLockKeyFormat
	}
	if conf.OrderSeqNoKeyFormat == "" {
		conf.OrderSeqNoKeyFormat = defOrderSeqNoKeyFormat
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
	if conf.MQConsumeName == "" {
		conf.MQConsumeName = defMQConsumeName
	}
}
