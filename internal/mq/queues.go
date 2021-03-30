package mq

import (
	"github.com/streadway/amqp"
)

const (
	exchange  = "payments"
	deposits  = "deposits"
	withdraws = "withdraws"
)

func (conn Conn) DeclareQueues(concurrency int) (amqp.Queue, amqp.Queue, error) {
	err := conn.Channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	deposit, err := conn.Channel.QueueDeclare(deposits, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	err = conn.Channel.QueueBind(deposits, "dep", exchange, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	withdraw, err := conn.Channel.QueueDeclare(withdraws, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	err = conn.Channel.QueueBind(withdraws, "wit", exchange, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	prefetchCount := concurrency * 4
	err = conn.Channel.Qos(prefetchCount, 0, false)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	return deposit, withdraw, nil
}
