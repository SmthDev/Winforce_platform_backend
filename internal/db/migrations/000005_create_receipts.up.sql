CREATE TYPE receipt_status AS ENUM ('credited', 'rejected');

CREATE TABLE receipts (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    status        receipt_status NOT NULL DEFAULT 'credited',
    file_object   TEXT NOT NULL,
    file_hash     CHAR(64) NOT NULL,
    check_number  TEXT,
    amount_minor  BIGINT NOT NULL,
    currency      VARCHAR(8) NOT NULL,
    check_date    TEXT,
    bank          TEXT,
    card          TEXT,
    payer         TEXT,
    service       TEXT,
    account       TEXT,
    recipient     TEXT,
    raw_response  JSONB NOT NULL,
    add_date_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_receipts_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX idx_receipts_user_id ON receipts (user_id, add_date_time DESC);
CREATE UNIQUE INDEX idx_receipts_file_hash ON receipts (file_hash);
CREATE UNIQUE INDEX idx_receipts_check_number ON receipts (check_number)
    WHERE check_number IS NOT NULL AND check_number <> '';
