package balance

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

type TransactionConsumer struct {
	Deposit     amqp.Queue
	Withdraw    amqp.Queue
	Concurrency int
}

func (tc TransactionConsumer) StartConsume(conn mq.Conn, db *sqlx.DB) error {
	deposits, err := conn.Channel.Consume(tc.Deposit.Name, "deposit-consumer", false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	withdraws, err := conn.Channel.Consume(tc.Withdraw.Name, "withdraw-consumer", false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for d := range deposits {
				ok, err2 := handleDeposit(d, db)
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

	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for w := range withdraws {
				ok, err2 := handleWithdraw(w, db)
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

	forever := make(chan bool)
	<-forever

	return nil
}

func (tc TransactionConsumer) ClosedConnectionListener(cfg mq.Config, db *sqlx.DB, closed <-chan *amqp.Error) {
	err := <-closed
	if err != nil {
		log.Errorf("closed mq connection: %v", err)

		var i int

		for i = 0; i < cfg.MaxReconnect; i++ {
			log.Info("attempting to reconnect to mq")

			if conn, err := mq.NewConnection(cfg); err == nil {
				log.Info("reconnected to mq")
				_ = tc.StartConsume(conn, db)
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

func handleDeposit(d amqp.Delivery, db *sqlx.DB) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	_, err = account.Deposit(db, payload.ID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, errors.New(fmt.Sprintf("account id %d is not found", payload.ID))
		}
		return false, nil
	}

	log.Infof("successfully deposited amount %.2f to account %d", payload.Amount, payload.ID)
	return true, nil
}

func handleWithdraw(d amqp.Delivery, db *sqlx.DB) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	_, err = account.Withdraw(db, payload.ID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, errors.New(fmt.Sprintf("account id %d is not found", payload.ID))
		}

		fe, ok := err.(*account.FundsError)
		if ok {
			return false, fe
		}

		return false, nil
	}

	log.Infof("successfully withdrew amount %.2f from account %d", payload.Amount, payload.ID)
	return true, nil
}

func decodeMessage(d amqp.Delivery) (account.BalanceOperationRequest, error) {
	var payload account.BalanceOperationRequest

	r := bytes.NewReader(d.Body)
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return account.BalanceOperationRequest{}, errors.New("invalid message payload, unable to parse")
	}

	return payload, nil
}

func validateAmount(amount float64) error {
	if amount < 0 {
		return errors.New("balance operation amount can't be negative")
	}
	return nil
}
