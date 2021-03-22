package handlers

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
)

var a *Application

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

	a = NewApplication(dbc)

	return m.Run()
}
