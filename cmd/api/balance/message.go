package balance

type TxMessage struct {
	AccountID int     `json:"id"`
	Amount    float64 `json:"amount"`
}
