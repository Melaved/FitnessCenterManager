-- +goose Up
-- +goose StatementBegin
DROP VIEW IF EXISTS public.view_client_enriched;

CREATE VIEW public.view_client_enriched AS
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
    COUNT(*)                                       AS subs_total,
    COUNT(*) FILTER (WHERE a."Статус" = 'Активен') AS subs_active,
    MAX(a."Дата_окончания")                        AS subs_last_end
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

  -- вычисляемые (англ. алиасы)
  DATE_PART('year', age(c."Дата_рождения"))::int AS age,
  (CURRENT_DATE - c."Дата_регистрации")         AS days_since_registration,
  COALESCE(s.subs_total, 0)                     AS subs_total,
  COALESCE(s.subs_active, 0)                    AS subs_active,

  -- из последнего абонемента (англ. алиасы)
  ls."id_абонемента"     AS last_subscription_id,
  ls."Название_тарифа"   AS last_tariff,
  ls."Дата_начала"       AS last_subscription_start,
  ls."Дата_окончания"    AS last_subscription_end,
  ls."Статус"            AS last_subscription_status
FROM public."Клиент" c
LEFT JOIN stat     s  ON s."id_клиента" = c."id_клиента"
LEFT JOIN last_sub ls ON ls."id_клиента" = c."id_клиента" AND ls.rn = 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS public.view_client_enriched;
-- +goose StatementEnd
