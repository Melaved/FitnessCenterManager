-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW public.v_personal_training_enriched AS
SELECT
  p."id_персональной_тренировки"       AS id,
  p."id_абонемента"                    AS subscription_id,
  c."ФИО"                              AS client_fio,     -- lookup через абонемент -> клиент
  p."id_тренера"                       AS trainer_id,
  tr."ФИО"                             AS trainer_fio,    -- lookup
  p."Время_начала"                     AS starts_at,
  p."Время_окончания"                  AS ends_at,
  p."Статус"                           AS status,
  COALESCE(p."Стоимость", 0)::numeric  AS price_effective, -- вычисляемая (coalesce)
  EXTRACT(EPOCH FROM (p."Время_окончания" - p."Время_начала"))/60::int AS duration_minutes, -- вычисляемая
  (p."Время_начала" >= NOW())          AS is_upcoming,    -- вычисляемая
  (p."Время_начала" >= NOW() - INTERVAL '30 days') AS is_recent, -- вычисляемая
  CASE p."Статус"
    WHEN 'Запланирована' THEN 'primary'
    WHEN 'Завершена'     THEN 'success'
    ELSE 'secondary'
  END                                   AS status_badge    -- подстановочная (для UI)
FROM public."Персональная_тренировка" p
JOIN public."Абонемент" a ON a."id_абонемента" = p."id_абонемента"
JOIN public."Клиент"    c ON c."id_клиента"    = a."id_клиента"
JOIN public."Тренер"   tr ON tr."id_тренера"   = p."id_тренера";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS public.v_personal_training_enriched;
-- +goose StatementEnd
