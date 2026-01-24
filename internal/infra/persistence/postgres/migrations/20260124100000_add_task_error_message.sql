-- +goose Up
-- +goose StatementBegin
ALTER TABLE crawl_tasks ADD COLUMN error_message TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crawl_tasks DROP COLUMN error_message;
-- +goose StatementEnd
