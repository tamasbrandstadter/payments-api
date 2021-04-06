package handler

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/balance"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testcache"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

type TestApp struct {
	Handler *Application
	DB      *sqlx.DB
	Conn    *mq.Conn
	Tc      *balance.TransactionConsumer
}

var a *TestApp

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	db, err := testdb.Open()
	if err != nil {
		log.WithError(err).Info("create test database connection")
		return 1
	}
	defer db.Close()

	conn, err := testmq.Open()
	if err != nil {
		log.WithError(err).Info("create test mq connection")
		return 1
	}

	deposit, withdraw, transfer, err := conn.DeclareQueues(5)
	tc := &balance.TransactionConsumer{
		Deposit:     deposit,
		Withdraw:    withdraw,
		Transfer:    transfer,
		Concurrency: 5,
	}

	redis, err := testcache.OpenConnection()
	if err != nil {
		log.WithError(err).Info("create test cache")
		return 1
	}

	a = &TestApp{
		Handler: NewApplication(db, redis),
		DB:      db,
		Conn:    conn,
		Tc:      tc,
	}

	code := m.Run()

	deleteRecords()

	return code
}

func deleteRecords() {
	a.DB.Exec("DELETE FROM transactions")
	a.DB.Exec("ALTER SEQUENCE transactions_id_seq RESTART WITH 1")

	a.DB.Exec("DELETE FROM accounts")
	a.DB.Exec("ALTER SEQUENCE accounts_id_seq RESTART WITH 1")

	a.DB.Exec("DELETE FROM customers")
	a.DB.Exec("ALTER SEQUENCE customers_id_seq RESTART WITH 1")
}
