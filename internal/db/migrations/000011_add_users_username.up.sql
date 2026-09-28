ALTER TABLE users ADD COLUMN username VARCHAR(255);

CREATE UNIQUE INDEX idx_users_username ON users (username);
