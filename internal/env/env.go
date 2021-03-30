package env

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"time"
)

type Cfg struct {
	DBUser string `envconfig:"DB_USER"`
	DBPass string `envconfig:"DB_PASSWORD"`
	DBName string `envconfig:"DB_NAME"`
	DBPort int    `envconfig:"DB_PORT" default:"5432"`

	MQUser string `envconfig:"MQ_USER"`
	MQPass string `envconfig:"MQ_PASSWORD"`
	MQHost string `envconfig:"MQ_HOST"`
	MQPort int    `envconfig:"DB_PORT" default:"5672"`

	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
}

func GetEnvCfg() Cfg {
	var cfg Cfg

	if err := envconfig.Process("APP", &cfg); err != nil {
		log.Fatal("parse environment variables: ", err)
	}

	return cfg
}
