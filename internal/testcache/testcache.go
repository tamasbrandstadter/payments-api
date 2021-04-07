package testcache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	c "github.com/tamasbrandstadter/payments-api/internal/cache"
)

const (
	host = "redis"
	pass = "securepass"
	port = 6379
)

func OpenConnection() (*c.Redis, error) {
	r := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": fmt.Sprintf("%s:%d", host, port),
		},
		Password: pass,
	})

	err := r.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	b := cache.New(&cache.Options{
		Redis:      r,
		LocalCache: cache.NewTinyLFU(1000, time.Hour),
	})

	return &c.Redis{
		Client:   r,
		Balances: b,
	}, nil
}
