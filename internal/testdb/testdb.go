package testdb

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/tamasbrandstadter/payments-api/internal/db"
)

const (
	databaseUser = "root"

	databasePass = "root"

	databaseName = "testdb"

	databasePort = 5432
)

var TestTime = time.Now().UTC().Truncate(time.Millisecond)

func Open() (*sqlx.DB, error) {
	return db.NewConnection(db.Config{
		User: databaseUser,
		Pass: databasePass,
		Name: databaseName,
		Port: databasePort,
	})
}

func SaveCustomerWithAccount(dbc *sqlx.DB) error {
	stmt, err := dbc.Prepare("INSERT INTO customers(first_name, last_name, email, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return err
	}

	row := stmt.QueryRow("first", "last", "test@test.com", TestTime, TestTime)
	err = row.Err()
	if err := stmt.Close(); err != nil {
		return err
	}

	stmt, err = dbc.Prepare("INSERT INTO accounts(customer_id, balance, currency, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return err
	}

	row = stmt.QueryRow(1, 999.0, "EUR", TestTime, TestTime)

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
