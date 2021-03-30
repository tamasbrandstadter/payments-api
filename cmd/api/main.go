package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/consumer"
	"github.com/tamasbrandstadter/payments-api/cmd/api/handler"
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

	server := http.Server{
		Addr:           fmt.Sprintf(":%d", 8080),
		Handler:        handler.NewApplication(dbc),
		ReadTimeout:    envCfg.ReadTimeout,
		WriteTimeout:   envCfg.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	mqCfg := mq.Config{
		User: envCfg.MQUser,
		Pass: envCfg.MQPass,
		Host: envCfg.MQHost,
		Port: envCfg.MQPort,
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

	deposit, withdraw, err := conn.DeclareQueues()
	if err != nil {
		log.Errorf("error declaring queues: %v", err)
		return
	}
	balanceHandler := consumer.BalanceOperationConsumer{
		Deposit:  deposit,
		Withdraw: withdraw,
	}

	go func() {
		log.Infof("server started, listening on %s", server.Addr)
		err = server.ListenAndServe()
		if err != nil {
			log.Errorf("server failed to start: %v", err)
			return
		}
	}()

	err = balanceHandler.ConsumeFromQueues(conn, dbc)
	if err != nil {
		log.Errorf("error starting consumers: %v", err)
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