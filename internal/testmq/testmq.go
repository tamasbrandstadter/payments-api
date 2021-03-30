package testmq

import (
	log "github.com/sirupsen/logrus"
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
		log.Errorf("error connecting to test mq: %v", err)
		return mq.Conn{}, err
	}

	return conn, err
}
