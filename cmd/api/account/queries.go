package account

const (
	selectById = "SELECT id, customer_id, balance_in_decimal, currency, created_at, modified_at, frozen " +
		"FROM accounts WHERE id=$1;"
	selectTwoById = "SELECT id, balance_in_decimal, currency FROM accounts WHERE id=$1 OR id=$2"
	selectAll     = "SELECT * FROM accounts;"
	insert        = "INSERT INTO accounts(customer_id, balance_in_decimal, currency, created_at, modified_at)" +
		" VALUES($1,$2,$3,$4,$5) RETURNING id;"
	deleteById     = "DELETE FROM accounts WHERE id=$1;"
	freezeById     = "UPDATE accounts SET frozen = TRUE, modified_at=$1 WHERE id=$2;"
	updateBalance  = "UPDATE accounts SET balance_in_decimal=$1, modified_at=$2 WHERE id=$3;"
	updateBalances = "UPDATE accounts as a SET balance_in_decimal = a2.balance_in_decimal, modified_at = a2.modified_at " +
		"FROM (values ($1::integer, $2::decimal, $3::timestamp), ($4::integer, $5::decimal, $6::timestamp)) " +
		"as a2(id, balance_in_decimal, modified_at) WHERE a2.id = a.id;"
)
