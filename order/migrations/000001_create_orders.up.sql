CREATE TABLE orders (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    amount     NUMERIC(20, 2) NOT NULL CHECK (amount >= 0),
    status     VARCHAR(50) NOT NULL
               CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
