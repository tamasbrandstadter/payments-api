package balance

type BalanceMessage struct {
	AccountID int   `json:"id"`
	Amount    int64 `json:"amount"`
}

type TransferMessage struct {
	FromID int   `json:"from"`
	ToID   int   `json:"to"`
	Amount int64 `json:"amount"`
}
