-- Orders Table Creation
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number VARCHAR(1024) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(255) NOT NULL,
    accrual NUMERIC(10,2),
    uploaded_at TIMESTAMP NOT NULL DEFAULT now()
);

-- UserBalances Table Creation
CREATE TABLE user_balances (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current NUMERIC(10,2) NOT NULL DEFAULT 0,
    withdrawn NUMERIC(10,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
