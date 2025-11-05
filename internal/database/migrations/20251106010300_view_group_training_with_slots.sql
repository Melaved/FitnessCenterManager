-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW group_training_with_slots AS
WITH enrolls AS (
  SELECT 
    e."id_групповой_тренировки" AS group_id,
    COUNT(*) FILTER (WHERE e."Статус" IN ('Записан','Посетил')) AS enrolled_count
  FROM "Запись_на_групповую_тренировку" e
  GROUP BY e."id_групповой_тренировки"
)
SELECT
  ge.*,
  COALESCE(en.enrolled_count, 0)                                       AS enrolled_count,
  GREATEST(ge.max_participants - COALESCE(en.enrolled_count, 0), 0)    AS free_slots
FROM group_training_enriched ge
LEFT JOIN enrolls en ON en.group_id = ge.group_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS group_training_with_slots;
-- +goose StatementEnd
