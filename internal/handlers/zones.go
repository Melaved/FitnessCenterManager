package handlers

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/models"
)

const maxUpload = 5 * 1024 * 1024 // 5MB

// ==== helpers ===================================================================================

var allowedStatuses = map[string]bool{
	"Доступна":    true,
	"На ремонте":  true,
	"Закрыта":     true,
}

func validateZoneInput(name string, capacity int, status string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("Название зоны обязательно")
	}
	if capacity <= 0 {
		return fmt.Errorf("Вместимость должна быть положительным числом")
	}
	if !allowedStatuses[status] {
		return fmt.Errorf("Недопустимый статус (допустимы: Доступна, На ремонте, Закрыта)")
	}
	return nil
}

// ==== READ (list + one) ========================================================================

// GetZones — страница/список зон (рендер шаблона)
func GetZones(c *fiber.Ctx) error {
	db := database.GetDB()
	log.Println("🔍 Получение зон из БД...")

	rows, err := db.Query(`
		SELECT 
			"id_зоны",
			"Название", 
			"Описание",
			"Вместимость",
			"Статус",
			("Фото" IS NOT NULL) AS has_photo
		FROM "Зона" 
		ORDER BY "id_зоны" DESC
	`)
	if err != nil {
		log.Printf("❌ Ошибка получения зон: %v", err)
		return c.Render("zones", fiber.Map{
			"Title": "Зоны",
			"Zones": []models.Zone{},
			"Error": "Не удалось загрузить данные зон: " + err.Error(),
		})
	}
	defer rows.Close()

	var zones []models.Zone
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(
			&z.ID,
			&z.Name,
			&z.Description,
			&z.Capacity,
			&z.Status,
			&z.HasPhoto,
		); err != nil {
			log.Printf("❌ Ошибка сканирования зоны: %v", err)
			continue
		}
		zones = append(zones, z)
	}
	if err = rows.Err(); err != nil {
		log.Printf("❌ Ошибка после итерации по зонам: %v", err)
	}

	log.Printf("✅ Загружено %d зон из БД", len(zones))
	return c.Render("zones", fiber.Map{
		"Title": "Зоны",
		"Zones": zones,
	})
}

// GetZoneByID — JSON-эндпоинт одной зоны (для формы редактирования)
/*
GET /api/zones/:id
{
  "success": true,
  "zone": { ... }
}
*/
func GetZoneByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	db := database.GetDB()
	var z models.Zone
	err = db.QueryRow(`
		SELECT 
			"id_зоны", "Название", "Описание", "Вместимость", "Статус",
			("Фото" IS NOT NULL) AS has_photo
		FROM "Зона" WHERE "id_зоны"=$1
	`, id).Scan(&z.ID, &z.Name, &z.Description, &z.Capacity, &z.Status, &z.HasPhoto)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Зона не найдена"})
	case err != nil:
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка чтения"})
	}

	return c.JSON(fiber.Map{"success": true, "zone": z})
}

// ==== CREATE ====================================================================================

// CreateZone — создать новую зону (ожидается form-data из модалки)
func CreateZone(c *fiber.Ctx) error {
	log.Println("🎯 Создание новой зоны...")

	type form struct {
		Name        string `form:"name"`
		Description string `form:"description"`
		Capacity    int    `form:"capacity"`
		Status      string `form:"status"`
	}
	var f form
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false, "error": "Неверные данные формы: " + err.Error(),
		})
	}
	if err := validateZoneInput(f.Name, f.Capacity, f.Status); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	db := database.GetDB()
	var zoneID int
	if err := db.QueryRow(`
		INSERT INTO "Зона" ("Название","Описание","Вместимость","Статус")
		VALUES ($1,$2,$3,$4)
		RETURNING "id_зоны"
	`, f.Name, f.Description, f.Capacity, f.Status).Scan(&zoneID); err != nil {
		log.Printf("❌ Ошибка создания зоны: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка создания зоны: " + err.Error()})
	}

	log.Printf("✅ Создана зона: %s (ID: %d)", f.Name, zoneID)
	return c.JSON(fiber.Map{"success": true, "message": "Зона успешно создана", "zone_id": zoneID})
}

// ==== UPDATE ====================================================================================

// UpdateZone — изменить зону по id (ожидается form-data или x-www-form-urlencoded)
func UpdateZone(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	type form struct {
		Name        string `form:"name"`
		Description string `form:"description"`
		Capacity    int    `form:"capacity"`
		Status      string `form:"status"`
	}
	var f form
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы: " + err.Error()})
	}
	if err := validateZoneInput(f.Name, f.Capacity, f.Status); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	db := database.GetDB()
	res, err := db.Exec(`
		UPDATE "Зона"
		SET "Название"=$2, "Описание"=$3, "Вместимость"=$4, "Статус"=$5
		WHERE "id_зоны"=$1
	`, id, f.Name, f.Description, f.Capacity, f.Status)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка обновления"})
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Зона не найдена"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Зона обновлена"})
}

// ClearZonePhoto — установить Фото = NULL
func ClearZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`UPDATE "Зона" SET "Фото"=NULL WHERE "id_зоны"=$1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка обновления"})
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Зона не найдена"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Фото удалено"})
}

// ==== DELETE ====================================================================================

// DeleteZone — удалить зону по id
func DeleteZone(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()

	res, err := db.Exec(`DELETE FROM "Зона" WHERE "id_зоны"=$1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "DB: ошибка удаления"})
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Зона не найдена"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Зона удалена"})
}

// ==== upload/read photo =========================================================================

// UploadZonePhoto — загрузить фото (bytea) для зоны
func UploadZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "Некорректный id зоны",
		})
	}

	fh, err := c.FormFile("photo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "Файл не получен (ожидается form-data: photo)",
		})
	}
	if fh.Size <= 0 || fh.Size > maxUpload {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"success": false, "error": "Файл пустой или больше 5 МБ",
		})
	}

	file, err := fh.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "Не удалось открыть файл",
		})
	}
	defer file.Close()

	lr := &io.LimitedReader{R: file, N: maxUpload + 1}
	buf, err := io.ReadAll(lr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "Ошибка чтения файла",
		})
	}
	if int64(len(buf)) > maxUpload {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
			"success": false, "error": "Файл превышает 5 МБ",
		})
	}

	head := buf
	if len(head) > 512 {
		head = head[:512]
	}
	mime := http.DetectContentType(head)
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false, "error": "Разрешены JPEG/PNG/WebP",
		})
	}

	db := database.GetDB()
	res, err := db.Exec(`UPDATE "Зона" SET "Фото"=$2 WHERE "id_зоны"=$1`, id, buf)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false, "error": "DB: ошибка сохранения",
		})
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false, "error": "Зона не найдена",
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Фото загружено"})
}

// GetZonePhoto — отдать фото зоны для <img>
func GetZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Некорректный id зоны")
	}

	db := database.GetDB()
	var img []byte
	err = db.QueryRow(`SELECT "Фото" FROM "Зона" WHERE "id_зоны"=$1`, id).Scan(&img)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return c.Status(fiber.StatusNotFound).SendString("Зона не найдена")
	case err != nil:
		return c.Status(fiber.StatusInternalServerError).SendString("DB: ошибка чтения")
	}
	if len(img) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("Фото отсутствует")
	}

	head := img
	if len(head) > 512 {
		head = head[:512]
	}
	mime := http.DetectContentType(head)
	if !strings.HasPrefix(mime, "image/") {
		mime = "application/octet-stream"
	}
	c.Set("Content-Type", mime)

	sum := sha256.Sum256(img)
	etag := fmt.Sprintf(`W/"%x"`, sum[:16])
	c.Set("ETag", etag)
	if inm := c.Get("If-None-Match"); inm != "" && inm == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}

	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	return c.Send(img)
}
