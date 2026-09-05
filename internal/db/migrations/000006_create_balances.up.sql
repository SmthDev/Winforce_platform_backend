CREATE TABLE balances (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL DEFAULT 0,
    currency     VARCHAR(8) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_balances_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX idx_balances_user_id ON balances (user_id);

CREATE TYPE balance_tx_kind AS ENUM ('receipt_credit', 'receipt_reversal', 'admin_adjustment');

CREATE TABLE balance_transactions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL,
    currency     VARCHAR(8) NOT NULL,
    kind         balance_tx_kind NOT NULL,
    receipt_id   BIGINT REFERENCES receipts (id) ON DELETE SET NULL,
    comment      TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_balance_transactions_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX idx_balance_transactions_user_id ON balance_transactions (user_id, created_at DESC, id DESC);
