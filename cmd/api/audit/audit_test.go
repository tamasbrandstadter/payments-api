package audit

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

func TestSaveAuditRecord(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO transactions\\(from_id, to_id, transaction_type, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	rows := sqlmock.NewRows([]string{"id"}).AddRow(11)

	mock.ExpectBegin()
	mock.ExpectPrepare(query).ExpectQuery().WithArgs(1, 2, "transfer", true, sqlmock.AnyArg()).WillReturnRows(rows)
	mock.ExpectCommit()

	err := SaveAuditRecord(db, 1, 2, 2, NewConn())
	if err != nil {
		t.Errorf("test save audit record failed, expected err nil, got: %v", err)
	}
}

func TestSaveAuditRecordError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO transactions\\(from_id, to_id, transaction_type, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	mock.ExpectBegin()
	mock.ExpectPrepare(query).ExpectQuery().WithArgs(1, 2, "transfer", true, sqlmock.AnyArg()).WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	err := SaveAuditRecord(db, 1, 2, 2, NewConn())

	assert.Error(t, err)
}

func NewMockDb() (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return sqlxDB, mock
}

func NewConn() mq.Conn {
	conn, err := testmq.Open()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a test mq connection", err)
	}

	return conn
}
