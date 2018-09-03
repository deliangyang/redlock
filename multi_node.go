package redlock

import (
	"github.com/go-redis/redis"
	"time"
	"fmt"
	"errors"
	"strconv"
	"log"
)

const entityStateDown = 0			// redis 状态：关闭
const entityStateAlive = 1			// redis 状态：存活

type redisEntity struct {
	redis *redis.Client
	state int32

}
type redisPool struct {
	key string
	pool []redisEntity
	availablePool int
	expireTime time.Duration
}

func (entity *redisEntity) setState(state int32) {
	entity.state = state
}

func Multi(expireTime time.Duration, config ...string) redisPool {
	length := len(config)
	entities := make([]redisEntity, length)
	for i, conf := range config {
		entities[i] = redisEntity{
			redis: redis.NewClient(&redis.Options{
				Addr: conf,
			}),
			state: entityStateAlive,
		}
	}
	rds := redisPool{
		pool: entities,
		expireTime: expireTime,
	}
	return rds
}

/**
 * 根据key分布到不同的一半实例上
 */
func (rds *redisPool) SetKey(key string) {
	rds.key = key
	s, err := strconv.Atoi(key)
	if err != nil {
		log.Fatal(err)
	}
	length := len(rds.pool)
	half := length / 2
	if length - half * 2 == 1 {
		half++
	}
	i := 0
	rds.availablePool = 0
	for ; i < half; i++ {
		rds.availablePool += 2 ^ ((s / length + i) % length)
	}
}

/**
 * 加锁
 */
func (rds *redisPool) Lock(timeout int64) error {
	if rds.key == "" {
		errors.New("empty key")
	}
	length := len(rds.pool)
	half := length / 2

	operateTimeout := time.Now().Unix() + timeout
	var count int
	for {
		count = 0
		for i, entity := range rds.pool {
			if i & rds.availablePool == i {
				expireAt, _ := entity.redis.Get(rds.key).Int64()
				if true == entity.redis.SetNX(rds.key, operateTimeout, rds.expireTime).Val() {
					count++
				} else if expireAt < time.Now().Unix() {				// 过期
					fmt.Println(expireAt, " time out: ", expireAt - time.Now().Unix())
					errors.New("time out")
				}
				if count == half {
					return nil
				}
			}
		}

		time.Sleep(time.Millisecond)
	}

}

/**
 * 释放锁
 */
func (rds *redisPool) UnLock() {
	for i, entity := range rds.pool {
		if i & rds.availablePool == i {
			entity.redis.Del(rds.key)
		}
	}
}

/**
 * 监控redis集群活跃的机器, 500ms轮训
 */
func (rds *redisPool) watch() {
	for {
		rds.check()
		time.Sleep(time.Millisecond * 500)
	}
}

/**
 *　检查机器
 */
func (rds *redisPool) check()  {
	i := 0
	alive := 0
	length := len(rds.pool)
	for ; i < length; i++ {
		check := rds.pool[i].redis.Ping().String()
		fmt.Println(check)
		if check == "Pong" {
			alive++
			rds.pool[i].setState(entityStateAlive)
		} else {
			rds.pool[i].setState(entityStateDown)
		}
	}
}