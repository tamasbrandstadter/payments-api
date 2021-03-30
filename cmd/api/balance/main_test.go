package balance

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

type TestApp struct {
	DB   *sqlx.DB
	Conn mq.Conn
	Tc   TransactionConsumer
}

var a *TestApp

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	dbc, err := testdb.Open()
	if err != nil {
		log.WithError(err).Info("create test database connection")
		return 1
	}
	defer dbc.Close()

	conn, err := testmq.Open()
	if err != nil {
		log.WithError(err).Info("create test mq connection")
		return 1
	}

	deposit, withdraw, err := conn.DeclareQueues(5)
	tc := TransactionConsumer{
		Deposit:     deposit,
		Withdraw:    withdraw,
		Concurrency: 5,
	}

	a = &TestApp{
		DB:   dbc,
		Conn: conn,
		Tc:  tc,
	}

	return m.Run()
}
