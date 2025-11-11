package handlers

import (
	"log"

	"fitness-center-manager/internal/database"
	"github.com/gofiber/fiber/v2"
)

func Dashboard(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()

	type Stats struct {
		Clients       int
		Trainers      int
		Subscriptions int
		Trainings     int
	}
	var s Stats
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Клиент"`).Scan(&s.Clients)
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Тренер"`).Scan(&s.Trainers)
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Абонемент" WHERE "Статус"='Активен'`).Scan(&s.Subscriptions)
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Групповая_тренировка" WHERE "Время_начала">NOW()`).Scan(&s.Trainings)

	type ZonesStats struct {
		Active        int
		Repair        int
		TotalCapacity int
	}
	var z ZonesStats
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Зона" WHERE "Статус"='Доступна'`).Scan(&z.Active)
    _ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Зона" WHERE "Статус"='На ремонте'`).Scan(&z.Repair)
    _ = db.QueryRowContext(ctx, `SELECT COALESCE(SUM("Вместимость"),0) FROM "Зона"`).Scan(&z.TotalCapacity)

	type EquipmentStats struct {
		Total   int
		Working int
		Repair  int
		NoPhoto int
	}
	var e EquipmentStats
    if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Оборудование"`).Scan(&e.Total); err != nil { log.Println("equip total:", err) }
    if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Оборудование" WHERE "Статус"='Работает'`).Scan(&e.Working); err != nil { log.Println("equip working:", err) }
    if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Оборудование" WHERE "Статус"='На ремонте'`).Scan(&e.Repair); err != nil { log.Println("equip repair:", err) }
    if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Оборудование" WHERE "Фото" IS NULL`).Scan(&e.NoPhoto); err != nil { log.Println("equip nophoto:", err) }

	return c.Render("dashboard", fiber.Map{
		"Title":          "Главная",
		"Stats":          s,
		"ZonesStats":     z,
		"EquipmentStats": e,
	})
}
