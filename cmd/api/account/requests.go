package account

type CreateAccountRequest struct {
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Email          string   `json:"email"`
	InitialBalance float64  `json:"balance"`
	Currency       Currency `json:"currency"`
}
