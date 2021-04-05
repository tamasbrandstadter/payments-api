package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/cmd/api/customer"
	"github.com/tamasbrandstadter/payments-api/internal/db"
	"github.com/tamasbrandstadter/payments-api/internal/web"
)

func (a *Application) GetAccountById(w http.ResponseWriter, r *http.Request) {
	// request validation
	id, err := strconv.Atoi(httprouter.ParamsFromContext(r.Context()).ByName("id"))
	if err != nil {
		web.RespondError(w, http.StatusBadRequest, "unable to parse account id")
		return
	}

	acc, err := account.SelectById(a.DB, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, http.StatusNotFound, fmt.Sprintf("account id %d is not found", id))
			return
		}

		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to find account: %s", err.Error()))
		return
	}

	web.Respond(w, http.StatusOK, acc)
}

func (a *Application) FindAllAccounts(w http.ResponseWriter, _ *http.Request) {
	accounts, err := account.SelectAll(a.DB)
	if err != nil {
		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to retrieve accounts: %s", err.Error()))
		return
	}

	if len(accounts) == 0 {
		accounts = make([]account.Account, 0)
	}

	web.Respond(w, http.StatusOK, accounts)
}

func (a *Application) CreateAccountForCustomer(w http.ResponseWriter, r *http.Request) {
	// request validation
	var payload account.AccCreationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		web.RespondError(w, http.StatusBadRequest, "invalid request payload, unable to parse")
		return
	}
	defer r.Body.Close()

	// custom validation
	if payload.FirstName == "" || payload.LastName == "" {
		web.RespondError(w, http.StatusBadRequest, "firstname and lastname are required fields")
		return
	}
	if payload.InitialBalance < 0 {
		web.RespondError(w, http.StatusBadRequest, "initial deposit can't be negative")
		return
	}

	// customer creation
	c, err := customer.Create(a.DB, payload)
	if err != nil {
		if pgErr, ok := errors.Cause(err).(*pq.Error); ok {
			if string(pgErr.Code) == db.PSQLErrUniqueConstraint {
				web.RespondError(w, http.StatusConflict, fmt.Sprintf("%s is taken, specify another one", payload.Email))
				return
			}
		}
		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to insert customer: %s", err.Error()))
		return
	}

	// account creation
	acc, err := account.Create(a.DB, c.ID, payload)
	if err != nil {
		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to insert account: %s", err.Error()))
	}

	web.Respond(w, http.StatusCreated, acc)
}

func (a *Application) DeleteAccountById(w http.ResponseWriter, r *http.Request) {
	// request validation
	id, err := strconv.Atoi(httprouter.ParamsFromContext(r.Context()).ByName("id"))
	if err != nil {
		web.RespondError(w, http.StatusBadRequest, "unable to parse account id")
		return
	}

	if err = account.Delete(a.DB, id); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, http.StatusNotFound, fmt.Sprintf("account id %d is not found", id))
			return
		}

		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to delete account: %s", err.Error()))
		return
	}

	web.Respond(w, http.StatusNoContent, nil)
}

func (a *Application) Freeze(w http.ResponseWriter, r *http.Request) {
	// request validation
	id, err := strconv.Atoi(httprouter.ParamsFromContext(r.Context()).ByName("id"))
	if err != nil {
		web.RespondError(w, http.StatusBadRequest, "unable to parse account id")
		return
	}

	acc, err := account.Freeze(a.DB, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, http.StatusNotFound, fmt.Sprintf("account id %d is not found", id))
			return
		}

		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to freeze account: %s", err.Error()))
		return
	}

	web.Respond(w, http.StatusOK, acc)
}
