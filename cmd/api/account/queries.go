package account

const (
	selectById     = "SELECT id, customer_id, balance, currency, created_at, modified_at, frozen FROM accounts WHERE id=$1;"
	selectTwoById  = "SELECT id, balance FROM accounts WHERE id=$1 OR id=$2"
	selectAll      = "SELECT * FROM accounts;"
	insert         = "INSERT INTO accounts(customer_id, balance, currency, created_at, modified_at) VALUES($1,$2,$3,$4,$5) RETURNING id;"
	deleteById     = "DELETE FROM accounts WHERE id=$1;"
	freezeById     = "UPDATE accounts SET frozen = TRUE, modified_at=$1 WHERE id=$2;"
	updateBalance  = "UPDATE accounts SET balance=$1, modified_at=$2 WHERE id=$3;"
	updateBalances = "UPDATE accounts as u SET balance = u2.balance, modified_at = u2.modified_at FROM" +
		" (values ($1::integer, $2::decimal, $3::timestamp), ($4::integer, $5::decimal, $6::timestamp)) " +
		"as u2(id, balance, modified_at) WHERE u2.id = u.id;"
)
