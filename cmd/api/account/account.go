package account

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Currency string

func (c Currency) Supported() bool {
	return c == "EUR" || c == "GBP" || c == "USD"
}

type Account struct {
	ID         int       `json:"id" db:"id"`
	CustomerID int       `json:"customerId" db:"customer_id"`
	Balance    float64   `json:"balance" db:"balance"`
	Currency   Currency  `json:"currency,omitempty" db:"currency"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	ModifiedAt time.Time `json:"modifiedAt" db:"modified_at"`
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

	pStmt, err := dbc.Preparex(selectById)
	if err != nil {
		return Account{}, errors.Wrap(err, "prepare select account query")
	}

	defer func() {
		if err := pStmt.Close(); err != nil {
			log.WithError(errors.Wrap(err, "close psql statement")).Info("select account")
		}
	}()

	row := pStmt.QueryRowx(id)

	if err := row.StructScan(&acc); err != nil {
		return Account{}, errors.Wrap(err, "select singular row from account table")
	}

	return acc, nil
}

func Create(dbc *sqlx.DB, customerId int, ar AccCreationRequest) (Account, error) {
	acc := Account{
		CustomerID: customerId,
		Balance:    ar.InitialBalance,
		Currency:   ar.Currency,
		CreatedAt:  time.Now().UTC(),
		ModifiedAt: time.Now().UTC(),
		Frozen:     false,
	}

	stmt, err := dbc.Prepare(insert)
	if err != nil {
		return Account{}, errors.Wrap(err, "insert new account row")
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(errors.Wrap(err, "close psql statement")).Info("create account")
		}
	}()

	row := stmt.QueryRow(acc.CustomerID, acc.Balance, acc.Currency, acc.CreatedAt, acc.ModifiedAt)

	if err = row.Scan(&acc.ID); err != nil {
		return Account{}, errors.Wrap(err, "get inserted row id for account")
	}

	return acc, nil
}

func Delete(dbc *sqlx.DB, id int) error {
	if _, err := SelectById(dbc, id); errors.Cause(err) == sql.ErrNoRows {
		return sql.ErrNoRows
	}

	if _, err := dbc.Exec(deleteById, id); err != nil {
		return errors.Wrap(err, "delete account row")
	}

	return nil
}

func Freeze(dbc *sqlx.DB, id int) (Account, error) {
	acc, err := SelectById(dbc, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return Account{}, sql.ErrNoRows
	}

	modifiedAt := time.Now().UTC()

	stmt, err := dbc.Prepare(freezeById)
	if err != nil {
		return Account{}, errors.Wrap(err, "freeze account row")
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(errors.Wrap(err, "close psql statement")).Info("freeze account")
		}
	}()

	if err = stmt.QueryRow(modifiedAt, id).Err(); err != nil {
		return Account{}, errors.Wrap(err, "get inserted row id for account freeze")
	}

	acc.ModifiedAt = modifiedAt
	acc.Frozen = true

	return acc, nil
}

func Deposit(dbc *sqlx.DB, id int, amount float64) (Account, error) {
	acc, err := SelectById(dbc, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return Account{}, sql.ErrNoRows
	}

	modifiedAt := time.Now().UTC()
	newBalance := acc.Balance + amount

	stmt, err := dbc.Prepare(deposit)
	if err != nil {
		return Account{}, errors.Wrap(err, "deposit to account row")
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(errors.Wrap(err, "close psql statement")).Info("deposit to account")
		}
	}()

	if err = stmt.QueryRow(newBalance, modifiedAt, id).Err(); err != nil {
		return Account{}, errors.Wrap(err, "get inserted row id for deposit")
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	return acc, nil
}
