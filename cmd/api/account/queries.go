package account

const (
	selectById = "SELECT id, customer_id, balance, currency, created_at, frozen FROM accounts WHERE id=$1;"
	selectAll  = "SELECT * FROM accounts;"
	//insertCustomer = "INSERT INTO customers(first_name, last_name) VALUES($1,$2) RETURNING id;"
	//insert     = "INSERT INTO accounts(customer_id, balance, currency, created_at) VALUES($1,$2,$3,$4) RETURNING id;"
	//deleteById = "DELETE FROM accounts WHERE id=$1 RETURNING id;"
	//freezeById = "UPDATE accounts SET frozen = TRUE WHERE id=$1 RETURNING id;"
)
