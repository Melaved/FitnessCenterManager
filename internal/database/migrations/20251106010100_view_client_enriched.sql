-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW client_enriched AS
SELECT
    c."id_клиента"                AS client_id,
    c."ФИО"                       AS fio,
    c."Номер_телефона"            AS phone,
    c."Дата_рождения"             AS birth_date,
    c."Дата_регистрации"          AS register_date,
    c."Медицинские_данные"        AS medical_data,

    -- вычисляемые
    DATE_PART('year', AGE(CURRENT_DATE, c."Дата_рождения"))::int AS age_years,
    (CURRENT_DATE - c."Дата_регистрации")::int                    AS registered_days_ago,
    REGEXP_REPLACE(c."Номер_телефона", '\D', '', 'g')            AS phone_clean
FROM "Клиент" c;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS client_enriched;
-- +goose StatementEnd
