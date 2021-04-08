package handler

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"github.com/tamasbrandstadter/payments-api/internal/cache"
)

const (
	accounts           = "/accounts"
	accountById        = "/accounts/:id"
	freezeAccount      = "/accounts/:id/freeze"
	balanceByAccountId = "/accounts/:id/balance"
	health             = "/health"
)

type Application struct {
	DB      *sqlx.DB
	Cache   *cache.Redis
	handler http.Handler
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func NewApplication(db *sqlx.DB, r *cache.Redis) *Application {
	app := Application{
		DB:    db,
		Cache: r,
	}

	router := httprouter.New()

	// API routes
	router.HandlerFunc(http.MethodGet, accountById, app.GetAccountById)
	router.HandlerFunc(http.MethodGet, accounts, app.FindAllAccounts)
	router.HandlerFunc(http.MethodPost, accounts, app.CreateAccountForCustomer)
	router.HandlerFunc(http.MethodDelete, accountById, app.DeleteAccountById)
	router.HandlerFunc(http.MethodPut, freezeAccount, app.Freeze)
	router.HandlerFunc(http.MethodGet, balanceByAccountId, app.GetBalance)

	// K8s probes
	router.HandlerFunc(http.MethodGet, health, app.health)

	app.handler = router
	return &app
}

func (a *Application) health(w http.ResponseWriter, _ *http.Request) {
	if err := a.DB.Ping(); err == nil {
		// Ping by itself is un-reliable, the connections are cached. This
		// ensures that the database is still running by executing a harmless
		// dummy query against it.
		if _, err = a.DB.Exec("SELECT true"); err == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	w.WriteHeader(http.StatusInternalServerError)
}
