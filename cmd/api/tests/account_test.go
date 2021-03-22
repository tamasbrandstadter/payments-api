package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tamasbrandstadter/payments-api/cmd/api/account"
	"github.com/tamasbrandstadter/payments-api/internal/testdb"
)

var expectedAcc = &account.Account{
	ID:         1,
	CustomerID: 1,
	Balance:    999,
	Currency:   "EUR",
	CreatedAt:  testdb.TestTime,
	ModifiedAt: testdb.TestTime,
	Frozen:     false,
}

func TestAccount_SelectById(t *testing.T) {
	err := testdb.SaveCustomerWithAccount(a.DB)
	if err != nil {
		t.Errorf("error creating test customer with account: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", 1), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if e, a := http.StatusOK, w.Code; e != a {
		t.Errorf("expected status code: %v, got status code: %v", e, a)
	}

	if expectedAcc != nil {
		var actualAccount account.Account
		if err := json.NewDecoder(w.Body).Decode(&actualAccount); err != nil {
			t.Errorf("error decoding response body: %v", err)
		}

		if d := cmp.Diff(*expectedAcc, actualAccount); d != "" {
			t.Errorf("unexpected difference in response body:\n%v", d)
		}
	}
}

func TestAccount_SelectById_NotFound(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", 2), nil)
	if err != nil {
		t.Errorf("error creating request: %v", err)
	}

	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)

	if e, a := http.StatusNotFound, w.Code; e != a {
		t.Errorf("expected status code: %v, got status vode: %v", e, a)
	}
}
