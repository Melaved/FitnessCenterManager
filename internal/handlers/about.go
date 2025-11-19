package handlers

import (
    "strconv"
    "strings"
    "time"
    "fmt"
    "fitness-center-manager/internal/database"
    "github.com/gofiber/fiber/v2"
)

// Страница «Отчетность»
func About(c *fiber.Ctx) error {
    return c.Render("about", fiber.Map{
        "Title": "Отчетность",
    })
}

// Валидации/констант
var repAllowedZoneStatuses = map[string]bool{
    "Доступна":   true,
    "На ремонте": true,
    "Закрыта":    true,
}

// ======= Запросы выборки =======

// POST /about/query/clients-after-date
// Параметр: date (YYYY-MM-DD)
func ReportClientsAfterDate(c *fiber.Ctx) error {
    type form struct{ Date string `form:"date"` }
    var f form
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if strings.TrimSpace(f.Date) == "" {
        return jsonError(c, 400, "Укажите дату в формате YYYY-MM-DD", nil)
    }
    dt, err := time.Parse("2006-01-02", f.Date)
    if err != nil {
        return jsonError(c, 400, "Неверный формат даты", err)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT "id_клиента", "ФИО", "Дата_регистрации"
        FROM "Клиент"
        WHERE "Дата_регистрации" > $1
        ORDER BY "Дата_регистрации" DESC
        LIMIT 100
    `, dt)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка выборки клиентов", err)
    }
    defer rows.Close()
    type rowT struct{
        ID int `json:"id_клиента"`
        FIO string `json:"фио"`
        Reg time.Time `json:"дата_регистрации"`
    }
    var out []rowT
    for rows.Next() {
        var r rowT
        if err := rows.Scan(&r.ID, &r.FIO, &r.Reg); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        out = append(out, r)
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"rows": out})
}

// POST /about/query/subscriptions-by-status
// Параметр: status (string)
func ReportSubscriptionsByStatus(c *fiber.Ctx) error {
    status := strings.TrimSpace(c.FormValue("status"))
    if status == "" {
        status = "Активен"
    }
    if len(status) > 20 {
        return jsonError(c, 400, "Статус слишком длинный", nil)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT s."id_абонемента",
               c."ФИО"              AS client_name,
               t."Название_тарифа"  AS tariff_name,
               s."Дата_начала",
               s."Дата_окончания",
               s."Статус",
               COALESCE(s."Цена", 0) AS price
        FROM "Абонемент" s
        JOIN "Клиент" c ON c."id_клиента" = s."id_клиента"
        JOIN "Тариф"  t ON t."id_тарифа"  = s."id_тарифа"
        WHERE s."Статус" = $1
        ORDER BY s."id_абонемента" DESC
        LIMIT 200
    `, status)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка выборки абонементов", err)
    }
    defer rows.Close()
    type rowT struct{
        ID int `json:"id_абонемента"`
        Client string `json:"фио_клиента"`
        Tariff string `json:"название_тарифа"`
        Start time.Time `json:"дата_начала"`
        End time.Time `json:"дата_окончания"`
        Status string `json:"статус"`
        Price float64 `json:"цена"`
    }
    var out []rowT
    for rows.Next() {
        var r rowT
        if err := rows.Scan(&r.ID, &r.Client, &r.Tariff, &r.Start, &r.End, &r.Status, &r.Price); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        out = append(out, r)
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"rows": out})
}

// POST /about/query/revenue-by-tariff
// Параметры: start_date, end_date (YYYY-MM-DD), min_revenue (float)
// Полный SELECT: SELECT, FROM, WHERE, GROUP BY, HAVING, ORDER BY
func ReportRevenueByTariff(c *fiber.Ctx) error {
    startS := c.FormValue("start_date")
    endS := c.FormValue("end_date")
    minRevS := strings.TrimSpace(c.FormValue("min_revenue"))
    if startS == "" || endS == "" {
        return jsonError(c, 400, "Укажите период дат", nil)
    }
    start, err := time.Parse("2006-01-02", startS)
    if err != nil { return jsonError(c, 400, "Неверная дата начала", err) }
    end, err := time.Parse("2006-01-02", endS)
    if err != nil { return jsonError(c, 400, "Неверная дата окончания", err) }
    if end.Before(start) {
        return jsonError(c, 400, "Дата окончания раньше даты начала", nil)
    }
    minRev := 0.0
    if minRevS != "" {
        if v, err := strconv.ParseFloat(minRevS, 64); err == nil && v >= 0 {
            minRev = v
        } else {
            return jsonError(c, 400, "min_revenue должен быть неотрицательным числом", err)
        }
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT
            t."Название_тарифа"                AS tariff,
            COUNT(s.*)                          AS subs_count,
            SUM(COALESCE(s."Цена", 0))         AS revenue
        FROM "Абонемент" s
        JOIN "Тариф" t ON t."id_тарифа" = s."id_тарифа"
        WHERE s."Дата_начала" >= $1 AND s."Дата_окончания" <= $2
        GROUP BY t."Название_тарифа"
        HAVING SUM(COALESCE(s."Цена", 0)) >= $3
        ORDER BY revenue DESC
    `, start, end, minRev)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка выборки выручки", err)
    }
    defer rows.Close()
    type rowT struct{
        Tariff string `json:"тариф"`
        Count int `json:"количество"`
        Revenue float64 `json:"выручка"`
    }
    var out []rowT
    for rows.Next() {
        var r rowT
        if err := rows.Scan(&r.Tariff, &r.Count, &r.Revenue); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        out = append(out, r)
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"rows": out})
}

// POST /about/query/zones-min-equip
// Параметр: min_count (int) — пример с подзапросом
func ReportZonesWithMinEquipment(c *fiber.Ctx) error {
    minS := strings.TrimSpace(c.FormValue("min_count"))
    if minS == "" { minS = "1" }
    minN, err := strconv.Atoi(minS)
    if err != nil || minN < 0 {
        return jsonError(c, 400, "min_count должен быть целым >= 0", err)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT z."id_зоны", z."Название",
               (SELECT COUNT(*) FROM "Оборудование" e WHERE e."id_зоны" = z."id_зоны") AS equip_count
        FROM "Зона" z
        WHERE (SELECT COUNT(*) FROM "Оборудование" e WHERE e."id_зоны" = z."id_зоны") >= $1
        ORDER BY equip_count DESC, z."id_зоны" DESC
    `, minN)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка выборки зон", err)
    }
    defer rows.Close()
    type rowT struct{
        ID int `json:"id_зоны"`
        Name string `json:"название"`
        Count int `json:"количество_оборудования"`
    }
    var out []rowT
    for rows.Next() {
        var r rowT
        if err := rows.Scan(&r.ID, &r.Name, &r.Count); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        out = append(out, r)
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"rows": out})
}

// POST /about/query/zones-above-avg-capacity
// Некоррелированный подзапрос: зоны с вместимостью выше средней по клубу
func ReportZonesAboveAvgCapacity(c *fiber.Ctx) error {
	db := database.GetDB()
	ctx, cancel := withDBTimeout()
	defer cancel()

	rows, err := db.QueryContext(ctx, `
        WITH avg_capacity AS (
            SELECT COALESCE(AVG("Вместимость")::float8, 0) AS avg_capacity
            FROM "Зона"
        )
        SELECT z."id_зоны", z."Название", z."Вместимость", avg_capacity.avg_capacity
        FROM "Зона" z
        CROSS JOIN avg_capacity
        WHERE z."Вместимость" > avg_capacity.avg_capacity
        ORDER BY z."Вместимость" DESC, z."id_зоны" DESC
    `)
	if err != nil {
		return jsonError(c, 500, "DB: ошибка выборки зон по вместимости", err)
	}
	defer rows.Close()

	type rowT struct {
		ID       int    `json:"id_зоны"`
		Name     string `json:"название"`
		Capacity int    `json:"вместимость"`
	}
	var out []rowT
	var avgValue float64
	var avgSet bool
	for rows.Next() {
		var r rowT
		var avg float64
		if err := rows.Scan(&r.ID, &r.Name, &r.Capacity, &avg); err != nil {
			return jsonError(c, 500, "Ошибка чтения строки", err)
		}
		avgValue = avg
		avgSet = true
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return jsonError(c, 500, "Ошибка курсора", err)
	}
	if !avgSet {
		if err := db.QueryRowContext(ctx, `SELECT COALESCE(AVG("Вместимость")::float8, 0) FROM "Зона"`).Scan(&avgValue); err != nil {
			return jsonError(c, 500, "DB: ошибка подсчёта средней вместимости", err)
		}
	}
	return jsonOK(c, fiber.Map{
		"rows": out,
		"summary": fiber.Map{
			"avg_capacity": avgValue,
		},
	})
}

// ======= Операторы изменения данных =======

// POST /about/op/insert-zone
func ReportInsertZone(c *fiber.Ctx) error {
    type form struct{
        Name string `form:"name"`
        Description string `form:"description"`
        Capacity int `form:"capacity"`
        Status string `form:"status"`
    }
    var f form
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    f.Name = strings.TrimSpace(f.Name)
    f.Description = strings.TrimSpace(f.Description)
    f.Status = strings.TrimSpace(f.Status)
    if f.Name == "" || f.Capacity <= 0 || !repAllowedZoneStatuses[f.Status] {
        return jsonError(c, 400, "Заполните корректно: name, capacity>0, status", nil)
    }

    db := database.GetDB()
    var zoneID int
    ctx, cancel := withDBTimeout()
    defer cancel()
    if err := db.QueryRowContext(ctx, `
        INSERT INTO "Зона" ("Название","Описание","Вместимость","Статус")
        VALUES ($1,$2,$3,$4)
        RETURNING "id_зоны"
    `, f.Name, f.Description, f.Capacity, f.Status).Scan(&zoneID); err != nil {
        return jsonError(c, 500, "DB: ошибка вставки", err)
    }
    return jsonOK(c, fiber.Map{"message": "Зона добавлена", "zone_id": zoneID})
}

// POST /about/query/personal-finished
// Параметры: start_date, end_date (YYYY-MM-DD)
// Возвращает список завершённых персональных тренировок за период и агрегаты
func ReportPersonalFinished(c *fiber.Ctx) error {
    startS := strings.TrimSpace(c.FormValue("start_date"))
    endS := strings.TrimSpace(c.FormValue("end_date"))
    if startS == "" || endS == "" {
        return jsonError(c, 400, "Укажите период дат", nil)
    }
    start, err := time.Parse("2006-01-02", startS)
    if err != nil { return jsonError(c, 400, "Неверная дата начала", err) }
    end, err := time.Parse("2006-01-02", endS)
    if err != nil { return jsonError(c, 400, "Неверная дата окончания", err) }
    if end.Before(start) {
        return jsonError(c, 400, "Дата окончания раньше даты начала", nil)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()

    // Фильтруем по статусу "Завершена" и дате начала в пределах периода (включая конец дня)
    rows, err := db.QueryContext(ctx, `
        SELECT id, client_fio, trainer_fio, starts_at, duration_minutes, price_effective
        FROM public.v_personal_training_enriched
        WHERE status = 'Завершена'
          AND starts_at >= $1
          AND starts_at < ($2::date + INTERVAL '1 day')
        ORDER BY starts_at DESC, id DESC
    `, start, end)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка выборки персональных тренировок", err)
    }
    defer rows.Close()

    type rowT struct {
        ID        int       `json:"id"`
        Client    string    `json:"клиент"`
        Trainer   string    `json:"тренер"`
        Starts    time.Time `json:"начало"`
        Duration  int       `json:"длительность_мин"`
        Price     float64   `json:"стоимость"`
    }
    var out []rowT
    var totalCount int
    var totalSum float64
    for rows.Next() {
        var r rowT
        if err := rows.Scan(&r.ID, &r.Client, &r.Trainer, &r.Starts, &r.Duration, &r.Price); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        out = append(out, r)
        totalCount++
        totalSum += r.Price
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }

    return jsonOK(c, fiber.Map{
        "rows": out,
        "summary": fiber.Map{
            "количество": totalCount,
            "сумма": totalSum,
        },
    })
}

// POST /about/op/update-zone-status
func ReportUpdateZoneStatus(c *fiber.Ctx) error {
    name := strings.TrimSpace(c.FormValue("name"))
    status := strings.TrimSpace(c.FormValue("status"))
    if name == "" {
        return jsonError(c, 400, "Укажите название зоны", nil)
    }
    if !repAllowedZoneStatuses[status] {
        return jsonError(c, 400, "Недопустимый статус (Доступна/На ремонте/Закрыта)", nil)
    }

    db := database.GetDB()
    // Находим однозначно ID по названию
    ctx1, cancel1 := withDBTimeout()
    rows, err := db.QueryContext(ctx1, `SELECT "id_зоны" FROM "Зона" WHERE "Название"=$1 ORDER BY "id_зоны" LIMIT 2`, name)
    if err != nil { cancel1(); return jsonError(c, 500, "DB: ошибка поиска зоны", err) }
    var ids []int
    for rows.Next() { var id int; if err := rows.Scan(&id); err == nil { ids = append(ids, id) } }
    rows.Close(); cancel1()
    if len(ids) == 0 { return jsonError(c, 404, "Зона с таким названием не найдена", nil) }
    if len(ids) > 1 { return jsonError(c, 409, "Найдено несколько зон с таким названием. Уточните название.", nil) }
    id := ids[0]

    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `UPDATE "Зона" SET "Статус"=$2 WHERE "id_зоны"=$1`, id, status)
    if err != nil { return jsonError(c, 500, "DB: ошибка обновления", err) }
    aff, _ := res.RowsAffected()
    if aff == 0 { return jsonError(c, 404, "Зона не найдена", nil) }
    return jsonOK(c, fiber.Map{"message": fmt.Sprintf("Статус обновлён (ID: %d)", id)})
}

// POST /about/op/delete-zone
func ReportDeleteZone(c *fiber.Ctx) error {
    name := strings.TrimSpace(c.FormValue("name"))
    if name == "" { return jsonError(c, 400, "Укажите название зоны", nil) }

    db := database.GetDB()
    // Находим однозначно ID по названию
    ctx1, cancel1 := withDBTimeout()
    rows, err := db.QueryContext(ctx1, `SELECT "id_зоны" FROM "Зона" WHERE "Название"=$1 ORDER BY "id_зоны" LIMIT 2`, name)
    if err != nil { cancel1(); return jsonError(c, 500, "DB: ошибка поиска зоны", err) }
    var ids []int
    for rows.Next() { var id int; if err := rows.Scan(&id); err == nil { ids = append(ids, id) } }
    rows.Close(); cancel1()
    if len(ids) == 0 { return jsonError(c, 404, "Зона с таким названием не найдена", nil) }
    if len(ids) > 1 { return jsonError(c, 409, "Найдено несколько зон с таким названием. Уточните название.", nil) }
    id := ids[0]

    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Зона" WHERE "id_зоны"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка удаления", err)
    }
    if rowsAff, _ := res.RowsAffected(); rowsAff == 0 {
        return jsonError(c, 404, "Зона не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": fmt.Sprintf("Зона удалена (ID: %d)", id)})
}
