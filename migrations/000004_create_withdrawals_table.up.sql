-- Withdrawals Table Creation
CREATE TABLE withdrawals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    order_number TEXT NOT NULL,
    sum NUMERIC(10,2),
    processed_at TIMESTAMP NOT NULL DEFAULT now()
);


