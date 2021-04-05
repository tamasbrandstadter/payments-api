package audit

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/notification"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

type TransactionType int

const (
	Deposit = iota
	Withdraw
	Transfer
)

func (tt *TransactionType) String() string {
	return [...]string{"deposit", "withdraw", "transfer"}[*tt]
}

type TxRecord struct {
	transactionId int       `db:"id"`
	fromId        int       `db:"from_id"`
	toId          int       `db:"to_id"`
	ack           bool      `db:"ack"`
	tt            string    `db:"transaction_type"`
	createdAt     time.Time `db:"created_at"`
}

func SaveAuditRecord(db *sqlx.DB, fromId, toId int, tt TransactionType, conn *mq.Conn) error {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}

	audit := TxRecord{
		fromId:    fromId,
		toId:      toId,
		tt:        tt.String(),
		ack:       true,
		createdAt: time.Now().UTC(),
	}

	stmt, err := tx.Prepare(insert)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	row := stmt.QueryRow(audit.fromId, audit.toId, audit.tt, audit.ack, audit.createdAt)

	if err = row.Scan(&audit.transactionId); err != nil {
		_ = tx.Rollback()
		log.Warnf("audit tx record creation was rolled back, error: %v", err)
		return err
	}
	if err = tx.Commit(); err != nil {
		log.Errorf("failed to commit audit tx record creation, error: %v", err)
		return err
	}

	log.Infof("successfully saved audit record with tx id %d", audit.transactionId)

	notification.PublishSuccessfulTxNotification(conn, audit.transactionId, audit.createdAt)

	return nil
}
