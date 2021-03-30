package handler

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewApplication(t *testing.T) {
	app := NewApplication(NewMockDb())

	assert.NotNil(t, app.handler)

	router, ok := app.handler.(*httprouter.Router)
	if !ok {
		t.Errorf("router expected type httprouter.Router, got %v", router)
	}
}

func NewMockDb() *sqlx.DB {
	db, _, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return sqlxDB
}
