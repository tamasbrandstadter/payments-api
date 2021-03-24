package account

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type FundsError struct {
	Balance float64
}

func (fe *FundsError) Error() string {
	return fmt.Sprintf("insufficient funds, balance: %.2f", fe.Balance)
}

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
	tx, err := dbc.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Account{}, err
	}

	acc := Account{
		CustomerID: customerId,
		Balance:    ar.InitialBalance,
		Currency:   ar.Currency,
		CreatedAt:  time.Now().UTC(),
		ModifiedAt: time.Now().UTC(),
		Frozen:     false,
	}

	stmt, err := tx.Prepare(insert)
	if err != nil {
		return Account{}, errors.Wrap(err, "insert new account row prepare")
	}

	row := stmt.QueryRow(acc.CustomerID, acc.Balance, acc.Currency, acc.CreatedAt, acc.ModifiedAt)

	if err = row.Scan(&acc.ID); err != nil {
		_ = tx.Rollback()
		log.Warnf("account creation for customer id %d was rolled back", customerId)
		return Account{}, err
	}
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit account creation, error: ", err)
		return Account{}, errors.Wrap(err, "account commit")
	}

	return acc, nil
}

func Delete(dbc *sqlx.DB, id int) error {
	if _, err := SelectById(dbc, id); errors.Cause(err) == sql.ErrNoRows {
		return sql.ErrNoRows
	}

	tx, err := dbc.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	if _, err = tx.Exec(deleteById, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("account deletion for id %d was rolled back", id)
		return errors.Wrap(err, "delete account row")
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit account deletion, error: ", err)
		return err
	}
	return nil
}

func Freeze(dbc *sqlx.DB, id int) (Account, error) {
	acc, err := SelectById(dbc, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return Account{}, sql.ErrNoRows
	}

	tx, err := dbc.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Account{}, err
	}

	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(freezeById)
	if err != nil {
		return Account{}, errors.Wrap(err, "freeze account row")
	}

	if err = stmt.QueryRow(modifiedAt, id).Err(); err != nil {
		_ = tx.Rollback()
		log.Warnf("freeze account for id %d was rolle back", id)
		return Account{}, errors.Wrap(err, "get inserted row id for account freeze")
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit account freeze, error: ", err)
		return Account{}, err
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

	stmt, err := dbc.Prepare(updateBalance)
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

func Withdraw(dbc *sqlx.DB, id int, amount float64) (Account, error) {
	acc, err := SelectById(dbc, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return Account{}, sql.ErrNoRows
	}

	newBalance := acc.Balance - amount
	if newBalance < 0 {
		return Account{}, &FundsError{Balance: acc.Balance}
	}
	modifiedAt := time.Now().UTC()

	stmt, err := dbc.Prepare(updateBalance)
	if err != nil {
		return Account{}, errors.Wrap(err, "withdraw to account row")
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(errors.Wrap(err, "close psql statement")).Info("withdraw to account")
		}
	}()

	if err = stmt.QueryRow(newBalance, modifiedAt, id).Err(); err != nil {
		return Account{}, errors.Wrap(err, "get inserted row id for withdraw")
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	return acc, nil
}
