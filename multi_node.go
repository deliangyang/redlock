package redlock

import (
	"github.com/go-redis/redis"
	"time"
	"github.com/pkg/errors"
	"strconv"
	"log"
)

const entityStateDown = 0
const entityStateAlive = 1

type redisEntity struct {
	redis *redis.Client
	state int32

}
type redisPool struct {
	key string
	alive int
	pool []redisEntity
	availablePool int
	expireTime time.Duration
}

type MultiNodeRedLock interface {
	Lock()
	UnLock()
}

func (entity *redisEntity) setState(state int32) {
	entity.state = state
}

func Multi(entities []redisEntity) redisPool {
	rds := redisPool{
		alive: 0,
		pool: entities,
	}
	//rds.check()
	//go rds.watch()
	return rds
}

func (rds *redisPool) setKey(key string) {
	rds.key = key
	s, err := strconv.Atoi(key)
	if err != nil {
		log.Fatal(err)
	}
	half := rds.alive / 2
	i := 0
	for ; i < half; i++ {
		rds.availablePool += 2 ^ ((s / rds.alive + 1) % len(rds.pool))
	}
}

func (rds *redisPool) Lock(timeout int64) error {

	half := rds.alive / 2

	if half <= 0 {
		errors.New("no alive redis node")
	}
	operateTimeout := time.Now().Unix() + timeout
	var count int

	var available int
	for i, entity := range rds.pool {
		if count >= half {
			break
		}
		if entity.state == entityStateAlive {
			available += 2 ^ i
		}
	}

	if available == 0 {
		errors.New("no alive redis node")
	}

	count = 0
	for {
		for i, entity := range rds.pool {
			if i & available == i {
				expireAt, _ := entity.redis.Get(rds.key).Int64()
				if true == entity.redis.SetNX(rds.key, operateTimeout, rds.expireTime).Val() {
					count++
				} else if expireAt < time.Now().Unix() {				// 过期
					errors.New("time out")
				}
			}
		}

		if count == 0 {
			return nil
		}

		time.Sleep(time.Millisecond * 100)
	}

}

func (rds *redisPool) UnLock() {

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
	rds.alive = alive
}