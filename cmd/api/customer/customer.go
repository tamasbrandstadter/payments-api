package customer

import (
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
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

func Create(db *sqlx.DB, ar account.AccCreationRequest) (*Customer, error) {
	c := &Customer{
		FirstName:  ar.FirstName,
		LastName:   ar.LastName,
		Email:      ar.Email,
		CreatedAt:  time.Now().UTC(),
		ModifiedAt: time.Now().UTC(),
	}

	stmt, err := db.Prepare(insert)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.WithError(err).Info("create customer")
		}
	}()

	row := stmt.QueryRow(c.FirstName, c.LastName, c.Email, c.CreatedAt, c.ModifiedAt)

	if err = row.Scan(&c.ID); err != nil {
		return nil, err
	}

	log.Infof("successfully created customer with email %s and id %d", c.Email, c.ID)

	return c, nil
}
