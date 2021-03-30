package mq

import (
	"fmt"

	"github.com/streadway/amqp"
)

type Config struct {
	User string
	Pass string
	Host string
	Port int
}

type Conn struct {
	Channel *amqp.Channel
}

func NewConnection(cfg Config) (Conn, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Pass, cfg.Host, cfg.Port)

	conn, err := amqp.Dial(url)
	if err != nil {
		return Conn{}, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return Conn{}, err
	}

	return Conn{Channel: ch}, nil
}
