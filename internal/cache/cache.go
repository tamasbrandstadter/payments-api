package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Host string
	Pass string
	Port int
}

type Redis struct {
	Client   *redis.Ring
	Balances *cache.Cache
}

func NewConnection(cfg Config) *Redis {
	log.Info("connecting to redis")

	r := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		},
		HeartbeatFrequency: 10 * time.Second,
		Password:           cfg.Pass,
		MaxRetries:         3,
		MaxRetryBackoff:    3 * time.Second,
		ReadTimeout:        1 * time.Second,
		WriteTimeout:       1 * time.Second,
		PoolSize:           10,
		MinIdleConns:       1,
	})

	log.Info("verifying redis connection")

	err := r.Ping(context.Background()).Err()
	if err != nil {
		log.Errorf("failed to ping redis, error: %v", err)
		return nil
	}

	log.Info("verified redis connection")

	b := cache.New(&cache.Options{
		Redis:      r,
		LocalCache: cache.NewTinyLFU(1000, time.Hour),
	})

	log.Info("created balances cache")

	return &Redis{
		Client:   r,
		Balances: b,
	}
}
