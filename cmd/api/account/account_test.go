package account

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

var customerId = 22

func TestCreate(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO accounts\\(customer_id, balance, currency, created_at, modified_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	rows := sqlmock.NewRows([]string{"id"}).AddRow(11)

	request := CreateAccountRequest{
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

func NewMockDb() (*sqlx.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return sqlxDB, mock
}
