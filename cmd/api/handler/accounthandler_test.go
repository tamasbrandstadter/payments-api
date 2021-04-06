package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
)

func TestGetAccountById(t *testing.T) {
	r := account.AccCreationRequest{
		FirstName:      "first15",
		LastName:       "last15",
		Email:          "test15@test15.com",
		InitialBalance: 999,
		Currency:       "EUR",
	}

	err := testdb.SaveCustomerWithAccount(a.DB, r)
	if err != nil {
		t.Errorf("error creating test customer with account: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", 1), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var expectedAcc = &account.Account{
		ID:               1,
		CustomerID:       1,
		BalanceInDecimal: 999,
		Currency:         "EUR",
		CreatedAt:        testdb.TestTime,
		ModifiedAt:       testdb.TestTime,
		Frozen:           false,
	}

	var actualAcc account.Account
	if err := json.NewDecoder(w.Body).Decode(&actualAcc); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	if diff := cmp.Diff(*expectedAcc, actualAcc); diff != "" {
		t.Errorf("unexpected difference in response body:\n%v", diff)
	}
}

func TestGetAccountByIdNotFound(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", 2), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusNotFound, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "account id 2 is not found", response["error"])
}

func TestGetAccountByIdWithInvalidId(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/accounts/textId", nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "unable to parse account id", response["error"])
}

func TestFindAllAccounts(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/accounts", nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var accounts []account.Account
	if err := json.NewDecoder(w.Body).Decode(&accounts); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Len(t, accounts, 1)
}

func TestCreateAccountForCustomer(t *testing.T) {
	payload := account.AccCreationRequest{
		FirstName:      "first",
		LastName:       "last",
		Email:          "first@last.com",
		InitialBalance: 15,
		Currency:       "GBP",
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Errorf("error encoding request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/accounts", &body)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusCreated, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var actualAcc account.Account
	if err := json.NewDecoder(w.Body).Decode(&actualAcc); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	expectedAcc := account.Account{
		ID:               2,
		CustomerID:       2,
		BalanceInDecimal: 15.0,
		Currency:         "GBP",
		Frozen:           false,
	}

	assert.Equal(t, expectedAcc.ID, actualAcc.ID)
	assert.Equal(t, expectedAcc.CustomerID, actualAcc.CustomerID)
	assert.Equal(t, expectedAcc.BalanceInDecimal, actualAcc.BalanceInDecimal)
	assert.Equal(t, expectedAcc.Currency, actualAcc.Currency)
	assert.False(t, actualAcc.Frozen)
	assert.NotNil(t, actualAcc.CreatedAt)
	assert.NotNil(t, actualAcc.ModifiedAt)
}

func TestCreateAccountForCustomerInvalidPayload(t *testing.T) {
	payload := "'"

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Errorf("error encoding request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/accounts", &body)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "invalid request payload, unable to parse", response["error"])
}

func TestCreateAccountForCustomerErrorInName(t *testing.T) {
	payload := account.AccCreationRequest{
		Email:          "first@last.com",
		InitialBalance: 15,
		Currency:       "GBP",
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Errorf("error encoding request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/accounts", &body)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "firstname and lastname are required fields", response["error"])
}

func TestCreateAccountForCustomerDuplicateEmail(t *testing.T) {
	payload := account.AccCreationRequest{
		FirstName:      "apple",
		LastName:       "pie",
		Email:          "first@last.com",
		InitialBalance: 15,
		Currency:       "GBP",
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Errorf("error encoding request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/accounts", &body)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusConflict, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "first@last.com is taken, specify another one", response["error"])
}

func TestCreateAccountForCustomerWithNegativeInitialBalance(t *testing.T) {
	payload := account.AccCreationRequest{
		FirstName:      "apple",
		LastName:       "pie",
		Email:          "second@second.com",
		InitialBalance: -1,
		Currency:       "GBP",
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Errorf("error encoding request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/accounts", &body)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "initial deposit can't be negative", response["error"])
}

func TestFindAllAccountsAfterCreation(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/accounts", nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var accounts []account.Account
	if err := json.NewDecoder(w.Body).Decode(&accounts); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Len(t, accounts, 2)

	_ = testdb.DeleteTestAccount(a.DB, 1)
}

func TestFreezeAccount(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d/freeze", 2), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var actualAcc account.Account
	if err := json.NewDecoder(w.Body).Decode(&actualAcc); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	expectedAcc := account.Account{
		ID:               2,
		CustomerID:       2,
		BalanceInDecimal: 15.0,
		Currency:         "GBP",
		Frozen:           true,
	}

	assert.Equal(t, expectedAcc.ID, actualAcc.ID)
	assert.Equal(t, expectedAcc.CustomerID, actualAcc.CustomerID)
	assert.Equal(t, expectedAcc.BalanceInDecimal, actualAcc.BalanceInDecimal)
	assert.Equal(t, expectedAcc.Currency, actualAcc.Currency)
	assert.True(t, actualAcc.Frozen)
	assert.NotNil(t, actualAcc.CreatedAt)
	assert.NotNil(t, actualAcc.ModifiedAt)
}

func TestFreezeAccountNotFound(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/accounts/%d/freeze", 77), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusNotFound, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "account id 77 is not found", response["error"])
}

func TestFreezeAccountInvalidId(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, "/accounts/textId/freeze", nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "unable to parse account id", response["error"])
}

func TestDeleteAccount(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%d", 2), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusNoContent, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}
}

func TestDeleteAccountNotFond(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/accounts/%d", 77), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusNotFound, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "account id 77 is not found", response["error"])
}

func TestDeleteAccountInvalidId(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, "/accounts/textId", nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusBadRequest, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Equal(t, "unable to parse account id", response["error"])
}

func TestFindAllAccountsEmpty(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts"), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	var accounts []account.Account
	if err := json.NewDecoder(w.Body).Decode(&accounts); err != nil {
		t.Errorf("error decoding response body: %v", err)
	}

	assert.Empty(t, accounts)
}

func TestDeposit(t *testing.T) {
	r := account.AccCreationRequest{
		FirstName:      "first1",
		LastName:       "last1",
		Email:          "test1@test1.com",
		InitialBalance: 999,
		Currency:       "EUR",
	}

	err := testdb.SaveCustomerWithAccount(a.DB, r)
	if err != nil {
		t.Errorf("error creating test customer with account: %v", err)
	}

	msg := []byte("{\"id\":3,\"amount\":1}")
	err = a.Conn.Channel.Publish("payments", "dep", false, false, amqp.Publishing{
		Body:         msg,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
	})

	go func() {
		a.Tc.StartConsuming(a.Conn, a.DB, nil)
	}()

	time.Sleep(time.Second / 2)

	acc, err := testdb.SelectById(a.DB, 3)
	if err != nil {
		t.Errorf("expected err nil, got %v", err)
	}
	assert.Equal(t, int64(1000), acc.BalanceInDecimal)
}

func TestWithdraw(t *testing.T) {
	msg := []byte("{\"id\":3,\"amount\":2}")
	err := a.Conn.Channel.Publish("payments", "wit", false, false, amqp.Publishing{
		Body:         msg,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
	})

	time.Sleep(time.Second / 2)

	acc, err := testdb.SelectById(a.DB, 3)
	if err != nil {
		t.Errorf("expected err nil, got %v", err)
	}
	assert.Equal(t, int64(998), acc.BalanceInDecimal)
}
