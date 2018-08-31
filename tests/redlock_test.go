package tests

import (
	"testing"
	"github.com/go-redis/redis"
	"redlock"
	"time"
	"fmt"
)

/**
 * 测试案例，修改用户的昵称
 */
type User struct {
	name string
}

func (user *User) getName() string {
	return user.name
}

func (user *User) setName(name string)  {
	user.name = name
}

func TestRedLock(t *testing.T) {
	rds := redis.NewClient(&redis.Options{
		Addr: "www.ydl.com:6379",
	})

	l := redlock.New("test123", rds, time.Second * 30)
	user := &User{}

	i := 0
	for ; i < 4; i++ {
		go test(l, user, "Lucy")
		go test(l, user, "John")
	}

	time.Sleep(time.Second * 10)
}

func TestSum(t *testing.T) {
	rds := redis.NewClient(&redis.Options{
		Addr: "www.ydl.com:6379",
	})

	l := redlock.New("test123", rds, time.Second * 30)
	i := 0
	sum := 0
	go func() {
		for ; i < 20; i++ {
			l.Lock(30)
			sum += i
			l.UnLock()
		}
	}()

	j := 20
	go func() {
		for ; j < 40; j++ {
			l.Lock(30)
			sum += j
			l.UnLock()
		}
	}()

	Sum := 0
	x := 0
	for ; x < 40; x++ {
		Sum += x
	}
	fmt.Println("x sum: ", Sum)

	time.Sleep(time.Second)
	fmt.Println("sum: ", sum)
}

func test(l *redlock.Container, user *User, name string)  {
	l.Lock(30)
	fmt.Println("set name:", name)
	user.setName(name)
	time.Sleep(time.Second * 2)
	newName := user.getName()
	if newName != name {
		fmt.Println("the name must be ", name, ", but is ", newName)
	} else {
		fmt.Println("get name:", newName)
	}
	l.UnLock()
}
