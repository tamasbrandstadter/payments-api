package account

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var customerId = 22

func TestSelectAll(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "SELECT \\* FROM accounts;"

	utc := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(11, 22, 999.0, "EUR", utc, utc, false)

	mock.ExpectQuery(query).WillReturnRows(rows)

	accounts, err := SelectAll(db)

	assert.NoError(t, err)
	assert.Len(t, accounts, 1)

	actualAcc := accounts[0]
	assert.Equal(t, 11, actualAcc.ID)
	assert.Equal(t, 22, actualAcc.CustomerID)
	assert.Equal(t, 999.0, actualAcc.Balance)
	assert.Equal(t, Currency("EUR"), actualAcc.Currency)
	assert.NotNil(t, actualAcc.CreatedAt)
	assert.NotNil(t, actualAcc.ModifiedAt)
	assert.False(t, actualAcc.Frozen)
}

func TestSelectAllError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "SELECT \\* FROM accounts;"

	mock.ExpectQuery(query).WillReturnError(errors.New("sql: statement is closed"))

	accounts, err := SelectAll(db)

	assert.Nil(t, accounts)
	assert.Error(t, err)
}

func TestSelectById(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 232.4, "GBP", utc, utc, true)

	mock.ExpectPrepare(query).ExpectQuery().WithArgs(1).WillReturnRows(rows)

	actualAcc, err := SelectById(db, accId)

	assert.NoError(t, err)
	assert.NotNil(t, actualAcc)
	assert.Equal(t, accId, actualAcc.ID)
	assert.Equal(t, 11, actualAcc.CustomerID)
	assert.Equal(t, 232.4, actualAcc.Balance)
	assert.Equal(t, Currency("GBP"), actualAcc.Currency)
	assert.NotNil(t, actualAcc.CreatedAt)
	assert.NotNil(t, actualAcc.ModifiedAt)
	assert.True(t, actualAcc.Frozen)
}

func TestSelectByIdError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	mock.ExpectPrepare(query).ExpectQuery().WithArgs(1).WillReturnError(sql.ErrNoRows)

	_, err := SelectById(db, 1)

	assert.Error(t, err)
}

func TestCreate(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO accounts\\(customer_id, balance, currency, created_at, modified_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	rows := sqlmock.NewRows([]string{"id"}).AddRow(11)

	request := AccCreationRequest{
		FirstName:      "first",
		LastName:       "last",
		Email:          "first@last.com",
		InitialBalance: 111.0,
		Currency:       Currency("EUR"),
	}

	mock.ExpectPrepare(query).ExpectQuery().WithArgs(customerId, request.InitialBalance, request.Currency, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	actualAcc, err := Create(db, customerId, request)
	if err != nil {
		t.Errorf("account creation test failed err expected nil but got: %v:", err)
	}

	assert.NotNil(t, actualAcc)
	assert.Equal(t, 11, actualAcc.ID)
	assert.Equal(t, customerId, actualAcc.CustomerID)
	assert.Equal(t, request.InitialBalance, actualAcc.Balance)
	assert.Equal(t, request.Currency, actualAcc.Currency)
	assert.False(t, actualAcc.Frozen)
	assert.NotNil(t, actualAcc.CreatedAt)
	assert.NotNil(t, actualAcc.ModifiedAt)
}

func TestCreateError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO accounts\\(customer_id, balance, currency, created_at, modified_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	request := AccCreationRequest{
		FirstName:      "first",
		LastName:       "last",
		Email:          "first@last.com",
		InitialBalance: 111.0,
		Currency:       Currency("EUR"),
	}

	mock.ExpectPrepare(query).ExpectQuery().WithArgs(customerId, request.InitialBalance, request.Currency, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrTxDone)

	_, err := Create(db, customerId, request)

	assert.Error(t, err)
}

func TestDelete(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 232.4, "GBP", utc, utc, true)

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnRows(rows)

	deleteQuery := "DELETE FROM accounts WHERE id=\\$1;"

	mock.ExpectExec(deleteQuery).WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))

	err := Delete(db, accId)
	if err != nil {
		t.Errorf("account deletion test failed err expected nil but got: %v:", err)
	}
}

func TestDeleteErrorInSelect(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnError(sql.ErrNoRows)

	err := Delete(db, accId)
	if err != sql.ErrNoRows {
		t.Errorf("account deletion test failed err expected sql.ErrNoRows but got: %v:", err)
	}
}

func TestDeleteErrorInDelete(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 232.4, "GBP", utc, utc, true)

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnRows(rows)

	deleteQuery := "DELETE FROM accounts WHERE id=\\$1;"

	mock.ExpectExec(deleteQuery).WithArgs(1).WillReturnError(sql.ErrConnDone)

	err := Delete(db, accId)
	if errors.Cause(err) != sql.ErrConnDone {
		t.Errorf("account deletion test failed err expected sql.ErrConnDone but got: %v:", err)
	}
}

func TestFreeze(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 232.4, "GBP", utc, nil, false)

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnRows(rows)

	updateQuery := "UPDATE accounts SET frozen = TRUE, modified_at=\\$1 WHERE id=\\$2;"

	mock.ExpectPrepare(updateQuery).ExpectQuery().WithArgs(sqlmock.AnyArg(), 1).WillReturnRows()

	actualAcc, err := Freeze(db, accId)

	assert.NoError(t, err)
	assert.NotNil(t, actualAcc)
	assert.True(t, actualAcc.Frozen)
	assert.NotNil(t, actualAcc.ModifiedAt)
}

func TestFreezeErrorInSelect(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnError(sql.ErrNoRows)

	_, err := Freeze(db, accId)
	if err != sql.ErrNoRows {
		t.Errorf("account freeze test failed err expected sql.ErrNoRows but got: %v:", err)
	}
}

func TestFreezeErrorInUpdate(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	selectQuery := "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=\\$1;"

	accId := 1
	utc := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "customer_id", "balance", "currency", "created_at", "modified_at", "frozen"}).
		AddRow(1, 11, 232.4, "GBP", utc, nil, false)

	mock.ExpectPrepare(selectQuery).ExpectQuery().WithArgs(1).WillReturnRows(rows)

	updateQuery := "UPDATE accounts SET frozen = TRUE, modified_at=\\$1 WHERE id=\\$2;"

	mock.ExpectPrepare(updateQuery).ExpectQuery().WithArgs(sqlmock.AnyArg(), 1).WillReturnError(sql.ErrTxDone)

	_, err := Freeze(db, accId)
	if errors.Cause(err) != sql.ErrTxDone {
		t.Errorf("account deletion test failed err expected sql.ErrTxDone but got: %v:", err)
	}
}

func NewMockDb() (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return sqlxDB, mock
}
