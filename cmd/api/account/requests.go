package account

type AccCreationRequest struct {
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Email          string   `json:"email"`
	InitialBalance float64  `json:"balance"`
	Currency       Currency `json:"currency"`
}

type BalanceOperationRequest struct {
	AccountID int     `json:"id"`
	Amount    float64 `json:"amount"`
}
