package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/web"
)

func (a *Application) GetAccountById(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	id, err := strconv.Atoi(params["id"])
	if err != nil {
		web.RespondError(w, r, http.StatusBadRequest, errors.Wrap(err, "unable to parse acc id"))
		return
	}

	acc, err := account.SelectById(a.DB, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, r, http.StatusNotFound, errors.New(http.StatusText(http.StatusNotFound)))
			return
		}

		web.RespondError(w, r, http.StatusInternalServerError, errors.Wrap(err, "select acc by id"))
		return
	}

	web.Respond(w, r, http.StatusOK, acc)
}

func (a *Application) FindAll(w http.ResponseWriter, r *http.Request) {
	accounts, err := account.SelectAll(a.DB)
	if err != nil {
		web.RespondError(w, r, http.StatusInternalServerError, errors.Wrap(err, "select all lists"))
		return
	}

	if len(accounts) == 0 {
		accounts = make([]account.Account, 0)
	}

	web.Respond(w, r, http.StatusOK, accounts)
}
