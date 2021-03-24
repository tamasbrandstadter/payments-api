package handlers

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
)

const (
	accounts      = "/accounts"
	accountById   = "/accounts/:id"
	freezeAccount = "/accounts/:id/freeze"
	deposit       = "/accounts/:id/deposit"
	withdraw      = "/accounts/:id/withdraw"
)

type Application struct {
	DB      *sqlx.DB
	handler http.Handler
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func NewApplication(db *sqlx.DB) *Application {
	app := Application{
		DB: db,
	}

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, accountById, app.GetAccountById)
	router.HandlerFunc(http.MethodGet, accounts, app.FindAllAccounts)
	router.HandlerFunc(http.MethodPost, accounts, app.CreateAccountForCustomer)
	router.HandlerFunc(http.MethodDelete, accountById, app.DeleteAccountById)
	router.HandlerFunc(http.MethodPut, freezeAccount, app.Freeze)
	router.HandlerFunc(http.MethodPut, deposit, app.Deposit)
	router.HandlerFunc(http.MethodPut, withdraw, app.Withdraw)

	app.handler = router
	return &app
}
