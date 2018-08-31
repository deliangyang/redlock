package redlock

import (
	"github.com/go-redis/redis"
	"time"
	"github.com/pkg/errors"
	"strconv"
	"log"
	"fmt"
)

const entityStateDown = 0
const entityStateAlive = 1

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
	//go rds.watch()
	return rds
}

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
	fmt.Println(half)
	i := 0
	rds.availablePool = 0
	for ; i < half; i++ {
		rds.availablePool += 2 ^ ((s / length + i) % length)
	}
}

func (rds *redisPool) Lock(timeout int64) error {
	if rds.key == "" {
		errors.New("empty key")
	}

	operateTimeout := time.Now().Unix() + timeout
	var count int
	fmt.Println(rds.availablePool)
	for {
		count = 0
		for i, entity := range rds.pool {
			expireAt, _ := entity.redis.Get(rds.key).Int64()
			if true == entity.redis.SetNX(rds.key, operateTimeout, rds.expireTime).Val() {
				fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
				count++
			} else if expireAt < time.Now().Unix() {				// 过期
				fmt.Println(expireAt, " time out: ", expireAt - time.Now().Unix())
				errors.New("time out")
			}
			if i & rds.availablePool == i {
				fmt.Println("i => ", i)

			}
		}
		fmt.Println("xxxxxx:", count)
		if count == 3 {
			return nil
		}

		time.Sleep(time.Millisecond)
	}

}

/**
 * 释放锁
 */
func (rds *redisPool) UnLock() {
	fmt.Println("key unlocked!!!")
	for i, entity := range rds.pool {
		entity.redis.Del(rds.key)
		if i & rds.availablePool == i {
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

func (rds *redisPool) check()  {
	i := 0
	alive := 0
	length := len(rds.pool)
	for ; i < length; i++ {
		check := rds.pool[i].redis.Ping().String()
		if check == "Pong" {
			alive++
			rds.pool[i].setState(entityStateAlive)
		} else {
			rds.pool[i].setState(entityStateDown)
		}
	}
}