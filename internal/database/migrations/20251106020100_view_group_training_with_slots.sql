-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW public.v_group_training_with_slots AS
SELECT
  g."id_групповой_тренировки"           AS id,
  g."Название"                          AS title,
  g."Описание"                          AS description,
  g."Уровень_сложности"                 AS level,
  g."Максимум_участников"               AS max,
  g."Время_начала"                      AS starts_at,
  g."Время_окончания"                   AS ends_at,
  g."id_тренера"                        AS trainer_id,
  t."ФИО"                               AS trainer_name,         -- lookup
  g."id_зоны"                           AS zone_id,
  z."Название"                          AS zone_name,            -- lookup

  -- вычисляемые
  EXTRACT(EPOCH FROM (g."Время_окончания" - g."Время_начала"))/60::int AS duration_minutes,
  COALESCE(e.enrolled_count, 0)                                    AS enrolled_count,
  GREATEST(g."Максимум_участников" - COALESCE(e.enrolled_count,0),0) AS slots_left,
  (g."Время_начала" >= NOW())                                      AS is_upcoming,
  (g."Время_начала" >= NOW() - INTERVAL '30 days')                 AS is_recent,
  CASE
    WHEN NOW() BETWEEN g."Время_начала" AND g."Время_окончания" THEN 'Идёт'
    WHEN g."Время_окончания" < NOW() THEN 'Прошла'
    ELSE 'Будет'
  END                                                              AS status_time,      -- подстановочная
  ROUND( CASE WHEN g."Максимум_участников" > 0
              THEN 100.0 * COALESCE(e.enrolled_count,0) / g."Максимум_участников"
              ELSE 0 END, 0) :: int                                AS capacity_usage_pct
FROM public."Групповая_тренировка" g
JOIN public."Тренер"   t ON t."id_тренера" = g."id_тренера"
JOIN public."Зона"     z ON z."id_зоны"    = g."id_зоны"
LEFT JOIN (
  SELECT "id_групповой_тренировки", COUNT(*) AS enrolled_count
  FROM public."Запись_на_групповую_тренировку"
  GROUP BY "id_групповой_тренировки"
) e ON e."id_групповой_тренировки" = g."id_групповой_тренировки";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS public.v_group_training_with_slots;
-- +goose StatementEnd
