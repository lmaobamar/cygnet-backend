package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(addr string) *redis.Client {
	loadStart := time.Now()
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis: can't connect to %s: %v", addr, err)
	}
	log.Printf("rdb in %v", time.Since(loadStart))
	return rdb
}
