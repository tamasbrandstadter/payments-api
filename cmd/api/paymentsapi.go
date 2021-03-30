package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/handlers"
	"github.com/tamasbrandstadter/payments-api/internal/db"
	"github.com/tamasbrandstadter/payments-api/internal/mq"
)

func main() {
	log.SetFormatter(&log.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})

	var cfg struct {
		DBUser string `envconfig:"DB_USER"`
		DBPass string `envconfig:"DB_PASSWORD"`
		DBName string `envconfig:"DB_NAME"`
		DBPort int    `envconfig:"DB_PORT" default:"5432"`

		ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
		WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
		ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
	}
	if err := envconfig.Process("APP", &cfg); err != nil {
		log.Fatal("parse environment variables: ", err)
	}

	dbCfg := db.Config{
		User: cfg.DBUser,
		Pass: cfg.DBPass,
		Name: cfg.DBName,
		Port: cfg.DBPort,
	}
	dbc, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("connect to postgres db: ", err)
	}

	defer func() {
		if err := dbc.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	server := http.Server{
		Addr:           fmt.Sprintf(":%d", 8080),
		Handler:        handlers.NewApplication(dbc),
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	conn, err := mq.GetConn("amqp://guest:guest@localhost:5672")
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown : Graceful shutdown did not complete in %v : %v", cfg.ShutdownTimeout, err)

		if err := server.Close(); err != nil {
			log.Printf("shutdown : Error killing server : %v", err)
		}
	}
}
