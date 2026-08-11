CREATE TABLE avatars (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    avatar_link   TEXT NOT NULL,
    add_date_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_avatars_user_id FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX idx_avatars_user_id ON avatars (user_id);
