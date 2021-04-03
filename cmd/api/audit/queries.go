package audit

const (
	insert = "INSERT INTO transactions(from_id, to_id, transaction_type, ack, created_at) VALUES($1,$2,$3,$4,$5) RETURNING id;"
)
