-- UserBalances Table Creation
CREATE TABLE user_balances (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current NUMERIC(10,2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(10,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);