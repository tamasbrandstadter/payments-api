package balance

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Rhymond/go-money"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/cache"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testcache"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

func TestDeposit(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":10}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance_in_decimal", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 155, "GBP", utc, utc, false)

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnRows(rows)

	balanceQuery := "UPDATE accounts SET balance_in_decimal=\\$1, modified_at=\\$2 WHERE id=\\$3;"

	mock.ExpectPrepare(balanceQuery).ExpectExec().WithArgs(165, sqlmock.AnyArg(), 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	auditQuery := "INSERT INTO transactions\\(from_id, to_id, transaction_type, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	row := sqlmock.NewRows([]string{"id"}).AddRow(534)

	mock.ExpectBegin()
	mock.ExpectPrepare(auditQuery).ExpectQuery().WithArgs(accId, 0, "deposit", true, sqlmock.AnyArg()).WillReturnRows(row)
	mock.ExpectCommit()

	ok, err := deposit(d, db, NewConn(), NewCache())

	if !ok || err != nil {
		t.Errorf("test handle deposit failed, ok true and err nil were expected got: %v and %v", ok, err)
	}
}

func TestDepositPayloadError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("invalid")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := deposit(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid message payload, unable to parse", err.Error())
}

func TestDepositAmountError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":-100}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := deposit(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "balance operation amount can't be negative", err.Error())
}

func TestDepositNotFoundError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":100}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	ok, err := deposit(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "account id 1 is not found", err.Error())
}

func TestDepositServerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":100}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(errors.New("test"))
	mock.ExpectRollback()

	ok, err := deposit(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Nil(t, err)
}

func TestWithdraw(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":10}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance_in_decimal", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 155, "GBP", utc, utc, false)

	balanceQuery := "UPDATE accounts SET balance_in_decimal=\\$1, modified_at=\\$2 WHERE id=\\$3;"

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnRows(rows)
	mock.ExpectPrepare(balanceQuery).ExpectExec().WithArgs(145, sqlmock.AnyArg(), 1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	auditQuery := "INSERT INTO transactions\\(from_id, to_id, transaction_type, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	row := sqlmock.NewRows([]string{"id"}).AddRow(534)

	mock.ExpectBegin()
	mock.ExpectPrepare(auditQuery).ExpectQuery().WithArgs(accId, 0, "withdraw", true, sqlmock.AnyArg()).WillReturnRows(row)
	mock.ExpectCommit()

	ok, err := withdraw(d, db, NewConn(), NewCache())

	if !ok || err != nil {
		t.Errorf("test handle withdraw failed, ok true and err nil were expected got: %v and %v", ok, err)
	}
}

func TestWithdrawPayloadError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("invalid")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := withdraw(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid message payload, unable to parse", err.Error())
}

func TestWithdrawAmountError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":-1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := withdraw(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "balance operation amount can't be negative", err.Error())
}

func TestWithdrawNotFoundError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	ok, err := withdraw(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "account id 1 is not found", err.Error())
}

func TestWithdrawServerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"id\":1,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(accId).WillReturnError(errors.New("test"))
	mock.ExpectRollback()

	ok, err := withdraw(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Nil(t, err)
}

func TestTransfer(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	redis := NewCache()

	msg := []byte("{\"from\":1,\"to\":2,\"amount\":10}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, balance_in_decimal, currency FROM accounts WHERE id=\\$1 OR id=\\$2"

	from := 1
	to := 2

	rows := sqlmock.NewRows([]string{"id", "balance_in_decimal", "currency"}).
		AddRow(1, 155, "EUR").AddRow(2, 56, "EUR")

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(from, to).WillReturnRows(rows)

	updateQuery := "UPDATE accounts as a SET balance_in_decimal = a2.balance_in_decimal, modified_at = a2.modified_at FROM " +
		"\\(values \\(\\$1::integer, \\$2::decimal, \\$3::timestamp\\), \\(\\$4::integer, \\$5::decimal, \\$6::timestamp\\)\\) " +
		"as a2\\(id, balance_in_decimal, modified_at\\) WHERE a2.id = a.id;"

	mock.ExpectPrepare(updateQuery).ExpectExec().WithArgs(from, 145, sqlmock.AnyArg(), to, 66, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(2, 2))
	mock.ExpectCommit()

	auditQuery := "INSERT INTO transactions\\(from_id, to_id, transaction_type, ack, created_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	row := sqlmock.NewRows([]string{"id"}).AddRow(534)

	mock.ExpectBegin()
	mock.ExpectPrepare(auditQuery).ExpectQuery().WithArgs(from, to, "transfer", true, sqlmock.AnyArg()).WillReturnRows(row)
	mock.ExpectCommit()

	ok, err := transfer(d, db, NewConn(), redis)

	if !ok || err != nil {
		t.Errorf("test handle transfer failed, ok true and err nil were expected got: %v and %v", ok, err)
	}

	var b []byte
	if err = redis.Balances.Get(context.Background(), "1", &b); err != nil {
		t.Errorf("test handle transfer failed, can't get balance from cache, error: %v", err)
	}

	var m money.Money
	_ = m.UnmarshalJSON(b)

	assert.Equal(t, int64(145), m.Amount())
	assert.Equal(t, "EUR", m.Currency().Code)

	if err = redis.Balances.Get(context.Background(), "2", &b); err != nil {
		t.Errorf("test handle transfer failed, can't get balance from cache, error: %v", err)
	}

	_ = m.UnmarshalJSON(b)

	assert.Equal(t, int64(66), m.Amount())
	assert.Equal(t, "EUR", m.Currency().Code)
}

func TestTransferPayloadError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("invalid")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := transfer(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid message payload, unable to parse", err.Error())
}

func TestTransferAmountError(t *testing.T) {
	db, _ := NewMockDb()
	defer db.Close()

	msg := []byte("{\"from\":1,\"to\":2,\"amount\":-1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	ok, err := transfer(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "balance operation amount can't be negative", err.Error())
}

func TestTransferAccountNotFoundError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"from\":1,\"to\":2,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, balance_in_decimal, currency FROM accounts WHERE id=\\$1 OR id=\\$2"

	from := 1
	to := 2

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(from, to).WillReturnError(account.InvalidAccountsError)
	mock.ExpectRollback()

	ok, err := transfer(d, db, NewConn(), NewCache())

	assert.False(t, ok)
	assert.Error(t, err)
	assert.Equal(t, "invalid transfer, account ids are not found", err.Error())
}

func TestTransferServerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	msg := []byte("{\"from\":1,\"to\":2,\"amount\":1}")

	d := amqp.Delivery{
		ContentType: "application/json",
		Body:        msg,
	}

	selectQuery := "SELECT id, balance_in_decimal, currency FROM accounts WHERE id=\\$1 OR id=\\$2"

	from := 1
	to := 2

	mock.ExpectBegin()
	mock.ExpectQuery(selectQuery).WithArgs(from, to).WillReturnError(errors.New("test"))
	mock.ExpectRollback()

	ok, err := transfer(d, db, NewConn(), NewCache())

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

func NewConn() *mq.Conn {
	conn, err := testmq.Open()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a test mq connection", err)
	}

	return conn
}

func NewCache() *cache.Redis {
	redis, err := testcache.OpenConnection()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when creating cache", err)
	}

	return redis
}
