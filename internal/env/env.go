package env

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Cfg struct {
	DBUser string `envconfig:"DB_USER"`
	DBPass string `envconfig:"DB_PASSWORD"`
	DBName string `envconfig:"DB_NAME"`
	DBPort int    `envconfig:"DB_PORT" default:"5432"`

	MQUser         string `envconfig:"MQ_USER"`
	MQPass         string `envconfig:"MQ_PASSWORD"`
	MQHost         string `envconfig:"MQ_HOST"`
	MQPort         int    `envconfig:"MQ_PORT" default:"5672"`
	MQConcurrency  int    `envconfig:"MQ_CONCURRENCY" default:"5"`
	MQMaxReconnect int    `envconfig:"MQ_MAXRECONNECT" default:"5"`

	CacheHost string `envconfig:"CACHE_HOST"`
	CachePass string `envconfig:"CACHE_PASSWORD"`
	CachePort int    `envconfig:"CACHE_PORT" default:"6379"`

	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
}

func GetEnvCfg() (*Cfg, error) {
	var cfg Cfg

	if err := envconfig.Process("APP", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
