package redis_client

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
)

func getRedisAddr() []string {
	raw_addr := os.Getenv("REDIS_ADDR")
	return strings.Split(raw_addr, ",")
}

func GetRedisClients() []*redis.Client {
	addr_arr := getRedisAddr()
	fmt.Fprintf(os.Stderr, "redis address is %v\n", addr_arr)
	rdb_arr := make([]*redis.Client, len(addr_arr))
	for i := 0; i < len(addr_arr); i++ {
		rdb_arr[i] = redis.NewClient(&redis.Options{
			Addr:            addr_arr[i],
			Password:        "", // no password set
			DB:              0,  // use default DB
			MaxRetries:      10,
			MaxRetryBackoff: time.Duration(4096) * time.Millisecond,
			ReadTimeout:     time.Duration(5) * time.Second,
		})
	}
	return rdb_arr
}
