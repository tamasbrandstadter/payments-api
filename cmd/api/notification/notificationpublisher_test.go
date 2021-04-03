package notification

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"

	"github.com/tamasbrandstadter/payments-api/internal/mq"
	"github.com/tamasbrandstadter/payments-api/internal/testmq"
)

func TestPublishSuccessfulTxNotification(t *testing.T) {
	conn := NewConn()

	utc := time.Now().UTC()
	txId := 55345

	PublishSuccessfulTxNotification(conn, txId, utc)

	messages, err := conn.Channel.Consume("test-queue", "test-consumer", false, false, false, false, nil)
	if err != nil {
		t.Errorf("test publish successfull tx notification failed, err nil expected, got: %v", err)
	}

	m := <-messages

	var n notification

	r := bytes.NewReader(m.Body)
	if err = json.NewDecoder(r).Decode(&n); err != nil {
		t.Errorf("test publish successfull tx notification failed, err nil expected, got: %v", err)
	}

	assert.Equal(t, txId, n.TransactionId)
	assert.Equal(t, utc, n.CreatedAt)
	assert.True(t, n.Ack)
}

func NewConn() mq.Conn {
	conn, err := testmq.Open()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a test mq connection", err)
	}

	return conn
}
