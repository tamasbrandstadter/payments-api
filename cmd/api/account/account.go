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

var InvalidAccounts = errors.New("invalid transfer, account ids are not found")

type FundsError struct {
	Balance float64
}

func (fe *FundsError) Error() string {
	return fmt.Sprintf("insufficient funds, balance: %.2f", fe.Balance)
}

type InvalidTransferError struct {
	MissingAccountID int
}

func (te *InvalidTransferError) Error() string {
	return fmt.Sprintf("invalid transfer, account id %d not found", te.MissingAccountID)
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

func SelectAll(db *sqlx.DB) ([]Account, error) {
	accounts := make([]Account, 0)

	if err := db.Select(&accounts, selectAll); err != nil {
		return nil, err
	}

	return accounts, nil
}

func SelectById(db *sqlx.DB, id int) (Account, error) {
	var acc Account

	pStmt, err := db.Preparex(selectById)
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

func Create(db *sqlx.DB, customerId int, ar AccCreationRequest) (Account, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
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
		log.Warnf("account creation for customer id %d was rolled back, error: %v", customerId, err)
		return Account{}, err
	}
	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit account creation for customer id %d, error: %v", customerId, err)
		return Account{}, err
	}

	log.Infof("successfully created account with id %d for customer id %d", acc.ID, acc.CustomerID)

	return acc, nil
}

func Delete(db *sqlx.DB, id int) error {
	if _, err := SelectById(db, id); errors.Cause(err) == sql.ErrNoRows {
		return sql.ErrNoRows
	}

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	if _, err = tx.Exec(deleteById, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("account deletion for id %d was rolled back, error: %v", id, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit account deletion for account id %d, error: %v", id, err)
		return err
	}

	log.Infof("successfully deleted account with id %d", id)

	return nil
}

func Freeze(db *sqlx.DB, id int) (Account, error) {
	acc, err := SelectById(db, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return Account{}, sql.ErrNoRows
	}

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
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
		log.Warnf("freeze account for id %d was rolled back %v", id, err)
		return Account{}, err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit account freeze for account id %d, error: %v", id, err)
		return Account{}, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Frozen = true

	log.Infof("successfully frozen account with id %d", id)

	return acc, nil
}

func Deposit(db *sqlx.DB, id int, amount float64) (Account, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
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
		log.Warnf("deposit for account id %d was rolled back, error: %v", id, err)
		return Account{}, err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit deposit to account id %d, error: %v", id, err)
		return Account{}, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	log.Infof("successfully deposited amount %.2f to account id %d", amount, id)

	return acc, nil
}

func Withdraw(db *sqlx.DB, id int, amount float64) (Account, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
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
		log.Warnf("withdraw for account id %d was rolled back due to insufficient funds", id)
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
		log.Warnf("withdraw from account id %d was rolled back, error: %v", id, err)
		return Account{}, err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit withdraw from account, error: %v", err)
		return Account{}, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Balance = newBalance

	log.Infof("successfully withdrew amount %.2f from account %d", amount, id)

	return acc, nil
}

func Transfer(db *sqlx.DB, fromId int, toId int, amount float64) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}

	accounts := make([]Account, 0)
	if err = tx.Select(&accounts, selectTwoById, fromId, toId); err != nil {
		_ = tx.Rollback()
		return err
	}

	if len(accounts) == 0 {
		_ = tx.Rollback()
		return InvalidAccounts
	}

	if len(accounts) == 1 {
		_ = tx.Rollback()
		var missingId int
		if accounts[0].ID == fromId {
			missingId = fromId
		} else {
			missingId = toId
		}
		return &InvalidTransferError{MissingAccountID: missingId}
	}

	from := accounts[0]
	to := accounts[1]

	if from.Balance < amount {
		_ = tx.Rollback()
		return &FundsError{Balance: from.Balance}
	}

	fromNewBalance := from.Balance - amount
	toNewBalance := to.Balance + amount

	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(updateBalances)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = stmt.Exec(from.ID, fromNewBalance, modifiedAt, to.ID, toNewBalance, modifiedAt); err != nil {
		_ = tx.Rollback()
		log.Warnf("transfer from account id %d to account id %d was rolled back, error: %v", fromId, toId, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit transfer from account id %d to account id %d, error: %v", fromId, toId, err)
		return err
	}

	log.Infof("successfully transfered amount %.2f from account id %d to account id %d", amount, fromId, toId)

	return nil
}
