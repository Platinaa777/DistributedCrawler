-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
    ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'READ';

UPDATE users
SET role = 'READ'
WHERE role IS NULL;

ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('READ', 'READ_WRITE', 'ADMINISTRATOR'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- +goose StatementEnd
