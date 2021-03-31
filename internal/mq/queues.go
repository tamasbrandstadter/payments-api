package mq

import (
	"github.com/streadway/amqp"
)

const (
	paymentsExchangeName = "payments"
	depositQueueName     = "deposits"
	withdrawQueueName    = "withdraws"
	kind                 = "topic"
	depositRouteKey      = "dep"
	withdrawRouteKey     = "wit"
)

func (conn Conn) DeclareQueues(concurrency int) (amqp.Queue, amqp.Queue, error) {
	err := conn.Channel.ExchangeDeclare(paymentsExchangeName, kind, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	deposit, err := conn.Channel.QueueDeclare(depositQueueName, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	err = conn.Channel.QueueBind(depositQueueName, depositRouteKey, paymentsExchangeName, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	withdraw, err := conn.Channel.QueueDeclare(withdrawQueueName, true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, amqp.Queue{}, err
	}

	err = conn.Channel.QueueBind(withdrawQueueName, withdrawRouteKey, paymentsExchangeName, false, nil)
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
