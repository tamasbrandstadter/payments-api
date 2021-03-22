package customer

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
)

var (
	id               = int(uuid.New().ID())
	createdAt        = time.Now().UTC()
	expectedCustomer = &Customer{
		ID:         id,
		FirstName:  "first",
		LastName:   "last",
		Email:      "first@last.com",
		CreatedAt:  createdAt,
		ModifiedAt: createdAt,
	}
)

func TestCreate(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO customers\\(first_name, last_name, email, created_at, modified_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	rows := sqlmock.NewRows([]string{"id"}).AddRow(id)

	request := account.CreateAccountRequest{
		FirstName: "first",
		LastName:  "last",
		Email:     "first@last.com",
	}

	mock.ExpectPrepare(query).ExpectQuery().WithArgs(request.FirstName, request.LastName, request.Email, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	actualCustomer, err := Create(db, request)

	assert.NoError(t, err)

	assert.NotNil(t, actualCustomer)
	assert.Equal(t, id, actualCustomer.ID)
	assert.Equal(t, expectedCustomer.FirstName, actualCustomer.FirstName)
	assert.Equal(t, expectedCustomer.LastName, actualCustomer.LastName)
	assert.Equal(t, expectedCustomer.Email, actualCustomer.Email)
	assert.NotNil(t, actualCustomer.CreatedAt)
	assert.NotNil(t, actualCustomer.ModifiedAt)
}

func TestCreateCustomerError(t *testing.T) {
	db, mock := NewMockDb()
	defer db.Close()

	query := "INSERT INTO customers\\(first_name, last_name, email, created_at, modified_at\\) VALUES\\(\\$1,\\$2,\\$3,\\$4,\\$5\\) RETURNING id;"

	mock.ExpectPrepare(query).WillReturnError(errors.New("sql: database is closed"))

	request := account.CreateAccountRequest{
		FirstName: "first",
		LastName:  "last",
		Email:     "first@last.com",
	}

	_, err := Create(db, request)

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
