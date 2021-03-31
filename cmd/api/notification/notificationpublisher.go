package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/audit"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

const (
	exchangeName = "balance-notifications"
	routeKey     = "notif"
)

type Notification struct {
	AccountID int
	CreatedAt time.Time
	Ack       bool
}

func PublishNotification(conn mq.Conn, auditTx audit.TxAudit) {
	n := Notification{
		AccountID: auditTx.AccountID,
		CreatedAt: auditTx.CreatedAt,
		Ack:       auditTx.Ack,
	}

	body, err := json.Marshal(n)
	if err != nil {
		log.Warnf("failed to marshal notification: %v", err)
		return
	}

	_ = conn.Channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)

	err = conn.Channel.Publish(exchangeName, routeKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		MessageId:    uuid.New().String(),
		Body:         body,
		DeliveryMode: amqp.Transient,
	})
	if err != nil {
		log.Errorf("error sending notification to balance-notifications topic: %v", err)
	}

}
