-- +goose Up
-- +goose StatementBegin
-- Групповые
CREATE INDEX IF NOT EXISTS idx_group_trainer         ON "Групповая_тренировка"("id_тренера");
CREATE INDEX IF NOT EXISTS idx_group_zone            ON "Групповая_тренировка"("id_зоны");
CREATE INDEX IF NOT EXISTS idx_group_starts_at       ON "Групповая_тренировка"("Время_начала");
CREATE INDEX IF NOT EXISTS idx_group_ends_at         ON "Групповая_тренировка"("Время_окончания");

-- Персональные
CREATE INDEX IF NOT EXISTS idx_personal_sub          ON "Персональная_тренировка"("id_абонемента");
CREATE INDEX IF NOT EXISTS idx_personal_trainer      ON "Персональная_тренировка"("id_тренера");
CREATE INDEX IF NOT EXISTS idx_personal_starts_at    ON "Персональная_тренировка"("Время_начала");

-- Записи на групповые
CREATE INDEX IF NOT EXISTS idx_enroll_group          ON "Запись_на_групповую_тренировку"("id_групповой_тренировки");
CREATE INDEX IF NOT EXISTS idx_enroll_subscription   ON "Запись_на_групповую_тренировку"("id_абонемента");

-- Клиенты (для поиска)
CREATE INDEX IF NOT EXISTS idx_client_fio_low        ON "Клиент"(LOWER("ФИО"));
CREATE INDEX IF NOT EXISTS idx_client_phone          ON "Клиент"("Номер_телефона");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_client_phone;
DROP INDEX IF EXISTS idx_client_fio_low;

DROP INDEX IF EXISTS idx_enroll_subscription;
DROP INDEX IF EXISTS idx_enroll_group;

DROP INDEX IF EXISTS idx_personal_starts_at;
DROP INDEX IF EXISTS idx_personal_trainer;
DROP INDEX IF EXISTS idx_personal_sub;

DROP INDEX IF EXISTS idx_group_ends_at;
DROP INDEX IF EXISTS idx_group_starts_at;
DROP INDEX IF EXISTS idx_group_zone;
DROP INDEX IF EXISTS idx_group_trainer;
-- +goose StatementEnd
