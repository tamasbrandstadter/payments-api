package testmq

import (
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	user     = "guest"
	password = "guest"
	host     = "localhost"
	port     = 5672
	queue    = "test-queue"
	exchange = "balance-notifications"
	topic    = "topic"
)

func Open() (*mq.Conn, error) {
	cfg := mq.Config{
		User:         user,
		Pass:         password,
		Host:         host,
		Port:         port,
		Concurrency:  5,
		MaxReconnect: 5,
	}

	conn, err := mq.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	err = conn.Channel.ExchangeDeclare(exchange, topic, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	q, err := conn.Channel.QueueDeclare(queue, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = conn.Channel.QueueBind(q.Name, "notif", exchange, false, nil)
	if err != nil {
		return nil, err
	}

	return conn, err
}
