package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"time"

	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/models"

	"github.com/gofiber/fiber/v2"
)

func templateScript(src string) template.HTML {
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, src))
}

// ====== Страница со списком ======
func GetSubscriptionsPage(c *fiber.Ctx) error {
	db := database.GetDB()

	rows, err := db.Query(`
		SELECT s."id_абонемента",
		       s."id_клиента",
		       s."id_тарифа",
		       s."Дата_начала",
		       s."Дата_окончания",
		       s."Статус",
		       s."Цена",
		       c."ФИО"              AS client_name,
		       t."Название_тарифа"  AS tariff_name
		FROM "Абонемент" s
		JOIN "Клиент" c ON c."id_клиента" = s."id_клиента"
		JOIN "Тариф"  t ON t."id_тарифа"  = s."id_тарифа"
		ORDER BY s."id_абонемента" DESC
	`)
	if err != nil {
		log.Printf("❌ subscriptions list error: %v", err)
		return c.Render("subscriptions", fiber.Map{
			"Title":         "Абонементы",
			"Subscriptions": []models.Subscription{},
			"Message":       "Не удалось загрузить данные абонементов",
			"ExtraScripts":  templateScript(`/static/js/subscriptions.js`),
		})
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		if err := rows.Scan(
			&s.ID, &s.ClientID, &s.TariffID,
			&s.StartDate, &s.EndDate,
			&s.Status, &s.Price,
			&s.ClientName, &s.TariffName,
		); err != nil {
			log.Printf("❌ scan sub: %v", err)
			continue
		}
		// 👇 ЭТОЙ СТРОКИ НЕ ХВАТАЛО
		subs = append(subs, s)
	}
	if err = rows.Err(); err != nil {
		log.Printf("❌ rows err: %v", err)
	}

	log.Printf("✅ загружено абонементов: %d", len(subs))

	return c.Render("subscriptions", fiber.Map{
		"Title":         "Абонементы",
		"Subscriptions": subs,
		"ExtraScripts":  templateScript(`/static/js/subscriptions.js`),
	})
}

// ====== Create ======
func CreateSubscription(c *fiber.Ctx) error {
	type formT struct {
		ClientID  int    `form:"client_id"`
		TariffID  int    `form:"tariff_id"`
		StartDate string `form:"start_date"` // YYYY-MM-DD
		EndDate   string `form:"end_date"`   // YYYY-MM-DD
		Status    string `form:"status"`
		Price     string `form:"price"` // если пусто — возьмём из тарифа
	}
	var f formT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.ClientID <= 0 || f.TariffID <= 0 || f.StartDate == "" || f.EndDate == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}

	start, err := time.Parse("2006-01-02", f.StartDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата начала"})
	}
	end, err := time.Parse("2006-01-02", f.EndDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата окончания"})
	}
	if end.Before(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Дата окончания раньше даты начала"})
	}

	db := database.GetDB()

	// цена
	var price float64
	if f.Price != "" {
		p, err := strconv.ParseFloat(f.Price, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная цена"})
		}
		price = p
	} else {
		if err := db.QueryRow(`SELECT "Стоимость" FROM "Тариф" WHERE "id_тарифа"=$1`, f.TariffID).Scan(&price); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Не удалось получить стоимость тарифа"})
		}
	}

	if f.Status == "" {
		f.Status = "Активен"
	}

	var id int
	err = db.QueryRow(`
		INSERT INTO "Абонемент" ("id_клиента","id_тарифа","Дата_начала","Дата_окончания","Статус","Цена")
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING "id_абонемента"
	`, f.ClientID, f.TariffID, start, end, f.Status, price).Scan(&id)
	if err != nil {
		log.Printf("❌ create sub: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка сохранения в БД"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Абонемент создан", "id": id})
}

// ====== Read one ======
func GetSubscriptionByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	db := database.GetDB()
	var s struct {
		ID         int       `json:"id"`
		ClientID   int       `json:"client_id"`
		TariffID   int       `json:"tariff_id"`
		StartDate  time.Time `json:"start_date"`
		EndDate    time.Time `json:"end_date"`
		Status     string    `json:"status"`
		Price      float64   `json:"price"`
		ClientName string    `json:"client_name"`
		TariffName string    `json:"tariff_name"`
	}

	err = db.QueryRow(`
		SELECT s."id_абонемента",
		       s."id_клиента",
		       s."id_тарифа",
		       s."Дата_начала",
		       s."Дата_окончания",
		       s."Статус",
		       s."Цена",
		       c."ФИО"              AS client_name,
		       t."Название_тарифа"  AS tariff_name
		FROM "Абонемент" s
		JOIN "Клиент" c ON c."id_клиента" = s."id_клиента"
		JOIN "Тариф"  t ON t."id_тарифа"  = s."id_тарифа"
		WHERE s."id_абонемента"=$1
	`, id).Scan(
		&s.ID, &s.ClientID, &s.TariffID,
		&s.StartDate, &s.EndDate,
		&s.Status, &s.Price,
		&s.ClientName, &s.TariffName,
	)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Абонемент не найден"})
	}
	if err != nil {
		log.Printf("❌ get sub: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка БД"})
	}
	return c.JSON(fiber.Map{"success": true, "subscription": s})
}

// ====== Update ======
func UpdateSubscription(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	type formT struct {
		ClientID  int    `form:"client_id"`
		TariffID  int    `form:"tariff_id"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
		Status    string `form:"status"`
		Price     string `form:"price"`
	}
	var f formT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.ClientID <= 0 || f.TariffID <= 0 || f.StartDate == "" || f.EndDate == "" || f.Status == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}

	start, err := time.Parse("2006-01-02", f.StartDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата начала"})
	}
	end, err := time.Parse("2006-01-02", f.EndDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата окончания"})
	}
	if end.Before(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Дата окончания раньше даты начала"})
	}

	var price float64
	if f.Price != "" {
		p, err := strconv.ParseFloat(f.Price, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная цена"})
		}
		price = p
	} else {
		// оставить прежнюю цену
		db := database.GetDB()
		if err := db.QueryRow(`SELECT "Цена" FROM "Абонемент" WHERE "id_абонемента"=$1`, id).Scan(&price); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Не удалось получить текущую цену"})
		}
	}

	db := database.GetDB()
	res, err := db.Exec(`
		UPDATE "Абонемент"
		SET "id_клиента"=$2, "id_тарифа"=$3, "Дата_начала"=$4, "Дата_окончания"=$5, "Статус"=$6, "Цена"=$7
		WHERE "id_абонемента"=$1
	`, id, f.ClientID, f.TariffID, start, end, f.Status, price)
	if err != nil {
		log.Printf("❌ update sub: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка обновления в БД"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Абонемент не найден"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Абонемент обновлён"})
}

// ====== Delete ======
func DeleteSubscription(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Не удалось начать транзакцию"})
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1) Персональные тренировки этого абонемента
	if _, err = tx.Exec(`DELETE FROM "Персональная_тренировка" WHERE "id_абонемента" = $1`, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Невозможно удалить связанные персональные тренировки"})
	}

	// 2) Записи на групповые тренировки этого абонемента
	if _, err = tx.Exec(`DELETE FROM "Запись_на_групповую_тренировку" WHERE "id_абонемента" = $1`, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Невозможно удалить групповые записи абонемента"})
	}

	// 3) Сам абонемент
	res, err := tx.Exec(`DELETE FROM "Абонемент" WHERE "id_абонемента" = $1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка удаления абонемента"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Абонемент не найден"})
	}

	if err = tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка фиксации транзакции"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Абонемент и связанные данные удалены"})
}


// ====== API: тарифы для селекта ======
func GetTariffsForSelect(c *fiber.Ctx) error {
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT "id_тарифа","Название_тарифа","Стоимость"
		FROM "Тариф"
		ORDER BY "id_тарифа"
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка чтения тарифов"})
	}
	defer rows.Close()

	type t struct {
		ID    int     `json:"id"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}
	var list []t
	for rows.Next() {
		var item t
		if err := rows.Scan(&item.ID, &item.Name, &item.Price); err == nil {
			list = append(list, item)
		}
	}
	return c.JSON(fiber.Map{"success": true, "tariffs": list})
}
