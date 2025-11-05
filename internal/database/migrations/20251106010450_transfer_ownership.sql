-- +goose Up
-- +goose StatementBegin
-- Таблицы
ALTER TABLE "Клиент"                         OWNER TO app_user;
ALTER TABLE "Тренер"                         OWNER TO app_user;
ALTER TABLE "Зона"                           OWNER TO app_user;
ALTER TABLE "Оборудование"                   OWNER TO app_user;
ALTER TABLE "Заявка_на_ремонт"               OWNER TO app_user;
ALTER TABLE "Тариф"                          OWNER TO app_user;
ALTER TABLE "Абонемент"                      OWNER TO app_user;
ALTER TABLE "Групповая_тренировка"           OWNER TO app_user;
ALTER TABLE "Персональная_тренировка"        OWNER TO app_user;
ALTER TABLE "Запись_на_групповую_тренировку" OWNER TO app_user;

-- Последовательности
ALTER SEQUENCE "Клиент_id_клиента_seq"                          OWNER TO app_user;
ALTER SEQUENCE "Тренер_id_тренера_seq"                          OWNER TO app_user;
ALTER SEQUENCE "Зона_id_зоны_seq"                               OWNER TO app_user;
ALTER SEQUENCE "Оборудование_id_оборудования_seq"               OWNER TO app_user;
ALTER SEQUENCE "Заявка_на_ремонт_id_заявки_seq"                 OWNER TO app_user;
ALTER SEQUENCE "Тариф_id_тарифа_seq"                            OWNER TO app_user;
ALTER SEQUENCE "Абонемент_id_абонемента_seq"                    OWNER TO app_user;
ALTER SEQUENCE "Групповая_трени_id_групповой_тре_seq"           OWNER TO app_user;
ALTER SEQUENCE "Персональная_тр_id_персональной__seq"           OWNER TO app_user;
ALTER SEQUENCE "Запись_на_групповую_тре_id_записи_seq"          OWNER TO app_user;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Клиент"                         OWNER TO postgres;
ALTER TABLE "Тренер"                         OWNER TO postgres;
ALTER TABLE "Зона"                           OWNER TO postgres;
ALTER TABLE "Оборудование"                   OWNER TO postgres;
ALTER TABLE "Заявка_на_ремонт"               OWNER TO postgres;
ALTER TABLE "Тариф"                          OWNER TO postgres;
ALTER TABLE "Абонемент"                      OWNER TO postgres;
ALTER TABLE "Групповая_тренировка"           OWNER TO postgres;
ALTER TABLE "Персональная_тренировка"        OWNER TO postgres;
ALTER TABLE "Запись_на_групповую_тренировку" OWNER TO postgres;

ALTER SEQUENCE "Клиент_id_клиента_seq"                          OWNER TO postgres;
ALTER SEQUENCE "Тренер_id_тренера_seq"                          OWNER TO postgres;
ALTER SEQUENCE "Зона_id_зоны_seq"                               OWNER TO postgres;
ALTER SEQUENCE "Оборудование_id_оборудования_seq"               OWNER TO postgres;
ALTER SEQUENCE "Заявка_на_ремонт_id_заявки_seq"                 OWNER TO postgres;
ALTER SEQUENCE "Тариф_id_тарифа_seq"                            OWNER TO postgres;
ALTER SEQUENCE "Абонемент_id_абонемента_seq"                    OWNER TO postgres;
ALTER SEQUENCE "Групповая_трени_id_групповой_тре_seq"           OWNER TO postgres;
ALTER SEQUENCE "Персональная_тр_id_персональной__seq"           OWNER TO postgres;
ALTER SEQUENCE "Запись_на_групповую_тре_id_записи_seq"          OWNER TO postgres;
-- +goose StatementEnd
