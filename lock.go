package redlock

import (
	"github.com/go-redis/redis"
	"time"
	"errors"
)

// 过期时间
const defaultExpireTime = time.Second * 100

// 轮训key时间间隔
const watchInterval = time.Millisecond * 100

type Container struct {
	key string
	redis *redis.Client
	expireTime time.Duration				// 持有锁超时时间
}

type RedLock interface {
	Lock(timeout int) error
	UnLock()
}

func New(key string, redis *redis.Client, expireTime time.Duration) *Container {
	if expireTime == 0 {
		expireTime = defaultExpireTime
	}
	return &Container{
		key: key,
		redis: redis,
		expireTime: expireTime,
	}
}

/**
 * 锁, 等待超时
 */
func (m *Container) Lock(timeout int64) error {
	now := time.Now()
	expireAt, _ := m.redis.Get(m.key).Int64()

	for {
		// key 不存在
		if true == m.redis.SetNX(m.key, now.Unix() + timeout, m.expireTime).Val() {
			return nil
		} else if expireAt < time.Now().Unix() {				// 过期
			errors.New("time out")
		}
		time.Sleep(watchInterval)
	}
}

/**
 * 释放锁
 */
func (m *Container) UnLock() {
	m.redis.Del(m.key)
}
