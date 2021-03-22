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

	app.handler = router
	return &app
}
