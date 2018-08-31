package tests

import (
	"testing"
	"redlock"
	"time"
	"fmt"
)

func TestMultiNode(t *testing.T) {
	pool := redlock.Multi(time.Second * 300, "www.ydl.com:6379", "www.ydl.com:6380", "www.ydl.com:6381")

	pool.SetKey("1232344")
	i := 0
	sum := 0
	go func() {
		for ; i < 20; i++ {
			pool.Lock(30)
			sum += i
			pool.UnLock()
		}
	}()

	j := 20
	go func() {
		for ; j < 40; j++ {
			pool.Lock(30)
			sum += j
			pool.UnLock()
		}
	}()

	Sum := 0
	x := 0
	for ; x < 40; x++ {
		Sum += x
	}
	fmt.Println("x sum: ", Sum)

	time.Sleep(time.Second * 10)
	fmt.Println("sum: ", sum)
}