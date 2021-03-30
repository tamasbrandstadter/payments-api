package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/handlers"
	"github.com/tamasbrandstadter/payments-api/internal/db"
	"github.com/tamasbrandstadter/payments-api/internal/env"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

func main() {
	log.SetFormatter(&log.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})

	envCfg := env.GetEnvCfg()

	dbCfg := db.Config{
		User: envCfg.DBUser,
		Pass: envCfg.DBPass,
		Name: envCfg.DBName,
		Port: envCfg.DBPort,
	}
	dbc, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("connect to db: ", err)
	}

	defer func() {
		if err := dbc.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	server := http.Server{
		Addr:           fmt.Sprintf(":%d", 8080),
		Handler:        handlers.NewApplication(dbc),
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
		log.Fatal("connect to mq: ", err)
	}

	deposit, withdraw, err := conn.DeclareQueues()
	if err != nil {
		log.Fatal("unable to declare queues: ", err)
	}
	balanceHandler := handlers.BalanceOperationConsumer{
		Deposit: deposit,
		Withdraw: withdraw,
	}

	go func() {
		log.Printf("server started, listening on %s", server.Addr)
		err = server.ListenAndServe()
		if err != nil {
			log.Fatal("server failed to start: ", err)
		}
	}()

	balanceHandler.StartConsumers(conn, dbc)

	ctx, cancel := context.WithTimeout(context.Background(), envCfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown : Graceful shutdown did not complete in %v : %v", envCfg.ShutdownTimeout, err)

		if err := server.Close(); err != nil {
			log.Printf("shutdown : Error killing server : %v", err)
		}
	}
}
