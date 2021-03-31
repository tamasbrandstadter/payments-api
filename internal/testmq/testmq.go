package testmq

import (
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	user = "guest"

	password = "guest"

	host = "localhost"

	port = 5672
)

func Open() (mq.Conn, error) {
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
		return mq.Conn{}, err
	}

	return conn, err
}
