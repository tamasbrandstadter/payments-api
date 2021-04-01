package balance

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

func TestHandleDeposit(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 15.5, "GBP", utc, utc, false)

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnRows(rows)

	balanceQuery := "UPDATE accounts SET balance=\\$1, modified_at=\\$2 WHERE id=\\$3;"

	mock.ExpectPrepare(balanceQuery).ExpectExec().WithArgs(16.5, sqlmock.AnyArg(), 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	auditQuery := "INSERT INTO transactions\\(account_id, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3\\) RETURNING id;"

	row := sqlmock.NewRows([]string{"id"}).AddRow(534)

	mock.ExpectBegin()
	mock.ExpectPrepare(auditQuery).ExpectQuery().WithArgs(accId, true, sqlmock.AnyArg()).WillReturnRows(row)
	mock.ExpectCommit()

	ok, err := handleDeposit(d, db, NewConn())
	if !ok || err != nil {
		t.Errorf("ok true and err nil was expected got: %v and %v", ok, err)
	}
}

func TestHandleDepositPayloadError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("invalid")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := handleDeposit(d, db, NewConn())
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid message payload, unable to parse", err.Error())
}

func TestHandleDepositAmountError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":-1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := handleDeposit(d, db, mq.Conn{})
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "balance operation amount can't be negative", err.Error())
}

func TestHandleDepositNotFoundError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	ok, err := handleDeposit(d, db, NewConn())
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "account id 1 is not found", err.Error())
}

func TestHandleDepositServerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(errors.New("test"))
	mock.ExpectRollback()

	ok, err := handleDeposit(d, db, NewConn())
	assert.False(t, ok)
	assert.Nil(t, err)
}

func TestHandleWithdraw(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 15.5, "GBP", utc, utc, false)

	balanceQuery := "UPDATE accounts SET balance=\\$1, modified_at=\\$2 WHERE id=\\$3;"

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnRows(rows)
	mock.ExpectPrepare(balanceQuery).ExpectExec().WithArgs(14.5, sqlmock.AnyArg(), 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	auditQuery := "INSERT INTO transactions\\(account_id, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3\\) RETURNING id;"

	row := sqlmock.NewRows([]string{"id"}).AddRow(534)

	mock.ExpectBegin()
	mock.ExpectPrepare(auditQuery).ExpectQuery().WithArgs(accId, true, sqlmock.AnyArg()).WillReturnRows(row)
	mock.ExpectCommit()

	ok, err := handleWithdraw(d, db, NewConn())
	if !ok || err != nil {
		t.Errorf("ok true and err nil was expected got: %v and %v", ok, err)
	}
}

func TestHandleWithdrawPayloadError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("invalid")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := handleWithdraw(d, db, NewConn())
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid message payload, unable to parse", err.Error())
}

func TestHandleWithdrawAmountError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":-1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := handleWithdraw(d, db, NewConn())
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "balance operation amount can't be negative", err.Error())
}

func TestHandleWithdrawNotFoundError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	ok, err := handleWithdraw(d, db, NewConn())
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "account id 1 is not found", err.Error())
}

func TestHandleWithdrawServerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(errors.New("test"))
	mock.ExpectRollback()

	ok, err := handleWithdraw(d, db, NewConn())
	assert.False(t, ok)
	assert.Nil(t, err)
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
