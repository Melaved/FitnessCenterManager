package handlers

import (
    "database/sql"
    "log"
    "strconv"
    "time"

    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"

    "github.com/gofiber/fiber/v2"
)

// ====== Страница со списком ======
func GetSubscriptionsPage(c *fiber.Ctx) error {
    db := database.GetDB()

    ctx, cancel := withDBTimeout()
    defer cancel()

    rows, err := db.QueryContext(ctx, `
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

// APIv1ListSubscriptions — JSON список абонементов
func APIv1ListSubscriptions(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
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
        return jsonError(c, 500, "Ошибка загрузки абонементов", err)
    }
    defer rows.Close()
    type dto struct {
        ID         int     `json:"id"`
        ClientID   int     `json:"client_id"`
        TariffID   int     `json:"tariff_id"`
        StartDate  string  `json:"start_date"`
        EndDate    string  `json:"end_date"`
        Status     string  `json:"status"`
        Price      float64 `json:"price"`
        ClientName string  `json:"client_name"`
        TariffName string  `json:"tariff_name"`
    }
    var list []dto
    for rows.Next() {
        var s models.Subscription
        if err := rows.Scan(&s.ID, &s.ClientID, &s.TariffID, &s.StartDate, &s.EndDate, &s.Status, &s.Price, &s.ClientName, &s.TariffName); err != nil {
            return jsonError(c, 500, "Ошибка чтения абонемента", err)
        }
        list = append(list, dto{
            ID: s.ID,
            ClientID: s.ClientID,
            TariffID: s.TariffID,
            StartDate: s.StartDate.Format("2006-01-02"),
            EndDate: s.EndDate.Format("2006-01-02"),
            Status: s.Status,
            Price: s.Price,
            ClientName: s.ClientName,
            TariffName: s.TariffName,
        })
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"subscriptions": list})
}

// APIv1CreateSubscription — 201 + Location (повторяет CreateSubscription)
func APIv1CreateSubscription(c *fiber.Ctx) error {
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.ClientID <= 0 || f.TariffID <= 0 || f.StartDate == "" || f.EndDate == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }
    start, err := time.Parse("2006-01-02", f.StartDate)
    if err != nil { return jsonError(c, 400, "Неверная дата начала", err) }
    end, err := time.Parse("2006-01-02", f.EndDate)
    if err != nil { return jsonError(c, 400, "Неверная дата окончания", err) }
    if end.Before(start) { return jsonError(c, 400, "Дата окончания раньше даты начала", nil) }

    db := database.GetDB()
    var price float64
    if f.Price != "" {
        p, err := strconv.ParseFloat(f.Price, 64)
        if err != nil { return jsonError(c, 400, "Неверная цена", err) }
        price = p
    } else {
        // взять стоимость из тарифа
        ctxP, cancelP := withDBTimeout()
        defer cancelP()
        if err := db.QueryRowContext(ctxP, `SELECT "Стоимость" FROM "Тариф" WHERE "id_тарифа"=$1`, f.TariffID).Scan(&price); err != nil {
            return jsonError(c, 400, "Не удалось получить стоимость тарифа", err)
        }
    }

    ctx, cancel := withDBTimeout()
    defer cancel()
    var id int
    err = db.QueryRowContext(ctx, `
        INSERT INTO "Абонемент" ("id_клиента","id_тарифа","Дата_начала","Дата_окончания","Статус","Цена")
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING "id_абонемента"
    `, f.ClientID, f.TariffID, start, end, f.Status, price).Scan(&id)
    if err != nil {
        return jsonError(c, 500, "Ошибка создания абонемента", err)
    }
    c.Set("Location", "/api/v1/subscriptions/"+strconv.Itoa(id))
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "id": id})
}

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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.ClientID <= 0 || f.TariffID <= 0 || f.StartDate == "" || f.EndDate == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }

    start, err := time.Parse("2006-01-02", f.StartDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата начала", err)
    }
    end, err := time.Parse("2006-01-02", f.EndDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата окончания", err)
    }
    if end.Before(start) {
        return jsonError(c, 400, "Дата окончания раньше даты начала", nil)
    }

	db := database.GetDB()

	var price float64
    if f.Price != "" {
        p, err := strconv.ParseFloat(f.Price, 64)
        if err != nil {
            return jsonError(c, 400, "Неверная цена", err)
        }
        price = p
    } else {
        ctx, cancel := withDBTimeout()
        defer cancel()
        if err := db.QueryRowContext(ctx, `SELECT "Стоимость" FROM "Тариф" WHERE "id_тарифа"=$1`, f.TariffID).Scan(&price); err != nil {
            return jsonError(c, 400, "Не удалось получить стоимость тарифа", err)
        }
    }

	if f.Status == "" {
		f.Status = "Активен"
	}

	var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
        INSERT INTO "Абонемент" ("id_клиента","id_тарифа","Дата_начала","Дата_окончания","Статус","Цена")
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING "id_абонемента"
    `, f.ClientID, f.TariffID, start, end, f.Status, price).Scan(&id)
    if err != nil {
        log.Printf("❌ create sub: %v", err)
        return jsonError(c, 500, "Ошибка сохранения в БД", err)
    }

    return jsonOK(c, fiber.Map{"message": "Абонемент создан", "id": id})
}

func GetSubscriptionByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
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

    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
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
        return jsonError(c, 404, "Абонемент не найден", nil)
    }
    if err != nil {
        log.Printf("❌ get sub: %v", err)
        return jsonError(c, 500, "Ошибка БД", err)
    }
    return jsonOK(c, fiber.Map{"subscription": s})
}

// ====== Update ======
func UpdateSubscription(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.ClientID <= 0 || f.TariffID <= 0 || f.StartDate == "" || f.EndDate == "" || f.Status == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }

	start, err := time.Parse("2006-01-02", f.StartDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата начала", err)
    }
	end, err := time.Parse("2006-01-02", f.EndDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата окончания", err)
    }
    if end.Before(start) {
        return jsonError(c, 400, "Дата окончания раньше даты начала", nil)
    }

	var price float64
    if f.Price != "" {
        p, err := strconv.ParseFloat(f.Price, 64)
        if err != nil {
            return jsonError(c, 400, "Неверная цена", err)
        }
        price = p
    } else {
        // оставить прежнюю цену
        db := database.GetDB()
        ctx, cancel := withDBTimeout()
        defer cancel()
        if err := db.QueryRowContext(ctx, `SELECT "Цена" FROM "Абонемент" WHERE "id_абонемента"=$1`, id).Scan(&price); err != nil {
            return jsonError(c, 400, "Не удалось получить текущую цену", err)
        }
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Абонемент"
        SET "id_клиента"=$2, "id_тарифа"=$3, "Дата_начала"=$4, "Дата_окончания"=$5, "Статус"=$6, "Цена"=$7
        WHERE "id_абонемента"=$1
    `, id, f.ClientID, f.TariffID, start, end, f.Status, price)
    if err != nil {
        log.Printf("❌ update sub: %v", err)
        return jsonError(c, 500, "Ошибка обновления в БД", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Абонемент не найден", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Абонемент обновлён"})
}

// ====== Delete ======
func DeleteSubscription(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return jsonError(c, 500, "Не удалось начать транзакцию", err)
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    // 1) Персональные тренировки этого абонемента
    if _, err = tx.ExecContext(ctx, `DELETE FROM "Персональная_тренировка" WHERE "id_абонемента" = $1`, id); err != nil {
        return jsonError(c, 500, "Невозможно удалить связанные персональные тренировки", err)
    }

    // 2) Записи на групповые тренировки этого абонемента
    if _, err = tx.ExecContext(ctx, `DELETE FROM "Запись_на_групповую_тренировку" WHERE "id_абонемента" = $1`, id); err != nil {
        return jsonError(c, 500, "Невозможно удалить групповые записи абонемента", err)
    }

    // 3) Сам абонемент
    res, err := tx.ExecContext(ctx, `DELETE FROM "Абонемент" WHERE "id_абонемента" = $1`, id)
    if err != nil {
        return jsonError(c, 500, "Ошибка удаления абонемента", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Абонемент не найден", nil)
    }

    if err = tx.Commit(); err != nil {
        return jsonError(c, 500, "Ошибка фиксации транзакции", err)
    }

    return jsonOK(c, fiber.Map{"message": "Абонемент и связанные данные удалены"})
}


// ====== API: тарифы для селекта ======
func GetTariffsForSelect(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT "id_тарифа","Название_тарифа","Стоимость"
        FROM "Тариф"
        ORDER BY "id_тарифа"
    `)
    if err != nil {
        return jsonError(c, 500, "Ошибка чтения тарифов", err)
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
    return jsonOK(c, fiber.Map{"tariffs": list})
}
