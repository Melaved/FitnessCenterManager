-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW group_training_enriched AS
SELECT
    g."id_групповой_тренировки"        AS group_id,
    g."Название"                        AS title,
    g."Описание"                        AS description,
    g."Максимум_участников"            AS max_participants,
    g."Уровень_сложности"              AS level,
    g."Время_начала"                   AS starts_at,
    g."Время_окончания"                AS ends_at,

    -- подстановочные
    t."ФИО"                             AS trainer_name,
    z."Название"                        AS zone_name,

    -- вычисляемые
    EXTRACT(EPOCH FROM (g."Время_окончания" - g."Время_начала"))/60.0 AS duration_minutes,
    (g."Время_начала" > NOW())                                        AS is_upcoming
FROM "Групповая_тренировка" g
JOIN "Тренер" t ON t."id_тренера" = g."id_тренера"
JOIN "Зона"   z ON z."id_зоны"    = g."id_зоны";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS group_training_enriched;
-- +goose StatementEnd
