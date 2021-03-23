package account

const (
	selectById = "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=$1;"
	selectAll  = "SELECT * FROM accounts;"
	insert     = "INSERT INTO accounts(customer_id, balance, currency, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;"
	deleteById = "DELETE FROM accounts WHERE id=$1;"
	freezeById = "UPDATE accounts SET frozen = TRUE, modified_at=$1 WHERE id=$2;"
	deposit    = "UPDATE accounts SET balance=$1, modified_at=$2 WHERE id=$3;"
)
