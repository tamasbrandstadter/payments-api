package mq

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Config struct {
	User         string
	Pass         string
	Host         string
	Port         int
	Concurrency  int
	MaxReconnect int
}

type Conn struct {
	Channel *amqp.Channel
}

func NewConnection(cfg Config) (*Conn, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Pass, cfg.Host, cfg.Port)

	log.Info("connecting to mq")

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	log.Info("connected to mq, opening channel")

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	log.Info("opened channel")
	return &Conn{Channel: ch}, nil
}
