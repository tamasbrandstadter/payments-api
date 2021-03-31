CREATE TABLE customers
(
    id          SERIAL PRIMARY KEY,
    first_name  VARCHAR(25) NOT NULL,
    last_name   VARCHAR(25) NOT NULL,
    email       VARCHAR(25) UNIQUE,
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() at time zone 'utc'),
    modified_at TIMESTAMP WITHOUT TIME ZONE
);

CREATE TABLE accounts
(
    id          SERIAL PRIMARY KEY,
    customer_id SERIAL,
    CONSTRAINT fk_customer
        FOREIGN KEY (customer_id)
            REFERENCES customers (id),
    currency    VARCHAR(3) NOT NULL,
    balance     DECIMAL    NOT NULL,
    frozen      BOOLEAN                     DEFAULT FALSE,
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() at time zone 'utc'),
    modified_at TIMESTAMP WITHOUT TIME ZONE
);

CREATE TABLE transactions
(
    id         SERIAL PRIMARY KEY,
    account_id SERIAL,
    CONSTRAINT fk_account
        FOREIGN KEY (account_id)
            REFERENCES accounts (id),
    ack        BOOLEAN,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() at time zone 'utc')
)