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
		return nil, err
	}

	return accounts, nil
}

func SelectById(dbc *sqlx.DB, id int) (Account, error) {
	var acc Account

	pStmt, err := dbc.Preparex(selectById)
	if err != nil {
		return Account{}, err
	}

	defer func() {
		if err := pStmt.Close(); err != nil {
			log.WithError(err).Info("select account")
		}
	}()

	row := pStmt.QueryRowx(id)

	if err := row.StructScan(&acc); err != nil {
		return Account{}, err
	}

	return acc, nil
}

func Create(dbc *sqlx.DB, customerId int, ar AccCreationRequest) (Account, error) {
	tx, err := dbc.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
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
		_ = tx.Rollback()
		return Account{}, err
	}

	row := stmt.QueryRow(acc.CustomerID, acc.Balance, acc.Currency, acc.CreatedAt, acc.ModifiedAt)

	if err = row.Scan(&acc.ID); err != nil {
		_ = tx.Rollback()
		log.Warnf("account creation for customer id %d was rolled back", customerId)
		return Account{}, err
	}
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit account creation, error: ", err)
		return Account{}, err
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
		return err
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
		_ = tx.Rollback()
		return Account{}, err
	}

	if _, err = stmt.Exec(modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("freeze account for id %d was rolled back", id)
		return Account{}, err
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
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := dbc.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return Account{}, err
	}

	var acc Account

	row := tx.QueryRowx(selectById, id)

	err = row.StructScan(&acc)
	if errors.Cause(err) == sql.ErrNoRows {
		_ = tx.Rollback()
		return Account{}, sql.ErrNoRows
	} else if err != nil {
		_ = tx.Rollback()
		return Account{}, err
	}

	modifiedAt := time.Now().UTC()
	newBalance := acc.Balance + amount

	stmt, err := tx.Prepare(updateBalance)
	if err != nil {
		return Account{}, err
	}

	if _, err = stmt.Exec(newBalance, modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("deposit for account id %d was rolled back", id)
		return Account{}, err
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit deposit to account, error: ", err)
		return Account{}, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	return acc, nil
}

func Withdraw(dbc *sqlx.DB, id int, amount float64) (Account, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := dbc.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return Account{}, err
	}

	var acc Account

	row := tx.QueryRowx(selectById, id)

	err = row.StructScan(&acc)
	if errors.Cause(err) == sql.ErrNoRows {
		_ = tx.Rollback()
		return Account{}, sql.ErrNoRows
	} else if err != nil {
		_ = tx.Rollback()
		return Account{}, err
	}

	newBalance := acc.Balance - amount
	if newBalance < 0 {
		_ = tx.Rollback()
		log.Warnf("withdraw for account id %d was rolled back", id)
		return Account{}, &FundsError{Balance: acc.Balance}
	}
	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(updateBalance)
	if err != nil {
		_ = tx.Rollback()
		return Account{}, err
	}

	if _, err = stmt.Exec(newBalance, modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("withdraw from account id %d was rolled back", id)
		return Account{}, err
	}

	if err = tx.Commit(); err != nil {
		log.Error("failed to commit withdraw from account, error: ", err)
		return Account{}, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	return acc, nil
}
