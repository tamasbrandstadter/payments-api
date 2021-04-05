package balance

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/avast/retry-go"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/cmd/api/audit"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	depositConsumer  = "deposit-consumer"
	withdrawConsumer = "withdraw-consumer"
	transferConsumer = "transfer-consumer"
)

type TransactionConsumer struct {
	Deposit     *amqp.Queue
	Withdraw    *amqp.Queue
	Transfer    *amqp.Queue
	Concurrency int
}

func (tc *TransactionConsumer) StartConsume(conn *mq.Conn, db *sqlx.DB) {
	forever := make(chan bool)

	err := tc.startDeposits(conn, db)
	if err != nil {
		log.Errorf("error starting deposit consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from deposits, attempt %b", attempt)
				err = tc.startDeposits(conn, db)
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(10), retry.Delay(3*time.Second),
		)
	}

	err = tc.startWithdraws(conn, db)
	if err != nil {
		log.Errorf("error starting withdraw consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from withdraws, attempt %b", attempt)
				err = tc.startWithdraws(conn, db)
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(10), retry.Delay(3*time.Second),
		)
	}

	err = tc.startTransfers(conn, db)
	if err != nil {
		log.Errorf("error starting transfer consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from transfers, attempt %b", attempt)
				err = tc.startTransfers(conn, db)
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(10), retry.Delay(3*time.Second),
		)
	}

	<-forever
}

func (tc *TransactionConsumer) ClosedConnectionListener(cfg mq.Config, db *sqlx.DB, closed <-chan *amqp.Error) {
	err := <-closed
	if err != nil {
		log.Errorf("closed mq connection: %v", err)

		var i int

		for i = 0; i < cfg.MaxReconnect; i++ {
			log.Info("attempting to reconnect to mq")

			if conn, err := mq.NewConnection(cfg); err == nil {
				log.Info("reconnected to mq")
				tc.StartConsume(conn, db)
			}

			time.Sleep(1 * time.Second)
		}

		if i == cfg.MaxReconnect {
			log.Error("reached max attempts, unable to reconnect to mq")
			return
		}
	} else {
		log.Info("mq connection closed normally, will not reconnect")
		os.Exit(0)
	}
}

func (tc *TransactionConsumer) startDeposits(conn *mq.Conn, db *sqlx.DB) error {
	deposits, err := conn.Channel.Consume(tc.Deposit.Name, depositConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for d := range deposits {
				ok, err2 := handleDeposit(d, db, conn)
				if err2 != nil {
					_ = d.Nack(false, false)
				} else if !ok {
					_ = d.Nack(false, true)
				} else {
					_ = d.Ack(false)
				}
			}
		}()
	}

	return nil
}

func (tc *TransactionConsumer) startWithdraws(conn *mq.Conn, db *sqlx.DB) error {
	withdraws, err := conn.Channel.Consume(tc.Withdraw.Name, withdrawConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for w := range withdraws {
				ok, err2 := handleWithdraw(w, db, conn)
				if err2 != nil {
					_ = w.Nack(false, false)
				} else if !ok {
					_ = w.Nack(false, true)
				} else {
					_ = w.Ack(false)
				}
			}
		}()
	}

	return nil
}

func (tc *TransactionConsumer) startTransfers(conn *mq.Conn, db *sqlx.DB) error {
	withdraws, err := conn.Channel.Consume(tc.Transfer.Name, transferConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for w := range withdraws {
				ok, err2 := handleTransfer(w, db, conn)
				if err2 != nil {
					_ = w.Nack(false, false)
				} else if !ok {
					_ = w.Nack(false, true)
				} else {
					_ = w.Ack(false)
				}
			}
		}()
	}

	return nil
}

func handleTransfer(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn) (bool, error) {
	var payload TransferMessage

	r := bytes.NewReader(d.Body)
	err := json.NewDecoder(r).Decode(&payload)
	if err != nil {
		return false, errors.New("invalid message payload, unable to parse")
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	err = account.Transfer(db, payload.FromID, payload.ToID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == account.InvalidAccounts {
			return false, err
		}

		if te, ok := err.(*account.InvalidTransferError); ok {
			return false, te
		}

		if fe, ok := err.(*account.FundsError); ok {
			return false, fe
		}

		return false, nil
	}

	if err = audit.SaveAuditRecord(db, payload.FromID, payload.ToID, audit.Transfer, conn); err != nil {
		log.Errorf("error saving audit record: %v", err)
	}

	return true, nil
}

func handleDeposit(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	err = account.Deposit(db, payload.AccountID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, errors.New(fmt.Sprintf("account id %d is not found", payload.AccountID))
		}
		return false, nil
	}

	if err = audit.SaveAuditRecord(db, payload.AccountID, 0, audit.Deposit, conn); err != nil {
		log.Errorf("error saving audit record: %v", err)
	}

	return true, nil
}

func handleWithdraw(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	err = account.Withdraw(db, payload.AccountID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, errors.New(fmt.Sprintf("account id %d is not found", payload.AccountID))
		}

		fe, ok := err.(*account.FundsError)
		if ok {
			return false, fe
		}

		return false, nil
	}

	if err = audit.SaveAuditRecord(db, payload.AccountID, 0, audit.Withdraw, conn); err != nil {
		log.Errorf("error saving audit record: %v", err)
	}

	return true, nil
}

func decodeMessage(d amqp.Delivery) (*BalanceMessage, error) {
	var payload BalanceMessage

	r := bytes.NewReader(d.Body)
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, errors.New("invalid message payload, unable to parse")
	}

	return &payload, nil
}

func validateAmount(amount int64) error {
	if amount < 0 {
		return errors.New("balance operation amount can't be negative")
	}
	return nil
}
