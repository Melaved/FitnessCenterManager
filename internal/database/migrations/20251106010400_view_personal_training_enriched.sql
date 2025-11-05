-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW personal_training_enriched AS
SELECT
    p."id_персональной_тренировки"       AS personal_id,
    p."id_абонемента"                    AS subscription_id,
    p."id_тренера"                        AS trainer_id,
    p."Время_начала"                     AS starts_at,
    p."Время_окончания"                  AS ends_at,
    p."Статус"                           AS status,
    p."Стоимость"                        AS price,

    -- подстановочные
    c."ФИО"                               AS client_fio,
    t."ФИО"                               AS trainer_fio,

    -- вычисляемые
    EXTRACT(EPOCH FROM (p."Время_окончания" - p."Время_начала"))/60.0 AS duration_minutes,
    (p."Статус" = 'Запланирована' AND p."Время_окончания" < NOW())    AS is_overdue
FROM "Персональная_тренировка" p
JOIN "Абонемент" a ON a."id_абонемента" = p."id_абонемента"
JOIN "Клиент"   c ON c."id_клиента"     = a."id_клиента"
JOIN "Тренер"   t ON t."id_тренера"     = p."id_тренера";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS personal_training_enriched;
-- +goose StatementEnd
