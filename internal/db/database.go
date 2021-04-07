package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var PSQLErrUniqueConstraint = "23505"

const driver = "postgres"

type Config struct {
	User string
	Pass string
	Name string
	Host string
	Port int
}

func NewConnection(cfg Config) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	dsn := fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=disable",
		cfg.User,
		cfg.Pass,
		cfg.Host,
		cfg.Port,
		cfg.Name)

	log.Info("connecting to db")
	if db, err = sqlx.Connect(driver, dsn); err != nil {
		return nil, err
	}

	log.Info("verifying db connection")
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Info("verified db connection")
	return db, nil
}
