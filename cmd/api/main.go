package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/balance"
	"github.com/tamasbrandstadter/payments-api/cmd/api/handler"
	"github.com/tamasbrandstadter/payments-api/internal/cache"
	"github.com/tamasbrandstadter/payments-api/internal/db"
	"github.com/tamasbrandstadter/payments-api/internal/env"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

func main() {
	log.SetFormatter(&log.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})

	envCfg, err := env.GetEnvCfg()
	if err != nil {
		log.Errorf("error parsing env vars: %v", err)
	}

	dbCfg := db.Config{
		User: envCfg.DBUser,
		Pass: envCfg.DBPass,
		Name: envCfg.DBName,
		Host: envCfg.DBHost,
		Port: envCfg.DBPort,
	}
	dbc, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Errorf("error connecting to db: %v", err)
		return
	}

	defer func() {
		if err := dbc.Close(); err != nil {
			log.Errorf("error closing db: %v", err)
		}
	}()

	mqCfg := mq.Config{
		User:         envCfg.MQUser,
		Pass:         envCfg.MQPass,
		Host:         envCfg.MQHost,
		Port:         envCfg.MQPort,
		Concurrency:  envCfg.MQConcurrency,
		MaxReconnect: envCfg.MQMaxReconnect,
	}
	conn, err := mq.NewConnection(mqCfg)
	if err != nil {
		log.Errorf("error connecting to mq: %v", err)
		return
	}

	defer func() {
		if err := conn.Channel.Close(); err != nil {
			log.Errorf("error closing mq channel: %v", err)
		}
	}()

	redisCfg := cache.Config{
		Host: envCfg.CacheHost,
		Pass: envCfg.CachePass,
		Port: envCfg.CachePort,
	}

	redis := cache.NewConnection(redisCfg)

	defer func() {
		if err := redis.Client.Close(); err != nil {
			log.Errorf("error closing redis client: %v", err)
		}
	}()

	deposit, withdraw, transfer, err := conn.DeclareQueues(mqCfg.Concurrency)
	if err != nil {
		log.Errorf("error declaring queues: %v", err)
		return
	}
	tc := balance.TransactionConsumer{
		Deposit:     deposit,
		Withdraw:    withdraw,
		Transfer:    transfer,
		Concurrency: mqCfg.Concurrency,
	}

	server := http.Server{
		Addr:           fmt.Sprintf(":%d", 8080),
		Handler:        handler.NewApplication(dbc, redis),
		ReadTimeout:    envCfg.ReadTimeout,
		WriteTimeout:   envCfg.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Infof("server started successfully, listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	tc.StartConsuming(conn, dbc, redis)
	go tc.ClosedConnectionListener(mqCfg, dbc, conn.Channel.NotifyClose(make(chan *amqp.Error)), redis)

	// Blocking main and waiting for shutdown of the daemon.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	// Waiting for an osSignal or a non-HTTP related server error.
	select {
	case e := <-serverErrors:
		err = fmt.Errorf("server failed to start: %+v", e)
		return

	case <-osSignals:
	}

	ctx, cancel := context.WithTimeout(context.Background(), envCfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Warnf("shutdown: Graceful shutdown did not complete in %v : %v", envCfg.ShutdownTimeout, err)

		if err := server.Close(); err != nil {
			log.Warnf("shutdown: Error killing server : %v", err)
		}
	}
}
