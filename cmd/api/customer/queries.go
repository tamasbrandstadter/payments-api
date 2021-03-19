package customer

const (
	insert = "INSERT INTO customers(first_name, last_name, email, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;"
)
