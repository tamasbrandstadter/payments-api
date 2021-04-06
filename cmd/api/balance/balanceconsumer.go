package balance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/avast/retry-go"
	"github.com/go-redis/cache/v8"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/cmd/api/audit"
	c "github.com/tamasbrandstadter/payments-api/internal/cache"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	depositConsumer  = "deposit-consumer"
	withdrawConsumer = "withdraw-consumer"
	transferConsumer = "transfer-consumer"
)

type fn func(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn, c *c.Redis) (bool, error)

type TransactionConsumer struct {
	Deposit     *amqp.Queue
	Withdraw    *amqp.Queue
	Transfer    *amqp.Queue
	Concurrency int
}

func (tc *TransactionConsumer) StartConsuming(conn *mq.Conn, db *sqlx.DB, cache *c.Redis) {
	forever := make(chan bool)

	err := tc.consumeDeposits(conn, db, cache)
	if err != nil {
		log.Errorf("error starting deposit consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from deposits, attempt %b", attempt)
				err = tc.consumeDeposits(conn, db, cache)
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(10), retry.Delay(3*time.Second),
		)
	}

	err = tc.consumeWithdraws(conn, db, cache)
	if err != nil {
		log.Errorf("error starting withdraw consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from withdraws, attempt %b", attempt)
				err = tc.consumeWithdraws(conn, db, cache)
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(10), retry.Delay(3*time.Second),
		)
	}

	err = tc.consumeTransfers(conn, db, cache)
	if err != nil {
		log.Errorf("error starting transfer consumer: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to consume from transfers, attempt %b", attempt)
				err = tc.consumeTransfers(conn, db, cache)
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

func (tc *TransactionConsumer) ClosedConnectionListener(cfg mq.Config, db *sqlx.DB, closed <-chan *amqp.Error, redis *c.Redis) {
	err := <-closed
	if err != nil {
		log.Errorf("closed mq connection: %v", err)

		var i int

		for i = 0; i < cfg.MaxReconnect; i++ {
			log.Info("attempting to reconnect to mq")

			if conn, err := mq.NewConnection(cfg); err == nil {
				log.Info("reconnected to mq")
				tc.StartConsuming(conn, db, redis)
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

func (tc *TransactionConsumer) consumeDeposits(conn *mq.Conn, db *sqlx.DB, cache *c.Redis) error {
	deposits, err := conn.Channel.Consume(tc.Deposit.Name, depositConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	tc.handleMessage(conn, db, cache, deposits, deposit)

	return nil
}

func (tc *TransactionConsumer) consumeWithdraws(conn *mq.Conn, db *sqlx.DB, cache *c.Redis) error {
	withdraws, err := conn.Channel.Consume(tc.Withdraw.Name, withdrawConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	tc.handleMessage(conn, db, cache, withdraws, withdraw)

	return nil
}

func (tc *TransactionConsumer) consumeTransfers(conn *mq.Conn, db *sqlx.DB, cache *c.Redis) error {
	transfers, err := conn.Channel.Consume(tc.Transfer.Name, transferConsumer, false, false,
		false, false, nil,
	)
	if err != nil {
		return err
	}

	tc.handleMessage(conn, db, cache, transfers, transfer)

	return nil
}

func (tc *TransactionConsumer) handleMessage(conn *mq.Conn, db *sqlx.DB, cache *c.Redis, msgs <-chan amqp.Delivery, f fn) {
	for i := 0; i < tc.Concurrency; i++ {
		go func() {
			for m := range msgs {
				ok, err := f(m, db, conn, cache)
				if err != nil {
					_ = m.Nack(false, false)
				} else if !ok {
					_ = m.Nack(false, true)
				} else {
					_ = m.Ack(false)
				}
			}
		}()
	}
}

func transfer(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn, c *c.Redis) (bool, error) {
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

	fromBalance, toBalance, err := account.Transfer(db, payload.FromID, payload.ToID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == account.InvalidAccountsError {
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

	updateBalanceCache(fromBalance, c, payload.FromID)
	updateBalanceCache(toBalance, c, payload.ToID)

	if err = audit.SaveAuditRecord(db, payload.FromID, payload.ToID, audit.Transfer, conn); err != nil {
		log.Errorf("error saving audit record: %v", err)
	}

	return true, nil
}

func deposit(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn, c *c.Redis) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	balance, err := account.Deposit(db, payload.AccountID, payload.Amount)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, errors.New(fmt.Sprintf("account id %d is not found", payload.AccountID))
		}
		return false, nil
	}

	updateBalanceCache(balance, c, payload.AccountID)

	if err = audit.SaveAuditRecord(db, payload.AccountID, 0, audit.Deposit, conn); err != nil {
		log.Errorf("error saving audit record: %v", err)
	}

	return true, nil
}

func withdraw(d amqp.Delivery, db *sqlx.DB, conn *mq.Conn, c *c.Redis) (bool, error) {
	payload, err := decodeMessage(d)
	if err != nil {
		return false, err
	}

	err = validateAmount(payload.Amount)
	if err != nil {
		return false, err
	}

	balance, err := account.Withdraw(db, payload.AccountID, payload.Amount)
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

	updateBalanceCache(balance, c, payload.AccountID)

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

func updateBalanceCache(balance *money.Money, c *c.Redis, id int) {
	value, err := balance.MarshalJSON()
	if err == nil {
		if err = c.Balances.Set(&cache.Item{
			Ctx:   context.Background(),
			Key:   strconv.Itoa(id),
			Value: value,
			TTL:   time.Hour,
		}); err != nil {
			log.Errorf("error setting new balance in cache for account id %d, error: %v", id, err)
		}
	} else {
		log.Warnf("failed to marshal balance error: %v", err)
	}
}
