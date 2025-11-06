-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW public.view_client_enriched AS
WITH last_sub AS (
  SELECT
    a."id_клиента",
    a."id_абонемента",
    a."id_тарифа",
    a."Дата_начала",
    a."Дата_окончания",
    a."Статус",
    t."Название_тарифа",
    ROW_NUMBER() OVER (
      PARTITION BY a."id_клиента"
      ORDER BY a."Дата_начала" DESC, a."id_абонемента" DESC
    ) AS rn
  FROM public."Абонемент" a
  JOIN public."Тариф" t ON t."id_тарифа" = a."id_тарифа"
),
stat AS (
  SELECT
    a."id_клиента",
    COUNT(*)                                          AS subs_total,
    COUNT(*) FILTER (WHERE a."Статус" = 'Активен')    AS subs_active,
    MAX(a."Дата_окончания")                           AS subs_last_end
  FROM public."Абонемент" a
  GROUP BY a."id_клиента"
)
SELECT
  c."id_клиента",
  c."ФИО",
  c."Номер_телефона",
  c."Дата_рождения",
  c."Дата_регистрации",
  c."Медицинские_данные",

  -- вычисляемые колонки
  DATE_PART('year', age(c."Дата_рождения"))::int      AS возраст,
  (CURRENT_DATE - c."Дата_регистрации")               AS дней_с_регистрации,
  COALESCE(s.subs_total, 0)                           AS всего_абонементов,
  COALESCE(s.subs_active, 0)                          AS активных_абонементов,

  -- подстановочные из последнего абонемента
  ls."id_абонемента"                                  AS последний_абонемент_id,
  ls."Название_тарифа"                                AS последний_тариф,
  ls."Дата_начала"                                    AS последний_абонемент_начало,
  ls."Дата_окончания"                                 AS последний_абонемент_конец,
  ls."Статус"                                         AS статус_последнего_абонемента
FROM public."Клиент" c
LEFT JOIN stat     s  ON s."id_клиента" = c."id_клиента"
LEFT JOIN last_sub ls ON ls."id_клиента" = c."id_клиента" AND ls.rn = 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS public.view_client_enriched;
-- +goose StatementEnd
