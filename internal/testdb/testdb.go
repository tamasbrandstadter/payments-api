package testdb

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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

func DeleteCustomerWithAccount(dbc *sqlx.DB) error {
	stmt := "DELETE FROM accounts WHERE NOT EXISTS(SELECT * FROM customers AS T1 WHERE T1.id == accounts.customer_id);"

	if _, err := dbc.Exec(stmt); err != nil {
		return errors.Wrap(err, "delete account with customer")
	}

	return nil
}

func SaveCustomerWithAccount(dbc *sqlx.DB) error {
	stmt, err := dbc.Prepare("INSERT INTO customers(first_name, last_name, email, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return errors.Wrap(err, "prepare test customer insertion")
	}

	row := stmt.QueryRow("first", "last", "test@test.com", TestTime, TestTime)
	err = row.Err()
	if err := stmt.Close(); err != nil {
		return errors.Wrap(err, "close psql statement")
	}

	stmt, err = dbc.Prepare("INSERT INTO accounts(customer_id, balance, currency, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;")
	if err != nil {
		return errors.Wrap(err, "prepare test acc insertion")
	}

	row = stmt.QueryRow(1, 999.0, "EUR", TestTime, TestTime)

	var id int
	if err = row.Scan(&id); err != nil {
		if err := stmt.Close(); err != nil {
			return errors.Wrap(err, "close psql statement")
		}

		return errors.Wrap(err, "capture test acc id")
	}

	if err := stmt.Close(); err != nil {
		return errors.Wrap(err, "close psql statement")
	}

	return nil
}
