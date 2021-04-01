package balance

type BalanceMessage struct {
	AccountID int     `json:"id"`
	Amount    float64 `json:"amount"`
}

type TransferMessage struct {
	FromID int     `json:"from"`
	ToID   int     `json:"to"`
	Amount float64 `json:"amount"`
}
