package account

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/jmoiron/sqlx"
)

type Currency string

type Customer struct {
	ID        int       `json:"id" db:"id"`
	FirstName string    `json:"firstName" db:"first_name"`
	LastName  string    `json:"lastName" db:"last_name"`
}

type Account struct {
	ID         int       `json:"id" db:"id"`
	CustomerID int       `json:"customerId" db:"customer_id"`
	Balance    float64   `json:"balance" db:"balance"`
	Currency   Currency  `json:"currency,omitempty" db:"currency"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	Frozen     bool      `json:"frozen" db:"frozen"`
}

func SelectAll(dbc *sqlx.DB) ([]Account, error) {
	accounts := make([]Account, 0)

	if err := dbc.Select(&accounts, selectAll); err != nil {
		return nil, errors.Wrap(err, "select all rows from accounts table")
	}

	return accounts, nil
}

func SelectById(dbc *sqlx.DB, id int) (Account, error) {
	var acc Account

	stmt := selectById

	pStmt, err := dbc.Preparex(stmt)
	if err != nil {
		return Account{}, errors.Wrap(err, "prepare select account query")
	}

	defer func() {
		if err := pStmt.Close(); err != nil {
			logrus.WithError(errors.Wrap(err, "close psql statement")).Info("select account")
		}
	}()

	row := pStmt.QueryRowx(id)

	if err := row.StructScan(&acc); err != nil {
		return Account{}, errors.Wrap(err, "select singular row from account table")
	}

	return acc, nil
}