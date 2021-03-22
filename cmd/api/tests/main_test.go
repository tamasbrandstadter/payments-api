package tests

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/handlers"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
)

var a *handlers.Application

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

	a = handlers.NewApplication(dbc)

	return m.Run()
}
