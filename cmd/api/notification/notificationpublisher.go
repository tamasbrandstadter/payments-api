package notification

import (
	"encoding/json"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	exchangeName = "balance-notifications"
	routeKey     = "notif"
	kind         = "topic"
	contentType  = "application/json"
)

type notification struct {
	TransactionId int       `json:"txId"`
	AccountID     int       `json:"accId"`
	CreatedAt     time.Time `json:"createdAt"`
	Ack           bool      `json:"ack"`
}

func PublishSuccessfulTxNotification(conn mq.Conn, txId int, accId int, createdAt time.Time) {
	n := &notification{
		TransactionId: txId,
		AccountID:     accId,
		CreatedAt:     createdAt,
		Ack:           true,
	}

	body, err := json.Marshal(n)
	if err != nil {
		log.Warnf("failed to marshal notification: %v", err)
		return
	}

	err = conn.Channel.ExchangeDeclare(exchangeName, kind, true, false, false, false, nil)
	if err != nil {
		log.Errorf("error declaring exchange for notifications: %v", err)
		return
	}

	id := uuid.New().String()
	err = conn.Channel.Publish(exchangeName, routeKey, false, false, amqp.Publishing{
		ContentType:  contentType,
		MessageId:    id,
		Body:         body,
		DeliveryMode: amqp.Transient,
	})
	if err != nil {
		log.Errorf("error sending notification to balance-notifications topic: %v", err)
		attempt := 0
		err = retry.Do(
			func() error {
				log.Infof("retrying to send notification, attempt %b", attempt)
				err = conn.Channel.Publish(exchangeName, routeKey, false, false, amqp.Publishing{
					ContentType:  contentType,
					MessageId:    id,
					Body:         body,
					DeliveryMode: amqp.Transient,
				})
				if err != nil {
					return err
				}
				return nil
			},
			retry.Attempts(3), retry.Delay(1*time.Second),
		)
	}

}
