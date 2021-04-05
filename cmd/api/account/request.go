package account

type AccCreationRequest struct {
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	Email          string `json:"email"`
	InitialBalance int64  `json:"balance"`
	Currency       string `json:"currency"`
}
