CREATE TYPE txtype AS ENUM ('deposit', 'withdraw', 'transfer');

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
    id               SERIAL PRIMARY KEY,
    from_id          INTEGER,
    CONSTRAINT fk_account
        FOREIGN KEY (from_id)
            REFERENCES accounts (id),
    to_id            INTEGER,
    transaction_type txtype NOT NULL,
    ack              BOOLEAN                     DEFAULT TRUE,
    created_at       TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() at time zone 'utc')
)