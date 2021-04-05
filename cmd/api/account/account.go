package account

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var InvalidAccounts = errors.New("invalid transfer, account ids are not found")

type FundsError struct {
	balance string
}

func (fe *FundsError) Error() string {
	return fmt.Sprintf("insufficient funds, balance: %s", fe.balance)
}

type InvalidTransferError struct {
	MissingAccountID int
}

func (te *InvalidTransferError) Error() string {
	return fmt.Sprintf("invalid transfer, account id %d not found", te.MissingAccountID)
}

type Account struct {
	ID               int       `json:"id" db:"id"`
	CustomerID       int       `json:"customerId" db:"customer_id"`
	BalanceInDecimal int64     `json:"balanceInDecimal" db:"balance_in_decimal"`
	Currency         string    `json:"currency,omitempty" db:"currency"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	ModifiedAt       time.Time `json:"modifiedAt" db:"modified_at"`
	Frozen           bool      `json:"frozen" db:"frozen"`
}

func SelectAll(db *sqlx.DB) (*[]Account, error) {
	accounts := make([]Account, 0)

	if err := db.Select(&accounts, selectAll); err != nil {
		return nil, err
	}

	return &accounts, nil
}

func SelectById(db *sqlx.DB, id int) (*Account, error) {
	var acc Account

	stmt, err := db.Preparex(selectById)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(err).Info("select account")
		}
	}()

	row := stmt.QueryRowx(id)

	if err := row.StructScan(&acc); err != nil {
		return nil, err
	}

	return &acc, nil
}

func Create(db *sqlx.DB, customerId int, ar AccCreationRequest) (*Account, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	m := money.New(ar.InitialBalance, ar.Currency)

	acc := &Account{
		CustomerID:       customerId,
		BalanceInDecimal: m.Amount(),
		Currency:         m.Currency().Code,
		CreatedAt:        time.Now().UTC(),
		ModifiedAt:       time.Now().UTC(),
		Frozen:           false,
	}

	stmt, err := tx.Prepare(insert)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	row := stmt.QueryRow(acc.CustomerID, acc.BalanceInDecimal, acc.Currency, acc.CreatedAt, acc.ModifiedAt)

	if err = row.Scan(&acc.ID); err != nil {
		_ = tx.Rollback()
		log.Warnf("account creation for customer id %d was rolled back, error: %v", customerId, err)
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit account creation for customer id %d, error: %v", customerId, err)
		return nil, err
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

func Freeze(db *sqlx.DB, id int) (*Account, error) {
	acc, err := SelectById(db, id)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(freezeById)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if _, err = stmt.Exec(modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("freeze account for id %d was rolled back, error: %v", id, err)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit account freeze for account id %d, error: %v", id, err)
		return nil, err
	}

	acc.ModifiedAt = modifiedAt
	acc.Frozen = true

	log.Infof("successfully frozen account with id %d", id)

	return acc, nil
}

func Deposit(db *sqlx.DB, id int, amount int64) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}

	var acc Account

	row := tx.QueryRowx(selectById, id)

	err = row.StructScan(&acc)
	if errors.Cause(err) == sql.ErrNoRows {
		_ = tx.Rollback()
		return sql.ErrNoRows
	} else if err != nil {
		_ = tx.Rollback()
		return err
	}

	balance := money.New(acc.BalanceInDecimal, acc.Currency)
	deposit := money.New(amount, acc.Currency)
	newBalance, err := balance.Add(deposit)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare(updateBalance)
	if err != nil {
		return err
	}

	modifiedAt := time.Now().UTC()

	if _, err = stmt.Exec(newBalance.Amount(), modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("deposit for account id %d was rolled back, error: %v", id, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit deposit to account id %d, error: %v", id, err)
		return err
	}

	acc.ModifiedAt = modifiedAt
	acc.BalanceInDecimal = newBalance.Amount()

	log.Infof("successfully deposited %s to account id %d", deposit.Display(), id)

	return nil
}

func Withdraw(db *sqlx.DB, id int, amount int64) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
	defer cancelFunc()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return err
	}

	var acc Account

	row := tx.QueryRowx(selectById, id)

	err = row.StructScan(&acc)
	if errors.Cause(err) == sql.ErrNoRows {
		_ = tx.Rollback()
		return sql.ErrNoRows
	} else if err != nil {
		_ = tx.Rollback()
		return err
	}

	balance := money.New(acc.BalanceInDecimal, acc.Currency)
	withdraw := money.New(amount, acc.Currency)

	less, _ := balance.LessThan(withdraw)
	if less {
		_ = tx.Rollback()
		log.Warnf("withdraw for account id %d was rolled back due to insufficient funds", id)
		return &FundsError{balance: balance.Display()}
	}

	newBalance, err := balance.Subtract(withdraw)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(updateBalance)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = stmt.Exec(newBalance.Amount(), modifiedAt, id); err != nil {
		_ = tx.Rollback()
		log.Warnf("withdraw from account id %d was rolled back, error: %v", id, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit withdraw from account, error: %v", err)
		return err
	}

	acc.ModifiedAt = modifiedAt
	acc.BalanceInDecimal = newBalance.Amount()

	log.Infof("successfully withdrew %s from account %d", withdraw.Display(), id)

	return nil
}

func Transfer(db *sqlx.DB, fromId int, toId int, amount int64) error {
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
			missingId = toId
		} else {
			missingId = fromId
		}
		return &InvalidTransferError{MissingAccountID: missingId}
	}

	from := accounts[0]
	to := accounts[1]

	balance := money.New(from.BalanceInDecimal, from.Currency)
	transfer := money.New(amount, from.Currency)
	less, _ := balance.LessThan(transfer)
	if less {
		_ = tx.Rollback()
		log.Warnf("transfer from account id %d to account id %d was rolled back due to insufficient funds", from.ID, to.ID)
		return &FundsError{balance: balance.Display()}
	}

	fromNewBalance, _ := balance.Subtract(transfer)
	toNewBalance, _ := money.New(to.BalanceInDecimal, to.Currency).Add(transfer)

	modifiedAt := time.Now().UTC()

	stmt, err := tx.Prepare(updateBalances)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = stmt.Exec(from.ID, fromNewBalance.Amount(), modifiedAt, to.ID, toNewBalance.Amount(), modifiedAt); err != nil {
		_ = tx.Rollback()
		log.Warnf("transfer from account id %d to account id %d was rolled back, error: %v", fromId, toId, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit transfer from account id %d to account id %d, error: %v", fromId, toId, err)
		return err
	}

	log.Infof("successfully transfered %s from account id %d to account id %d", transfer.Display(), from.ID, to.ID)

	return nil
}
