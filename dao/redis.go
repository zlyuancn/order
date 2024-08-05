package dao

import (
	"context"
	"time"

	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/client"
)

/*
从redis设置一个锁

	op 操作类型
	expireTime 锁有效时间, 单位秒

return

	unlock 解锁操作.
	    limitProcessTime 限制过程时间, 单位秒. lock到unlock之间的时间如果超过了limitProcessTime的值则不会去解锁.
			limitProcessTime > 0 && limitProcessTime < expireTime 才会生效
*/
func SetRedisLock(ctx context.Context, key string, expireTime int) (
	unlock func(ctx context.Context, limitProcessTime int) (bool, error), ok bool, err error) {
	startTime := time.Now().UnixNano()
	ok, err = client.GetRedisClient().SetNX(ctx, key, 1, time.Duration(expireTime)*time.Second).Result()
	if err != nil {
		logger.Log.Error(ctx, "SetRedisLock error",
			zap.String("key", key),
			zap.Int("expireTime", expireTime),
			zap.Error(err),
		)
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	unlock = func(ctx context.Context, limitProcessTime int) (bool, error) {
		if limitProcessTime < 0 || limitProcessTime >= expireTime {
			limitProcessTime = expireTime / 2
		}

		nowTime := time.Now().UnixNano()
		if nowTime-startTime > int64(limitProcessTime)*int64(time.Second) {
			return false, nil
		}

		_, err := client.GetRedisClient().Del(ctx, key).Result()
		return true, err // 这里不判断是否真的删了key, 因为锁失效了也表示删除成功
	}
	return unlock, true, nil
}

func RedisIncrBy(ctx context.Context, key string, incr int64) (int64, error) {
	if incr == 0 {
		incr = 1
	}
	ret, err := client.GetRedisClient().IncrBy(ctx, key, incr).Result()
	return ret, err
}
