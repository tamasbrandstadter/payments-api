package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Rhymond/go-money"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/web"
)

func (a *Application) GetBalance(w http.ResponseWriter, r *http.Request) {
	id := httprouter.ParamsFromContext(r.Context()).ByName("id")

	if len(id) == 0 {
		web.RespondError(w, http.StatusBadRequest, "account id is missing")
		return
	}

	// get balance from cache
	var b []byte
	if err := a.Cache.Balances.Get(context.Background(), id, &b); err != nil {
		log.Warnf("failed to get balance from cache for accound id %s", id)
	} else {
		var m money.Money
		if err = m.UnmarshalJSON(b); err == nil {
			web.Respond(w, http.StatusOK, map[string]string{"balance": m.Display()})
			return
		}
	}

	// not in cache, find in db
	accId, _ := strconv.Atoi(httprouter.ParamsFromContext(r.Context()).ByName("id"))
	acc, err := account.SelectById(a.DB, accId)

	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			web.RespondError(w, http.StatusNotFound, fmt.Sprintf("account id %d is not found", accId))
			return
		}

		web.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("unable to find account: %s", err.Error()))
		return
	}

	m := money.New(acc.BalanceInDecimal, acc.Currency)

	web.Respond(w, http.StatusOK, map[string]string{"balance": m.Display()})
}
