UPDATE balance_transactions SET kind = 'admin_adjustment' WHERE kind = 'game_charge';

ALTER TYPE balance_tx_kind RENAME TO balance_tx_kind_old;
CREATE TYPE balance_tx_kind AS ENUM ('receipt_credit', 'receipt_reversal', 'admin_adjustment');
ALTER TABLE balance_transactions
    ALTER COLUMN kind TYPE balance_tx_kind USING kind::text::balance_tx_kind;
DROP TYPE balance_tx_kind_old;
