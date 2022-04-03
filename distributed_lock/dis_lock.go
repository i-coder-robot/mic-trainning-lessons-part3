package distributed_lock

import (
	"fmt"
	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"sync"
	"time"
)

func RedisLock(wg *sync.WaitGroup) {
	defer wg.Done()
	//redisAddr := fmt.Sprintf("%s:%d",internal.AppConf.RedisConfig.Host,
	//	internal.AppConf.RedisConfig.Port)
	redisAddr := "192.168.0.107:6379"
	client := goredislib.NewClient(&goredislib.Options{
		Addr: redisAddr,
	})
	pool := goredis.NewPool(client)

	rs := redsync.New(pool)

	mutexname := "product@1"
	mutex := rs.NewMutex(mutexname)

	fmt.Println("Lock()...")
	err := mutex.Lock()
	if err != nil {
		panic(err)
	}

	//业务逻辑
	fmt.Println("Get Lock!!!")
	time.Sleep(time.Second * 8)

	fmt.Println("Unlock()")
	ok, err := mutex.Unlock()
	if !ok || err != nil {
		panic(err)
	}
	fmt.Println("Released Lock!!!")
}
