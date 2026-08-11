CREATE TABLE qr_codes (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    target_link   TEXT NOT NULL,
    qr_link       TEXT NOT NULL,
    add_date_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_qr_codes_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX idx_qr_codes_user_id ON qr_codes (user_id);
