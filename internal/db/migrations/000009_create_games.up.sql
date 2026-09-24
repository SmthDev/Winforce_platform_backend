
CREATE TABLE game_settings (
    id                  SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    charge_amount_minor BIGINT NOT NULL CHECK (charge_amount_minor > 0),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO game_settings (id, charge_amount_minor) VALUES (1, 140);

CREATE TABLE games (
    id         BIGSERIAL PRIMARY KEY,
    played_on  DATE NOT NULL,
    opponent   VARCHAR(255) NOT NULL,
    charged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_games_played_on ON games (played_on DESC, id DESC);

ALTER TABLE balance_transactions
    ADD COLUMN game_id BIGINT REFERENCES games (id) ON DELETE SET NULL;

CREATE INDEX idx_balance_transactions_game_id ON balance_transactions (game_id) WHERE game_id IS NOT NULL;
