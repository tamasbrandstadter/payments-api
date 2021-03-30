package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var PSQLErrUniqueConstraint = "23505"

type Config struct {
	User string
	Pass string
	Name string
	Port int
}

func NewConnection(cfg Config) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	conn := fmt.Sprintf("user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.User, cfg.Pass, cfg.Name, cfg.Port)

	log.Info("connecting to database...")
	if db, err = sqlx.Connect("postgres", conn); err != nil {
		return nil, err
	}

	log.Info("verifying connection...")
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info("verified postgres connection")
	return db, nil
}
