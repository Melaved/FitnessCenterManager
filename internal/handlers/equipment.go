package handlers

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fitness-center-manager/internal/database"

	"github.com/gofiber/fiber/v2"
)

const maxUpload = 5 * 1024 * 1024 // 5MB

// подключение скрипта на странице через layout ({{.ExtraScripts}})
func templateScript(src string) template.HTML {
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, src))
}

func dateYMD(t sql.NullTime) string {
	if t.Valid {
		return t.Time.Format("2006-01-02")
	}
	return ""
}
func nullableTimeArg(t sql.NullTime) any {
	if t.Valid {
		return t.Time
	}
	return nil
}
func nullablePhoto(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// ---------------- Нормализация статусов ----------------

func normEqStatus(s string) string {
	switch strings.TrimSpace(s) {
	case "Исправен", "Работает", "исправен":
		return "Исправен"
	case "На ремонте", "ремонт":
		return "На ремонте"
	case "Списан", "Списано":
		return "Списан"
	default:
		return "Исправен"
	}
}

func normRepairStatus(s string) string {
	// подстрой, если в БД другие значения
	switch strings.TrimSpace(s) {
	case "Открыта":
		return "Открыта"
	case "В работе", "В_работе", "В процессе":
		return "В работе"
	case "Закрыта", "Завершена":
		return "Закрыта"
	default:
		return "Открыта"
	}
}

// ---------------- API: зоны для селекта ----------------

func GetZonesForSelect(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `SELECT "id_зоны","Название" FROM "Зона" ORDER BY "id_зоны"`)
    if err != nil {
        return jsonError(c, 500, "Ошибка чтения зон", err)
    }
    defer rows.Close()

	type z struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var list []z
	for rows.Next() {
		var it z
		if err := rows.Scan(&it.ID, &it.Name); err == nil {
			list = append(list, it)
		}
	}
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора зон", err)
    }
    return jsonOK(c, fiber.Map{"zones": list})
}

// ---------------- Получить оборудование по ID (для модалки) ----------------

func GetEquipmentByID(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()
    var (
        zoneID                 int
        name, status, zoneName string
        purchase, lastTO       sql.NullTime
    )
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
        SELECT e."id_зоны",
               e."Название",
               e."Дата_покупки",
               e."Дата_последнего_ТО",
               e."Статус",
               z."Название" AS zone_name
        FROM "Оборудование" e
        JOIN "Зона" z ON z."id_зоны" = e."id_зоны"
        WHERE e."id_оборудования"=$1
    `, id).Scan(&zoneID, &name, &purchase, &lastTO, &status, &zoneName)
    if errors.Is(err, sql.ErrNoRows) {
        return jsonError(c, 404, "Оборудование не найдено", nil)
    }
    if err != nil {
        return jsonError(c, 500, "Ошибка БД", err)
    }
    return jsonOK(c, fiber.Map{
        "item": fiber.Map{
            "ID":              id,
            "ZoneID":          zoneID,
            "Name":            name,
            "PurchaseDate":    dateYMD(purchase),
            "LastServiceDate": dateYMD(lastTO),
            "Status":          status,
            "ZoneName":        zoneName,
        },
    })
}

// ---------------- Страница оборудования ----------------

func GetEquipmentPage(c *fiber.Ctx) error {
    db := database.GetDB()

    // Оборудование
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT e."id_оборудования",
               e."id_зоны",
               e."Название",
               e."Дата_покупки",
               e."Дата_последнего_ТО",
               e."Статус",
               (e."Фото" IS NOT NULL) AS has_photo,
               z."Название" AS zone_name
        FROM "Оборудование" e
        JOIN "Зона" z ON z."id_зоны" = e."id_зоны"
        ORDER BY e."id_оборудования"
    `)
	if err != nil {
		log.Printf("equipment list error: %v", err)
		return c.Render("equipment", fiber.Map{
			"Title":        "Оборудование",
			"Items":        []fiber.Map{},
			"Repairs":      []fiber.Map{},
			"Error":        "Ошибка загрузки оборудования: " + err.Error(),
			"ExtraScripts": templateScript("/static/js/equipment.js"),
		})
	}
	defer rows.Close()

	var items []fiber.Map
	for rows.Next() {
		var (
			id, zoneID             int
			name, status, zoneName string
			hasPhoto               bool
			purchase, lastTO       sql.NullTime
		)
		if err := rows.Scan(&id, &zoneID, &name, &purchase, &lastTO, &status, &hasPhoto, &zoneName); err != nil {
			continue
		}
		items = append(items, fiber.Map{
			"ID":              id,
			"ZoneID":          zoneID,
			"Name":            name,
			"PurchaseDate":    dateYMD(purchase),
			"LastServiceDate": dateYMD(lastTO),
			"Status":          status,
			"HasPhoto":        hasPhoto,
			"ZoneName":        zoneName,
		})
	}
	_ = rows.Err()

	// Последние заявки на ремонт
    r2, err := db.QueryContext(ctx, `
        SELECT r."id_заявки",
               r."id_оборудования",
               r."Дата_создания",
               r."Описание_проблемы",
               r."Статус",
               r."Приоритет",
               (r."Фото" IS NOT NULL) AS has_photo,
               e."Название" AS eq_name
        FROM "Заявка_на_ремонт" r
        JOIN "Оборудование" e ON e."id_оборудования" = r."id_оборудования"
        ORDER BY r."id_заявки" DESC
        LIMIT 10
    `)
	var repairs []fiber.Map
	if err == nil {
		defer r2.Close()
		for r2.Next() {
			var (
				id, eqID        int
				desc, status    string
				priority        string
				hasPhoto        bool
				created         time.Time
				eqName          string
			)
			if err := r2.Scan(&id, &eqID, &created, &desc, &status, &priority, &hasPhoto, &eqName); err != nil {
				continue
			}
			repairs = append(repairs, fiber.Map{
				"ID":            id,
				"EquipmentID":   eqID,
				"EquipmentName": eqName,
				"CreatedAt":     created,
				"Description":   desc,
				"Status":        status,
				"Priority":      priority,
				"HasPhoto":      hasPhoto,
			})
		}
	}

	return c.Render("equipment", fiber.Map{
		"Title":        "Оборудование",
		"Items":        items,
		"Repairs":      repairs,
		"ExtraScripts": templateScript("/static/js/equipment.js"),
	})
}

// ---------------- Create ----------------

func CreateEquipment(c *fiber.Ctx) error {
	type formT struct {
		ZoneID   int    `form:"zone_id"`
		Name     string `form:"name"`
		Purchase string `form:"purchase_date"`     // YYYY-MM-DD
		LastTO   string `form:"last_service_date"` // YYYY-MM-DD
		Status   string `form:"status"`
	}
	var f formT
	if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
	}
	if f.ZoneID <= 0 || f.Name == "" {
        return jsonError(c, 400, "Укажите зону и название", nil)
	}
	if f.Status == "" {
		f.Status = "Исправен"
	}
	f.Status = normEqStatus(f.Status)

	var purchase, lastTO sql.NullTime
	if f.Purchase != "" {
		if t, err := time.Parse("2006-01-02", f.Purchase); err == nil {
			purchase = sql.NullTime{Time: t, Valid: true}
		}
	}
	if f.LastTO != "" {
		if t, err := time.Parse("2006-01-02", f.LastTO); err == nil {
			lastTO = sql.NullTime{Time: t, Valid: true}
		}
	}

	db := database.GetDB()
    var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Оборудование" ("id_зоны","Название","Дата_покупки","Дата_последнего_ТО","Статус")
        VALUES ($1,$2,$3,$4,$5)
        RETURNING "id_оборудования"
    `, f.ZoneID, f.Name, nullableTimeArg(purchase), nullableTimeArg(lastTO), f.Status).Scan(&id)
    if err != nil {
        return jsonError(c, 500, "Ошибка сохранения", err)
    }
    return jsonOK(c, fiber.Map{"message": "Оборудование создано", "id": id})
}

// ---------------- Update ----------------

func UpdateEquipment(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
	type formT struct {
		ZoneID   int    `form:"zone_id"`
		Name     string `form:"name"`
		Purchase string `form:"purchase_date"`
		LastTO   string `form:"last_service_date"`
		Status   string `form:"status"`
	}
	var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.ZoneID <= 0 || f.Name == "" || f.Status == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }
	f.Status = normEqStatus(f.Status)

	var purchase, lastTO sql.NullTime
	if f.Purchase != "" {
		if t, err := time.Parse("2006-01-02", f.Purchase); err == nil {
			purchase = sql.NullTime{Time: t, Valid: true}
		}
	}
	if f.LastTO != "" {
		if t, err := time.Parse("2006-01-02", f.LastTO); err == nil {
			lastTO = sql.NullTime{Time: t, Valid: true}
		}
	}

	db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Оборудование"
        SET "id_зоны"=$2, "Название"=$3, "Дата_покупки"=$4, "Дата_последнего_ТО"=$5, "Статус"=$6
        WHERE "id_оборудования"=$1
    `, id, f.ZoneID, f.Name, nullableTimeArg(purchase), nullableTimeArg(lastTO), f.Status)
    if err != nil {
        return jsonError(c, 500, "Ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Оборудование не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Оборудование обновлено"})
}

// ---------------- Delete ----------------

func DeleteEquipment(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Оборудование" WHERE "id_оборудования"=$1`, id)
    if err != nil {
        // возможно, FK из "Заявка_на_ремонт"
        return jsonError(c, 500, "Ошибка удаления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Оборудование не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Удалено"})
}

// ---------------- Фото оборудования ----------------

func UploadEquipmentPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
	fh, err := c.FormFile("photo")
    if err != nil {
        return jsonError(c, 400, "Файл не получен", err)
    }
    if fh.Size <= 0 || fh.Size > maxUpload {
        return jsonError(c, 413, "Файл пустой или больше 5 МБ", nil)
    }
	f, err := fh.Open()
    if err != nil {
        return jsonError(c, 500, "Не удалось открыть файл", err)
    }
	defer f.Close()

	lr := &io.LimitedReader{R: f, N: maxUpload + 1}
	buf, err := io.ReadAll(lr)
    if err != nil {
        return jsonError(c, 500, "Ошибка чтения файла", err)
    }
    if int64(len(buf)) > maxUpload {
        return jsonError(c, 413, "Файл превышает 5 МБ", nil)
    }
	head := buf
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)
	switch ct {
	case "image/jpeg", "image/png", "image/webp":
    default:
        return jsonError(c, 400, "Разрешены JPEG/PNG/WebP", nil)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    _, err = db.ExecContext(ctx, `UPDATE "Оборудование" SET "Фото"=$2 WHERE "id_оборудования"=$1`, id, buf)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка сохранения", err)
    }
    return jsonOK(c, fiber.Map{"message": "Фото загружено"})
}

func GetEquipmentPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).SendString("Некорректный id")
	}
    db := database.GetDB()
    var img []byte
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `SELECT "Фото" FROM "Оборудование" WHERE "id_оборудования"=$1`, id).Scan(&img)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(404).SendString("Оборудование не найдено")
	}
	if err != nil {
		return c.Status(500).SendString("DB: ошибка чтения")
	}
	if len(img) == 0 {
		return c.Status(404).SendString("Фото отсутствует")
	}
	head := img
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)
	if !strings.HasPrefix(ct, "image/") {
		ct = "application/octet-stream"
	}
	c.Set("Content-Type", ct)
	sum := sha256.Sum256(img)
	c.Set("ETag", fmt.Sprintf(`W/"%x"`, sum[:16]))
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(img)
}

func DeleteEquipmentPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
	}
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `UPDATE "Оборудование" SET "Фото"=NULL WHERE "id_оборудования"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Оборудование не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Фото удалено"})
}

// ---------------- Заявка на ремонт ----------------

func CreateRepairRequest(c *fiber.Ctx) error {
	eqID, _ := strconv.Atoi(c.FormValue("eq_id"))
	desc := c.FormValue("description")
	priority := c.FormValue("priority")
	if eqID <= 0 || strings.TrimSpace(desc) == "" {
        return jsonError(c, 400, "Укажите оборудование и описание", nil)
	}
	if priority == "" {
		priority = "Средний"
	}

	// опциональное фото
	var photo []byte
	if fh, err := c.FormFile("photo"); err == nil && fh != nil && fh.Size > 0 {
		if fh.Size > maxUpload {
            return jsonError(c, fiber.StatusRequestEntityTooLarge, "Фото больше 5 МБ", nil)
		}
		f, err := fh.Open()
		if err != nil {
            return jsonError(c, fiber.StatusInternalServerError, "Не удалось открыть фото", err)
		}
		defer f.Close()
		lr := &io.LimitedReader{R: f, N: maxUpload + 1}
		buf, err := io.ReadAll(lr)
		if err != nil {
            return jsonError(c, fiber.StatusInternalServerError, "Ошибка чтения фото", err)
		}
		if int64(len(buf)) > maxUpload {
            return jsonError(c, fiber.StatusRequestEntityTooLarge, "Фото превышает 5 МБ", nil)
		}
		head := buf
		if len(head) > 512 {
			head = head[:512]
		}
		ct := http.DetectContentType(head)
		switch ct {
		case "image/jpeg", "image/png", "image/webp":
		default:
            return jsonError(c, fiber.StatusBadRequest, "Фото: только JPEG/PNG/WebP", nil)
		}
		photo = buf
	}

	// ВАЖНО: не указываем колонку "Статус" — сработает DEFAULT в БД, который соответствует CHECK
    db := database.GetDB()
    var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Заявка_на_ремонт"
        ("id_оборудования","Дата_создания","Описание_проблемы","Приоритет","Фото")
        VALUES ($1, NOW(), $2, $3, $4)
        RETURNING "id_заявки"
    `, eqID, desc, priority, nullablePhoto(photo)).Scan(&id)
    if err != nil {
        return jsonError(c, 500, "Ошибка создания заявки", err)
    }
    // Перевести оборудование в статус "На ремонте"
    _, _ = db.ExecContext(ctx, `UPDATE "Оборудование" SET "Статус"=$2 WHERE "id_оборудования"=$1`, eqID, "На ремонте")
    return jsonOK(c, fiber.Map{"message": "Заявка создана", "id": id})
}

// ---------- Удалить заявку на ремонт ----------
func DeleteRepairRequest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
	}
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Заявка_на_ремонт" WHERE "id_заявки"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "Ошибка удаления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Заявка не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Заявка удалена"})
}


// ---------- Обновить заявку на ремонт ----------
func UpdateRepairRequest(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    type formT struct {
        EquipmentID int    `form:"equipment_id"`
        Description string `form:"description"`
        Status      string `form:"status"`
        Priority    string `form:"priority"`
    }
    var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    f.Description = strings.TrimSpace(f.Description)
    if f.Description == "" {
        return jsonError(c, 400, "Описание обязательно", nil)
    }
    // нормализуем статус заявки и приоритет
    st := normRepairStatus(f.Status)
    pr := strings.TrimSpace(f.Priority)
    switch pr {
    case "Низкий", "Средний", "Высокий":
    default:
        pr = "Средний"
    }

    // собираем динамический UPDATE
    sets := []string{"\"Описание_проблемы\"=$2", "\"Статус\"=$3", "\"Приоритет\"=$4"}
    args := []any{id, f.Description, st, pr}
    if f.EquipmentID > 0 {
        sets = append(sets, "\"id_оборудования\"=$5")
        args = append(args, f.EquipmentID)
    }
    sqlUpd := "UPDATE \"Заявка_на_ремонт\" SET " + strings.Join(sets, ", ") + " WHERE \"id_заявки\"=$1"

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, sqlUpd, args...)
    if err != nil {
        return jsonError(c, 500, "Ошибка обновления заявки", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Заявка не найдена", nil)
    }
    // Обновление статуса оборудования в зависимости от статуса заявки
    // Определим equipmentID: если не передан — прочитаем из БД
    eqID := f.EquipmentID
    if eqID <= 0 {
        _ = db.QueryRowContext(ctx, `SELECT "id_оборудования" FROM "Заявка_на_ремонт" WHERE "id_заявки"=$1`, id).Scan(&eqID)
    }
    if eqID > 0 {
        if st == "Открыта" || st == "В работе" {
            // Переводим оборудование в "На ремонте"
            _, _ = db.ExecContext(ctx, `UPDATE "Оборудование" SET "Статус"='На ремонте' WHERE "id_оборудования"=$1`, eqID)
        } else if st == "Закрыта" {
            // Если нет других активных заявок, вернуть статус в "Исправен"
            var cnt int
            _ = db.QueryRowContext(ctx, `
                SELECT COUNT(*) FROM "Заявка_на_ремонт"
                WHERE "id_оборудования"=$1 AND "Статус" IN ('Открыта','В работе')
            `, eqID).Scan(&cnt)
            if cnt == 0 {
                _, _ = db.ExecContext(ctx, `UPDATE "Оборудование" SET "Статус"='Исправен' WHERE "id_оборудования"=$1`, eqID)
            }
        }
    }
    return jsonOK(c, fiber.Map{"message": "Заявка обновлена"})
}

// ---------- Загрузить/заменить фото заявки ----------
func UploadRepairPhoto(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    fh, err := c.FormFile("photo")
    if err != nil {
        return jsonError(c, fiber.StatusBadRequest, "Файл не получен (photo)", err)
    }
    if fh.Size <= 0 || fh.Size > maxUpload {
        return jsonError(c, fiber.StatusRequestEntityTooLarge, "Файл пустой или больше 5 МБ", nil)
    }
    f, err := fh.Open()
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "Не удалось открыть файл", err)
    }
    defer f.Close()
    lr := &io.LimitedReader{R: f, N: maxUpload + 1}
    buf, err := io.ReadAll(lr)
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "Ошибка чтения файла", err)
    }
    if int64(len(buf)) > maxUpload {
        return jsonError(c, fiber.StatusRequestEntityTooLarge, "Файл превышает 5 МБ", nil)
    }
    head := buf
    if len(head) > 512 { head = head[:512] }
    ct := http.DetectContentType(head)
    switch ct {
    case "image/jpeg", "image/png", "image/webp":
    default:
        return jsonError(c, fiber.StatusBadRequest, "Разрешены JPEG/PNG/WebP", nil)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `UPDATE "Заявка_на_ремонт" SET "Фото"=$2 WHERE "id_заявки"=$1`, id, buf)
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "DB: ошибка сохранения", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, fiber.StatusNotFound, "Заявка не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Фото загружено"})
}


func GetLatestRepairs(c *fiber.Ctx) error {
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT r."id_заявки",
               r."id_оборудования",
               r."Дата_создания",
               r."Описание_проблемы",
               r."Статус",
               r."Приоритет",
               (r."Фото" IS NOT NULL) AS has_photo,
               e."Название" AS eq_name
        FROM "Заявка_на_ремонт" r
        JOIN "Оборудование" e ON e."id_оборудования" = r."id_оборудования"
        ORDER BY r."id_заявки" DESC
        LIMIT 10
    `)
    if err != nil {
        return jsonError(c, 500, "Ошибка загрузки заявок", err)
    }
	defer rows.Close()
	type row struct {
		ID            int       `json:"id"`
		EquipmentID   int       `json:"equipment_id"`
		EquipmentName string    `json:"equipment_name"`
		CreatedAt     time.Time `json:"created_at"`
		Status        string    `json:"status"`
		Priority      string    `json:"priority"`
		HasPhoto      bool      `json:"has_photo"`
	}
	var list []row
	for rows.Next() {
		var r row
		var desc string
		if err := rows.Scan(&r.ID, &r.EquipmentID, &r.CreatedAt, &desc, &r.Status, &r.Priority, &r.HasPhoto, &r.EquipmentName); err == nil {
			list = append(list, r)
		}
	}
    return jsonOK(c, fiber.Map{"repairs": list})
}

func GetRepairPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).SendString("Некорректный id")
	}
    db := database.GetDB()
    var img []byte
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `SELECT "Фото" FROM "Заявка_на_ремонт" WHERE "id_заявки"=$1`, id).Scan(&img)
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(404).SendString("Заявка не найдена")
	}
	if err != nil {
		return c.Status(500).SendString("DB: ошибка чтения")
	}
	if len(img) == 0 {
		return c.Status(404).SendString("Фото отсутствует")
	}
	head := img
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)
	if !strings.HasPrefix(ct, "image/") {
		ct = "application/octet-stream"
	}
	c.Set("Content-Type", ct)
	sum := sha256.Sum256(img)
	c.Set("ETag", fmt.Sprintf(`W/"%x"`, sum[:16]))
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(img)
}

// ---------------- API v1: Список оборудования (JSON) ----------------

func APIv1ListEquipment(c *fiber.Ctx) error {
    db := database.GetDB()

    // простые фильтры (необязательные)
    zoneID := strings.TrimSpace(c.Query("zone_id"))
    status := strings.TrimSpace(c.Query("status"))
    hasPhoto := c.Query("has_photo") // "1" или "0"

    where := []string{}
    args := []any{}
    nextPH := func() string { return "$" + strconv.Itoa(len(args)+1) }

    if zoneID != "" {
        where = append(where, `e."id_зоны" = `+nextPH()+`::int`)
        args = append(args, zoneID)
    }
    if status != "" {
        where = append(where, `e."Статус" = `+nextPH())
        args = append(args, status)
    }
    if hasPhoto == "1" {
        where = append(where, `e."Фото" IS NOT NULL`)
    } else if hasPhoto == "0" {
        where = append(where, `e."Фото" IS NULL`)
    }

    query := `
        SELECT e."id_оборудования",
               e."id_зоны",
               e."Название",
               e."Дата_покупки",
               e."Дата_последнего_ТО",
               e."Статус",
               (e."Фото" IS NOT NULL) AS has_photo,
               z."Название" AS zone_name
        FROM "Оборудование" e
        JOIN "Зона" z ON z."id_зоны" = e."id_зоны"`
    if len(where) > 0 {
        query += " WHERE " + strings.Join(where, " AND ")
    }
    query += ` ORDER BY e."id_оборудования"`

    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return jsonError(c, 500, "Ошибка загрузки оборудования", err)
    }
    defer rows.Close()

    type item struct {
        ID              int    `json:"id"`
        ZoneID          int    `json:"zone_id"`
        Name            string `json:"name"`
        PurchaseDate    string `json:"purchase_date"`
        LastServiceDate string `json:"last_service_date"`
        Status          string `json:"status"`
        HasPhoto        bool   `json:"has_photo"`
        ZoneName        string `json:"zone_name"`
    }
    var list []item
    for rows.Next() {
        var (
            id, zoneIDv       int
            name, statusV     string
            zoneName          string
            hasPhotoV         bool
            purchase, lastTO  sql.NullTime
        )
        if err := rows.Scan(&id, &zoneIDv, &name, &purchase, &lastTO, &statusV, &hasPhotoV, &zoneName); err != nil {
            return jsonError(c, 500, "Ошибка чтения оборудования", err)
        }
        list = append(list, item{
            ID: id,
            ZoneID: zoneIDv,
            Name: name,
            PurchaseDate: dateYMD(purchase),
            LastServiceDate: dateYMD(lastTO),
            Status: statusV,
            HasPhoto: hasPhotoV,
            ZoneName: zoneName,
        })
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка курсора", err)
    }
    return jsonOK(c, fiber.Map{"items": list})
}
