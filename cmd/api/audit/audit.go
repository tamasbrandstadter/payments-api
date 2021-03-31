package audit

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type TxAudit struct {
	ID        int       `json:"id" db:"id"`
	AccountID int       `json:"accountId" db:"account_id"`
	Ack       bool      `json:"ack" db:"ack"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

func SaveAuditTx(db *sqlx.DB, accId int, ack bool) (*TxAudit, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return nil, err
	}

	audit := TxAudit{
		AccountID: accId,
		Ack:       ack,
		CreatedAt: time.Now().UTC(),
	}

	stmt, err := tx.Prepare(insert)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	row := stmt.QueryRow(audit.AccountID, audit.Ack, audit.CreatedAt)

	if err = row.Scan(&audit.ID); err != nil {
		_ = tx.Rollback()
		log.Warnf("audit tx record creation for account id %d was rolled back", accId)
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		log.Error("failed to commit audit tx record creation, error: ", err)
		return nil, err
	}

	return &audit, nil
}
