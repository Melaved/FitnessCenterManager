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
	rows, err := db.Query(`SELECT "id_зоны","Название" FROM "Зона" ORDER BY "id_зоны"`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка чтения зон: " + err.Error()})
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
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка курсора зон: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "zones": list})
}

// ---------------- Получить оборудование по ID (для модалки) ----------------

func GetEquipmentByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	var (
		zoneID                 int
		name, status, zoneName string
		purchase, lastTO       sql.NullTime
	)
	err = db.QueryRow(`
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
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Оборудование не найдено"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка БД: " + err.Error()})
	}
	return c.JSON(fiber.Map{
		"success": true,
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
	rows, err := db.Query(`
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
	r2, err := db.Query(`
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
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.ZoneID <= 0 || f.Name == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Укажите зону и название"})
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
	err := db.QueryRow(`
		INSERT INTO "Оборудование" ("id_зоны","Название","Дата_покупки","Дата_последнего_ТО","Статус")
		VALUES ($1,$2,$3,$4,$5)
		RETURNING "id_оборудования"
	`, f.ZoneID, f.Name, nullableTimeArg(purchase), nullableTimeArg(lastTO), f.Status).Scan(&id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка сохранения: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Оборудование создано", "id": id})
}

// ---------------- Update ----------------

func UpdateEquipment(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
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
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.ZoneID <= 0 || f.Name == "" || f.Status == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
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
	res, err := db.Exec(`
		UPDATE "Оборудование"
		SET "id_зоны"=$2, "Название"=$3, "Дата_покупки"=$4, "Дата_последнего_ТО"=$5, "Статус"=$6
		WHERE "id_оборудования"=$1
	`, id, f.ZoneID, f.Name, nullableTimeArg(purchase), nullableTimeArg(lastTO), f.Status)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка обновления: " + err.Error()})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Оборудование не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Оборудование обновлено"})
}

// ---------------- Delete ----------------

func DeleteEquipment(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`DELETE FROM "Оборудование" WHERE "id_оборудования"=$1`, id)
	if err != nil {
		// возможно, FK из "Заявка_на_ремонт"
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка удаления: " + err.Error()})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Оборудование не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Удалено"})
}

// ---------------- Фото оборудования ----------------

func UploadEquipmentPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	fh, err := c.FormFile("photo")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Файл не получен"})
	}
	if fh.Size <= 0 || fh.Size > maxUpload {
		return c.Status(413).JSON(fiber.Map{"success": false, "error": "Файл пустой или больше 5 МБ"})
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Не удалось открыть файл"})
	}
	defer f.Close()

	lr := &io.LimitedReader{R: f, N: maxUpload + 1}
	buf, err := io.ReadAll(lr)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка чтения файла"})
	}
	if int64(len(buf)) > maxUpload {
		return c.Status(413).JSON(fiber.Map{"success": false, "error": "Файл превышает 5 МБ"})
	}
	head := buf
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)
	switch ct {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Разрешены JPEG/PNG/WebP"})
	}

	db := database.GetDB()
	_, err = db.Exec(`UPDATE "Оборудование" SET "Фото"=$2 WHERE "id_оборудования"=$1`, id, buf)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка сохранения"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Фото загружено"})
}

func GetEquipmentPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).SendString("Некорректный id")
	}
	db := database.GetDB()
	var img []byte
	err = db.QueryRow(`SELECT "Фото" FROM "Оборудование" WHERE "id_оборудования"=$1`, id).Scan(&img)
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
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`UPDATE "Оборудование" SET "Фото"=NULL WHERE "id_оборудования"=$1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка обновления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Оборудование не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Фото удалено"})
}

// ---------------- Заявка на ремонт ----------------

func CreateRepairRequest(c *fiber.Ctx) error {
	eqID, _ := strconv.Atoi(c.FormValue("eq_id"))
	desc := c.FormValue("description")
	priority := c.FormValue("priority")
	if eqID <= 0 || strings.TrimSpace(desc) == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Укажите оборудование и описание"})
	}
	if priority == "" {
		priority = "Средний"
	}

	// опциональное фото
	var photo []byte
	if fh, err := c.FormFile("photo"); err == nil && fh != nil && fh.Size > 0 {
		if fh.Size > maxUpload {
			return c.Status(413).JSON(fiber.Map{"success": false, "error": "Фото больше 5 МБ"})
		}
		f, err := fh.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "Не удалось открыть фото"})
		}
		defer f.Close()
		lr := &io.LimitedReader{R: f, N: maxUpload + 1}
		buf, err := io.ReadAll(lr)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка чтения фото"})
		}
		if int64(len(buf)) > maxUpload {
			return c.Status(413).JSON(fiber.Map{"success": false, "error": "Фото превышает 5 МБ"})
		}
		head := buf
		if len(head) > 512 {
			head = head[:512]
		}
		ct := http.DetectContentType(head)
		switch ct {
		case "image/jpeg", "image/png", "image/webp":
		default:
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Фото: только JPEG/PNG/WebP"})
		}
		photo = buf
	}

	// ВАЖНО: не указываем колонку "Статус" — сработает DEFAULT в БД, который соответствует CHECK
	db := database.GetDB()
	var id int
	err := db.QueryRow(`
		INSERT INTO "Заявка_на_ремонт"
		("id_оборудования","Дата_создания","Описание_проблемы","Приоритет","Фото")
		VALUES ($1, NOW(), $2, $3, $4)
		RETURNING "id_заявки"
	`, eqID, desc, priority, nullablePhoto(photo)).Scan(&id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка создания заявки: " + err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Заявка создана", "id": id})
}

// ---------- Удалить заявку на ремонт ----------
func DeleteRepairRequest(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`DELETE FROM "Заявка_на_ремонт" WHERE "id_заявки"=$1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка удаления: " + err.Error()})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Заявка не найдена"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Заявка удалена"})
}


func GetLatestRepairs(c *fiber.Ctx) error {
	db := database.GetDB()
	rows, err := db.Query(`
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
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка загрузки заявок: " + err.Error()})
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
	return c.JSON(fiber.Map{"success": true, "repairs": list})
}

func GetRepairPhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).SendString("Некорректный id")
	}
	db := database.GetDB()
	var img []byte
	err = db.QueryRow(`SELECT "Фото" FROM "Заявка_на_ремонт" WHERE "id_заявки"=$1`, id).Scan(&img)
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
