CREATE TABLE users (
    id                  UUID            PRIMARY KEY,
    email               VARCHAR(254)    NOT NULL,
    password_hash       VARCHAR(255)    NOT NULL,
    full_name           VARCHAR(100)    NOT NULL,
    status              VARCHAR(30)     NOT NULL,
    email_verified_at   TIMESTAMP WITH TIME ZONE,
    locked_until        TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE     NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE     NOT NULL,

    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE INDEX idx_users_status ON users (status);
