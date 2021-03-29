package mq

import (
	"github.com/streadway/amqp"
)

type Conn struct {
	Channel *amqp.Channel
}

func GetConn(url string) (Conn, error) {
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
