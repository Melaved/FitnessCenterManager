-- +goose Up
-- +goose StatementBegin
-- Базовые права на схему и последовательности
GRANT USAGE ON SCHEMA public TO app_user;
GRANT CREATE ON SCHEMA public TO app_user;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT USAGE, SELECT ON SEQUENCES TO app_user;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
REVOKE USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public FROM app_user;
REVOKE CREATE ON SCHEMA public FROM app_user;
REVOKE USAGE ON SCHEMA public FROM app_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
REVOKE USAGE, SELECT ON SEQUENCES FROM app_user;
-- +goose StatementEnd
