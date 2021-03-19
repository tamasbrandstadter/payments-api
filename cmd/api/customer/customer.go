package customer

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
)

type Customer struct {
	ID         int       `json:"id" db:"id"`
	FirstName  string    `json:"firstName" db:"first_name"`
	LastName   string    `json:"lastName" db:"last_name"`
	Email      string    `json:"email" db:"email"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	ModifiedAt time.Time `json:"modifiedAt" db:"modified_at"`
}

func Create(dbc *sqlx.DB, ar account.CreateAccountRequest) (Customer, error) {
	c := Customer{
		FirstName:  ar.FirstName,
		LastName:   ar.LastName,
		Email:      ar.Email,
		CreatedAt:  time.Now().UTC(),
		ModifiedAt: time.Now().UTC(),
	}

	stmt, err := dbc.Prepare(insert)
	if err != nil {
		return Customer{}, errors.Wrap(err, "insert new customer row")
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			logrus.WithError(errors.Wrap(err, "close psql statement")).Info("create customer")
		}
	}()

	row := stmt.QueryRow(c.FirstName, c.LastName, c.Email, c.CreatedAt, c.ModifiedAt)

	if err = row.Scan(&c.ID); err != nil {
		return Customer{}, errors.Wrap(err, "get inserted row id for customer")
	}

	return c, nil
}
