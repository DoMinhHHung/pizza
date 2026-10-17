ALTER TABLE users
    ADD COLUMN username VARCHAR(30);

UPDATE users
SET username = REGEXP_REPLACE(LOWER(SUBSTRING(email, 1, POSITION('@' IN email) - 1)), '[^a-z0-9_]', '', 'g')
WHERE username IS NULL;

UPDATE users u
SET username = u.username || '_' || SUBSTRING(REPLACE(CAST(u.id AS VARCHAR), '-', ''), 1, 6)
WHERE u.id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY username ORDER BY created_at) AS rn
        FROM users
    ) t WHERE rn > 1
);

ALTER TABLE users
    ALTER COLUMN username SET NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT uq_users_username UNIQUE (username);

CREATE INDEX idx_users_username ON users (username);
