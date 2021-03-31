package audit

const (
	insert = "INSERT INTO transactions(account_id, ack, created_at) VALUES($1,$2,$3) RETURNING id;"
)
