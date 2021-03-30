package mq

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

const (
	exchange  = "payments"
	deposits  = "deposits"
	withdraws = "withdraws"
)

func (conn Conn) DeclareQueues() (amqp.Queue, amqp.Queue, error)  {
	err := conn.Channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	if err != nil {
		log.Fatal("declare exchange payments: ", err)
	}

	deposit, err := conn.Channel.QueueDeclare(deposits, true, false, false, false, nil)
	if err != nil {
		log.Fatal("declare queue deposits: ", err)
	}

	err = conn.Channel.QueueBind(deposits, "dep", exchange, false, nil)
	if err != nil {
		log.Fatal("bind queue deposits to exchange: ", err)
	}

	withdraw, err := conn.Channel.QueueDeclare(withdraws, true, false, false, false, nil)
	if err != nil {
		log.Fatal("declare queue deposits: ", err)
	}

	err = conn.Channel.QueueBind(withdraws, "wit", exchange, false, nil)
	if err != nil {
		log.Fatal("bind queue withdraws to exchange: ", err)
	}

	return deposit, withdraw, nil
}
