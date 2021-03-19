package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/cmd/api/customer"
	"github.com/tamasbrandstadter/payments-api/internal/db"
	"github.com/tamasbrandstadter/payments-api/internal/web"
)

func (a *Application) GetAccountById(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, err := strconv.Atoi(params["id"])
	if err != nil {
		web.RespondError(w, http.StatusBadRequest, "Unable to parse account id")
		return
	}

	acc, err := account.SelectById(a.DB, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, http.StatusNotFound, fmt.Sprintf("%d account id is not found", id))
			return
		}

		web.RespondError(w, http.StatusInternalServerError, "Unable to find account")
		return
	}

	web.Respond(w, http.StatusOK, acc)
}

func (a *Application) FindAll(w http.ResponseWriter, _ *http.Request) {
	accounts, err := account.SelectAll(a.DB)
	if err != nil {
		web.RespondError(w, http.StatusInternalServerError, "Unable to retrieve accounts")
		return
	}

	if len(accounts) == 0 {
		accounts = make([]account.Account, 0)
	}

	web.Respond(w, http.StatusOK, accounts)
}

func (a *Application) CreateAccountForCustomer(w http.ResponseWriter, r *http.Request) {
	var payload account.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		web.RespondError(w, http.StatusBadRequest, "Unable to unmarshal customer request payload")
		return
	}
	defer r.Body.Close()

	// custom validation
	if payload.FirstName == "" || payload.LastName == "" {
		web.RespondError(w, http.StatusBadRequest, "firstName and lastName are required fields")
		return
	}
	if !payload.Currency.Supported() {
		web.RespondError(w, http.StatusBadRequest, fmt.Sprintf("%s currency not supported", payload.Currency))
		return
	}

	// customer creation
	c, err := customer.Create(a.DB, payload)
	if err != nil {
		if pgErr, ok := errors.Cause(err).(*pq.Error); ok {
			if string(pgErr.Code) == db.PSQLErrUniqueConstraint {
				web.RespondError(w, http.StatusBadRequest, fmt.Sprintf("%s taken, specify another one", payload.Email))
				return
			}
		}
		web.RespondError(w, http.StatusInternalServerError, "Unable to insert customer")
		return
	}

	// account creation
	acc, err := account.Create(a.DB, c.ID, payload)
	if err != nil {
		web.RespondError(w, http.StatusInternalServerError, "Unable to insert account")
	}

	web.Respond(w, http.StatusCreated, acc)
}
