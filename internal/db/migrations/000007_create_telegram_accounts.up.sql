CREATE TABLE telegram_accounts (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    telegram_id BIGINT NOT NULL,
    username    VARCHAR(255),
    first_name  VARCHAR(255),
    last_name   VARCHAR(255),
    photo_url   TEXT,
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_telegram_accounts_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX idx_telegram_accounts_user_id ON telegram_accounts (user_id);
CREATE UNIQUE INDEX idx_telegram_accounts_telegram_id ON telegram_accounts (telegram_id);
