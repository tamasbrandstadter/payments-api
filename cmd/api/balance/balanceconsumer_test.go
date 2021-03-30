package balance

import (
	"testing"
	"time"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
)

func TestDeposit(t *testing.T) {
	err := testdb.SaveCustomerWithAccount(a.DB)

	msg := []byte("{\"id\":1,\"amount\":1}")
	err = a.Conn.Channel.Publish("payments", "dep", false, false, amqp.Publishing{
		Body:         msg,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
	})

	go func() {
		_ = a.Tc.StartConsume(a.Conn, a.DB)
	}()

	time.Sleep(time.Second / 2)

	acc, err := testdb.SelectById(a.DB, 1)
	if err != nil {
		t.Errorf("expected err nil, got %v", err)
	}
	assert.Equal(t, 1000.0, acc.Balance)
}

func TestWithDraw(t *testing.T) {
	msg := []byte("{\"id\":1,\"amount\":2}")
	err := a.Conn.Channel.Publish("payments", "wit", false, false, amqp.Publishing{
		Body:         msg,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
	})

	time.Sleep(time.Second / 2)

	acc, err := testdb.SelectById(a.DB, 1)
	if err != nil {
		t.Errorf("expected err nil, got %v", err)
	}
	assert.Equal(t, 998.0, acc.Balance)
}
