package handlers

import (
    "database/sql"
    "errors"
    "log"
    "strconv"
    "strings"

    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"

    "github.com/gofiber/fiber/v2"
)

// GetTariffsPage — страница со списком тарифов
func GetTariffsPage(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()

    rows, err := db.QueryContext(ctx, `
        SELECT 
            "id_тарифа",
            "Название_тарифа",
            COALESCE("Описание", ''),
            COALESCE("Стоимость", 0),
            COALESCE("Время_доступа", '0 hours'::interval),
            COALESCE("Наличие_групповых_тренировок", false),
            COALESCE("Наличие_персональных_тренировок", false)
        FROM "Тариф"
        ORDER BY "id_тарифа" DESC
    `)
    if err != nil {
        log.Printf("❌ tariffs list error: %v", err)
        return c.Render("tariffs", fiber.Map{
            "Title":        "Тарифы",
            "Tariffs":      []models.Tariff{},
            "Message":      "Не удалось загрузить тарифы",
            "ExtraScripts": templateScript("/static/js/tariffs.js"),
        })
    }
    defer rows.Close()

    var list []models.Tariff
    for rows.Next() {
        var t models.Tariff
        var access sql.NullString
        if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Price, &access, &t.HasGroupTrainings, &t.HasPersonalTrainings); err != nil {
            log.Printf("❌ scan tariff: %v", err)
            continue
        }
        if access.Valid {
            t.AccessTime = access.String
        }
        list = append(list, t)
    }
    if err = rows.Err(); err != nil {
        log.Printf("❌ rows err: %v", err)
    }

    return c.Render("tariffs", fiber.Map{
        "Title":        "Тарифы",
        "Tariffs":      list,
        "ExtraScripts": templateScript("/static/js/tariffs.js"),
    })
}

// GetTariffByID — JSON один тариф (для редактирования)
func GetTariffByID(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()

    var t models.Tariff
    var access sql.NullString
    err = db.QueryRowContext(ctx, `
        SELECT 
            "id_тарифа",
            "Название_тарифа",
            COALESCE("Описание", ''),
            COALESCE("Стоимость", 0),
            COALESCE("Время_доступа", '0 hours'::interval),
            COALESCE("Наличие_групповых_тренировок", false),
            COALESCE("Наличие_персональных_тренировок", false)
        FROM "Тариф"
        WHERE "id_тарифа"=$1
    `, id).Scan(&t.ID, &t.Name, &t.Description, &t.Price, &access, &t.HasGroupTrainings, &t.HasPersonalTrainings)

    switch {
    case errors.Is(err, sql.ErrNoRows):
        return jsonError(c, 404, "Тариф не найден", nil)
    case err != nil:
        return jsonError(c, 500, "DB: ошибка чтения", err)
    }
    if access.Valid {
        t.AccessTime = access.String
    }
    return jsonOK(c, fiber.Map{"tariff": t})
}

// CreateTariff — создать тариф
func CreateTariff(c *fiber.Ctx) error {
    type formT struct {
        Name        string `form:"name"`
        Description string `form:"description"`
        Price       string `form:"price"`
        AccessTime  string `form:"access_time"` // строка interval: напр. "30 days", "1 mon"
        HasGroup    string `form:"has_group"`
        HasPersonal string `form:"has_personal"`
    }
    var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    name := strings.TrimSpace(f.Name)
    if name == "" {
        return jsonError(c, 400, "Название тарифа обязательно", nil)
    }
    var price float64
    if strings.TrimSpace(f.Price) != "" {
        p, err := strconv.ParseFloat(strings.ReplaceAll(f.Price, ",", "."), 64)
        if err != nil || p <= 0 {
            return jsonError(c, 400, "Неверная стоимость", err)
        }
        price = p
    } else {
        return jsonError(c, 400, "Стоимость обязательна", nil)
    }
    hasGroup := strings.ToLower(strings.TrimSpace(f.HasGroup)) == "on"
    hasPersonal := strings.ToLower(strings.TrimSpace(f.HasPersonal)) == "on"

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()

    var id int
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Тариф" ("Название_тарифа","Описание","Стоимость","Время_доступа","Наличие_групповых_тренировок","Наличие_персональных_тренировок")
        VALUES ($1,$2,$3, NULLIF($4,'')::interval, $5, $6)
        RETURNING "id_тарифа"
    `, name, f.Description, price, strings.TrimSpace(f.AccessTime), hasGroup, hasPersonal).Scan(&id)
    if err != nil {
        return jsonError(c, 500, "Ошибка создания тарифа", err)
    }
    return jsonOK(c, fiber.Map{"message": "Тариф создан", "id": id})
}

// UpdateTariff — обновить тариф
func UpdateTariff(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    type formT struct {
        Name        string `form:"name"`
        Description string `form:"description"`
        Price       string `form:"price"`
        AccessTime  string `form:"access_time"`
        HasGroup    string `form:"has_group"`
        HasPersonal string `form:"has_personal"`
    }
    var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    name := strings.TrimSpace(f.Name)
    if name == "" {
        return jsonError(c, 400, "Название тарифа обязательно", nil)
    }
    p, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(f.Price), ",", "."), 64)
    if err != nil || p <= 0 {
        return jsonError(c, 400, "Неверная стоимость", err)
    }
    hasGroup := strings.ToLower(strings.TrimSpace(f.HasGroup)) == "on"
    hasPersonal := strings.ToLower(strings.TrimSpace(f.HasPersonal)) == "on"

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Тариф"
        SET "Название_тарифа"=$2,
            "Описание"=$3,
            "Стоимость"=$4,
            "Время_доступа"=NULLIF($5,'')::interval,
            "Наличие_групповых_тренировок"=$6,
            "Наличие_персональных_тренировок"=$7
        WHERE "id_тарифа"=$1
    `, id, name, f.Description, p, strings.TrimSpace(f.AccessTime), hasGroup, hasPersonal)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Тариф не найден", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Тариф обновлён"})
}

// DeleteTariff — удалить тариф
func DeleteTariff(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    _, err = db.ExecContext(ctx, `DELETE FROM "Тариф" WHERE "id_тарифа"=$1`, id)
    if err != nil {
        return jsonError(c, 409, "Невозможно удалить тариф: есть связанные абонементы", err)
    }
    return jsonOK(c, fiber.Map{"message": "Тариф удалён"})
}

