package handlers

import (
	"context"
	"database/sql"
	"log"
	"time"

	"fitness-center-manager/internal/database"
	"github.com/gofiber/fiber/v2"
)

const (
	dashboardStatsQuery = `
WITH
    clients AS (
        SELECT COUNT(*)::int AS total
        FROM public."Клиент"
    ),
    trainers AS (
        SELECT COUNT(*)::int AS total
        FROM public."Тренер"
    ),
    active_subscriptions AS (
        SELECT COUNT(*) FILTER (WHERE "Статус" = 'Активен')::int AS total
        FROM public."Абонемент"
    ),
    group_trainings AS (
        SELECT COUNT(*)::int AS upcoming
        FROM public."Групповая_тренировка"
        WHERE "Время_начала" >= NOW()
    ),
    personal_trainings AS (
        SELECT COUNT(*)::int AS upcoming
        FROM public."Персональная_тренировка"
        WHERE "Время_начала" >= NOW()
    ),
    zones AS (
        SELECT
            COUNT(*) FILTER (WHERE "Статус" = 'Доступна')::int AS active,
            COUNT(*) FILTER (WHERE "Статус" = 'На ремонте')::int AS repair,
            COALESCE(SUM("Вместимость"), 0)::int AS capacity
        FROM public."Зона"
    ),
    equipment AS (
        SELECT
            COUNT(*)::int AS total,
            COUNT(*) FILTER (WHERE "Статус" = 'Работает')::int AS working,
            COUNT(*) FILTER (WHERE "Статус" = 'На ремонте')::int AS repair,
            COUNT(*) FILTER (WHERE "Фото" IS NULL)::int AS no_photo
        FROM public."Оборудование"
    )
SELECT
    clients.total,
    trainers.total,
    active_subscriptions.total,
    group_trainings.upcoming,
    personal_trainings.upcoming,
    zones.active,
    zones.repair,
    zones.capacity,
    equipment.total,
    equipment.working,
    equipment.repair,
    equipment.no_photo
FROM clients, trainers, active_subscriptions, group_trainings, personal_trainings, zones, equipment;
`
	recentClientsQuery = `
SELECT
    "id_клиента",
    "ФИО",
    "Дата_регистрации",
    COALESCE(last_tariff, '') AS last_tariff,
    COALESCE(last_subscription_status, '') AS last_status
FROM view_client_enriched
ORDER BY "Дата_регистрации" DESC
LIMIT 5;
`
	expiringSubscriptionsQuery = `
SELECT
    a."id_абонемента",
    c."ФИО",
    t."Название_тарифа",
    a."Дата_окончания"
FROM "Абонемент" a
JOIN "Клиент" c ON c."id_клиента" = a."id_клиента"
JOIN "Тариф" t ON t."id_тарифа" = a."id_тарифа"
WHERE a."Статус" = 'Активен'
  AND a."Дата_окончания" BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '30 days')
ORDER BY a."Дата_окончания"
LIMIT 5;
`
	equipmentInRepairQuery = `
SELECT
    e."Название",
    COALESCE(z."Название", '—') AS zone_name,
    e."Дата_последнего_ТО"
FROM "Оборудование" e
LEFT JOIN "Зона" z ON z."id_зоны" = e."id_зоны"
WHERE e."Статус" = 'На ремонте'
ORDER BY e."Дата_последнего_ТО" DESC NULLS LAST
LIMIT 5;
`
)

const (
	dateDisplayFormat = "02.01.2006"
)

type recentClientCard struct {
	ID         int
	FIO        string
	Registered string
	LastTariff string
	LastStatus string
}

type expiringSubscriptionCard struct {
	ID     int
	Client string
	Tariff string
	EndsAt string
}

type equipmentRepairCard struct {
	Name        string
	Zone        string
	LastService string
}

func Dashboard(c *fiber.Ctx) error {
	db := database.GetDB()
	ctx, cancel := withDBTimeout()
	defer cancel()

	type Stats struct {
		Clients           int
		Trainers          int
		Subscriptions     int
		GroupTrainings    int
		PersonalTrainings int
		Trainings         int
	}
	type ZonesStats struct {
		Active        int
		Repair        int
		TotalCapacity int
	}
	type EquipmentStats struct {
		Total   int
		Working int
		Repair  int
		NoPhoto int
	}

	var (
		stats     Stats
		zones     ZonesStats
		equipment EquipmentStats
		warnings  []string
	)

	if err := db.QueryRowContext(ctx, dashboardStatsQuery).Scan(
		&stats.Clients,
		&stats.Trainers,
		&stats.Subscriptions,
		&stats.GroupTrainings,
		&stats.PersonalTrainings,
		&zones.Active,
		&zones.Repair,
		&zones.TotalCapacity,
		&equipment.Total,
		&equipment.Working,
		&equipment.Repair,
		&equipment.NoPhoto,
	); err != nil {
		log.Printf("dashboard stats query failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Не удалось получить статистику: " + err.Error())
	}
	stats.Trainings = stats.GroupTrainings + stats.PersonalTrainings

	recentClients, err := loadRecentClients(ctx, db)
	if err != nil {
		log.Printf("recent clients query failed: %v", err)
		warnings = append(warnings, "Не удалось получить список последних клиентов")
	}

	expiringSubs, err := loadExpiringSubscriptions(ctx, db)
	if err != nil {
		log.Printf("expiring subscriptions query failed: %v", err)
		warnings = append(warnings, "Не удалось получить абонементы с истекающим сроком")
	}

	equipmentRepairs, err := loadEquipmentInRepair(ctx, db)
	if err != nil {
		log.Printf("equipment repairs query failed: %v", err)
		warnings = append(warnings, "Не удалось получить список оборудования на ремонте")
	}

	return c.Render("dashboard", fiber.Map{
		"Title":                 "Главная",
		"Stats":                 stats,
		"ZonesStats":            zones,
		"EquipmentStats":        equipment,
		"RecentClients":         recentClients,
		"ExpiringSubscriptions": expiringSubs,
		"EquipmentRepairs":      equipmentRepairs,
		"DashboardWarnings":     warnings,
	})
}

func loadRecentClients(ctx context.Context, db *sql.DB) ([]recentClientCard, error) {
	rows, err := db.QueryContext(ctx, recentClientsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		list       []recentClientCard
		regDate    time.Time
		lastTariff sql.NullString
		lastStatus sql.NullString
	)

	for rows.Next() {
		var item recentClientCard
		if err := rows.Scan(&item.ID, &item.FIO, &regDate, &lastTariff, &lastStatus); err != nil {
			return nil, err
		}
		item.Registered = regDate.Format(dateDisplayFormat)
		if lastTariff.Valid && lastTariff.String != "" {
			item.LastTariff = lastTariff.String
		} else {
			item.LastTariff = "—"
		}
		if lastStatus.Valid && lastStatus.String != "" {
			item.LastStatus = lastStatus.String
		} else {
			item.LastStatus = "Нет абонементов"
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func loadExpiringSubscriptions(ctx context.Context, db *sql.DB) ([]expiringSubscriptionCard, error) {
	rows, err := db.QueryContext(ctx, expiringSubscriptionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		list   []expiringSubscriptionCard
		endsAt time.Time
	)

	for rows.Next() {
		var item expiringSubscriptionCard
		if err := rows.Scan(&item.ID, &item.Client, &item.Tariff, &endsAt); err != nil {
			return nil, err
		}
		item.EndsAt = endsAt.Format(dateDisplayFormat)
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func loadEquipmentInRepair(ctx context.Context, db *sql.DB) ([]equipmentRepairCard, error) {
	rows, err := db.QueryContext(ctx, equipmentInRepairQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		list        []equipmentRepairCard
		lastService sql.NullTime
	)

	for rows.Next() {
		var item equipmentRepairCard
		if err := rows.Scan(&item.Name, &item.Zone, &lastService); err != nil {
			return nil, err
		}
		if lastService.Valid {
			item.LastService = lastService.Time.Format(dateDisplayFormat)
		} else {
			item.LastService = "—"
		}
		list = append(list, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
