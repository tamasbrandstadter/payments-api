package testdb

import (
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/db"
)

const (
	databaseUser = "root"
	databasePass = "root"
	databaseName = "testdb"
	databaseHost = "db"
	databasePort = 5432
)

var TestTime = time.Now().UTC().Truncate(time.Millisecond)

func Open() (*sqlx.DB, error) {
	return db.NewConnection(db.Config{
		User: databaseUser,
		Pass: databasePass,
		Name: databaseName,
		Host: databaseHost,
		Port: databasePort,
	})
}

func SaveCustomerWithAccount(db *sqlx.DB, r account.AccCreationRequest) error {
	stmt, err := db.Prepare("INSERT INTO customers(first_name, last_name, email, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return err
	}

	row := stmt.QueryRow(r.FirstName, r.LastName, r.Email, TestTime, TestTime)
	err = row.Err()
	if err := stmt.Close(); err != nil {
		return err
	}

	stmt, err = db.Prepare("INSERT INTO accounts(customer_id, balance_in_decimal, currency, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return err
	}

	row = stmt.QueryRow(1, r.InitialBalance, r.Currency, TestTime, TestTime)

	var id int
	if err = row.Scan(&id); err != nil {
		if err := stmt.Close(); err != nil {
			return err
		}

		return err
	}

	if err := stmt.Close(); err != nil {
		return err
	}

	return nil
}

func DeleteTestAccount(db *sqlx.DB, id int) error {
	stmt, err := db.Prepare("DELETE FROM accounts WHERE id=$1")
	if err != nil {
		return err
	}

	if _, err = stmt.Exec(id); err != nil {
		return err
	}

	return nil
}

func SelectById(db *sqlx.DB, id int) (*account.Account, error) {
	var acc account.Account

	stmt, err := db.Preparex("SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen FROM accounts WHERE id=$1;")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(err).Info("select account")
		}
	}()

	row := stmt.QueryRowx(id)

	if err := row.StructScan(&acc); err != nil {
		return nil, err
	}

	return &acc, nil
}
