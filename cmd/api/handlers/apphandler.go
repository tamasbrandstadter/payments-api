package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

const (
	accounts      = "/accounts"
	accountById   = "/accounts/{id}"
	//freezeAccount = "/accounts/{id}/freeze"
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

	router := mux.NewRouter()
	router.HandleFunc(accountById, app.GetAccountById).Methods("GET")
	router.HandleFunc(accounts, app.FindAll).Methods("GET")
	//router.HandleFunc(accounts, accounts.CreateAccountForCustomer).Methods("POST")
	//router.HandleFunc(accountById, accounts.Delete).Methods("DELETE")
	//router.HandleFunc(freezeAccount, accounts.Freeze).Methods("PUT")

	app.handler = router
	return &app
}
