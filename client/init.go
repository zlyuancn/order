package client

import (
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
)

func Init(app core.IApp) {
	sqlxCreator = sqlx.NewSqlx(app)
	SqlxClient = sqlxCreator.GetSqlx(conf.Conf.DBName)

	redisCreator = redis.NewRedisCreator(app)
	RedisClient = redisCreator.GetRedis(conf.Conf.RedisName)
}
func Close() {
	sqlxCreator.Close()
	redisCreator.Close()
}
