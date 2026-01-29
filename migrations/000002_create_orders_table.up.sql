-- Orders Table Creation
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    accrual NUMERIC(10,2),
    uploaded_at TIMESTAMP NOT NULL DEFAULT now()
);
