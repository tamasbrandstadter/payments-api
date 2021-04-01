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

type TxRecord struct {
	transactionId int       `db:"id"`
	accountID     int       `db:"account_id"`
	ack           bool      `db:"ack"`
	createdAt     time.Time `db:"created_at"`
}

func SaveAuditRecord(db *sqlx.DB, accId int, conn mq.Conn) error {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}

	audit := TxRecord{
		accountID: accId,
		ack:       true,
		createdAt: time.Now().UTC(),
	}

	stmt, err := tx.Prepare(insert)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	row := stmt.QueryRow(audit.accountID, audit.ack, audit.createdAt)

	if err = row.Scan(&audit.transactionId); err != nil {
		_ = tx.Rollback()
		log.Warnf("audit tx record creation for account id %d was rolled back", accId)
		return err
	}
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit audit tx record creation, error: ", err)
		return err
	}

	notification.PublishSuccessfulTxNotification(conn, audit.transactionId, audit.accountID, audit.createdAt)

	return nil
}
