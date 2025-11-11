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

	"github.com/gofiber/fiber/v2"

	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/models"
)

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

// GetZones — страница/список зон (рендер шаблона)
func GetZones(c *fiber.Ctx) error {
	db := database.GetDB()
	log.Println("🔍 Получение зон из БД...")

	ctx, cancel := withDBTimeout()
	defer cancel()
	rows, err := db.QueryContext(ctx, `
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
		"ExtraScripts": template.HTML(`<script src="/static/js/zones.js"></script>`),
	})
}

func GetZoneByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }

	db := database.GetDB()
	var z models.Zone
	ctx, cancel := withDBTimeout()
	defer cancel()
	err = db.QueryRowContext(ctx, `
		SELECT 
			"id_зоны", "Название", "Описание", "Вместимость", "Статус",
			("Фото" IS NOT NULL) AS has_photo
		FROM "Зона" WHERE "id_зоны"=$1
	`, id).Scan(&z.ID, &z.Name, &z.Description, &z.Capacity, &z.Status, &z.HasPhoto)
    switch {
    case errors.Is(err, sql.ErrNoRows):
        return jsonError(c, 404, "Зона не найдена", nil)
    case err != nil:
        return jsonError(c, 500, "DB: ошибка чтения", err)
    }

    return jsonOK(c, fiber.Map{"zone": z})
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if err := validateZoneInput(f.Name, f.Capacity, f.Status); err != nil {
        return jsonError(c, 400, err.Error(), nil)
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
        log.Printf("❌ Ошибка создания зоны: %v", err)
        return jsonError(c, 500, "Ошибка создания зоны", err)
    }

	log.Printf("✅ Создана зона: %s (ID: %d)", f.Name, zoneID)
	return c.JSON(fiber.Map{"success": true, "message": "Зона успешно создана", "zone_id": zoneID})
}

// ==== UPDATE ====================================================================================

// UpdateZone — изменить зону по id (ожидается form-data или x-www-form-urlencoded)
func UpdateZone(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }

	type form struct {
		Name        string `form:"name"`
		Description string `form:"description"`
		Capacity    int    `form:"capacity"`
		Status      string `form:"status"`
	}
	var f form
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if err := validateZoneInput(f.Name, f.Capacity, f.Status); err != nil {
        return jsonError(c, 400, err.Error(), nil)
    }

	db := database.GetDB()
	ctx, cancel := withDBTimeout()
	defer cancel()
	res, err := db.ExecContext(ctx, `
		UPDATE "Зона"
		SET "Название"=$2, "Описание"=$3, "Вместимость"=$4, "Статус"=$5
		WHERE "id_зоны"=$1
	`, id, f.Name, f.Description, f.Capacity, f.Status)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка обновления", err)
    }
    aff, _ := res.RowsAffected()
    if aff == 0 {
        return jsonError(c, 404, "Зона не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Зона обновлена"})
}

// ClearZonePhoto — установить Фото = NULL
func ClearZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
	db := database.GetDB()
	ctx, cancel := withDBTimeout()
	defer cancel()
	res, err := db.ExecContext(ctx, `UPDATE "Зона" SET "Фото"=NULL WHERE "id_зоны"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка обновления", err)
    }
    if rows, _ := res.RowsAffected(); rows == 0 {
        return jsonError(c, 404, "Зона не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Фото удалено"})
}

// ==== DELETE ====================================================================================

// DeleteZone — удалить зону по id
func DeleteZone(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
	db := database.GetDB()

	ctx, cancel := withDBTimeout()
	defer cancel()
	res, err := db.ExecContext(ctx, `DELETE FROM "Зона" WHERE "id_зоны"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "DB: ошибка удаления", err)
    }
	aff, _ := res.RowsAffected()
    if aff == 0 {
        return jsonError(c, 404, "Зона не найдена", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Зона удалена"})
}

// ==== upload/read photo =========================================================================

// UploadZonePhoto — загрузить фото (bytea) для зоны
func UploadZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, fiber.StatusBadRequest, "Некорректный id зоны", err)
    }

	fh, err := c.FormFile("photo")
    if err != nil {
        return jsonError(c, fiber.StatusBadRequest, "Файл не получен (ожидается form-data: photo)", err)
    }
    if fh.Size <= 0 || fh.Size > maxUpload {
        return jsonError(c, fiber.StatusRequestEntityTooLarge, "Файл пустой или больше 5 МБ", nil)
    }

	file, err := fh.Open()
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "Не удалось открыть файл", err)
    }
	defer file.Close()

	lr := &io.LimitedReader{R: file, N: maxUpload + 1}
	buf, err := io.ReadAll(lr)
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "Ошибка чтения файла", err)
    }
    if int64(len(buf)) > maxUpload {
        return jsonError(c, fiber.StatusRequestEntityTooLarge, "Файл превышает 5 МБ", nil)
    }

	head := buf
	if len(head) > 512 {
		head = head[:512]
	}
	mime := http.DetectContentType(head)
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
    default:
        return jsonError(c, fiber.StatusBadRequest, "Разрешены JPEG/PNG/WebP", nil)
    }

	db := database.GetDB()
	ctx, cancel := withDBTimeout()
	defer cancel()
	res, err := db.ExecContext(ctx, `UPDATE "Зона" SET "Фото"=$2 WHERE "id_зоны"=$1`, id, buf)
    if err != nil {
        return jsonError(c, fiber.StatusInternalServerError, "DB: ошибка сохранения", err)
    }
    if rows, _ := res.RowsAffected(); rows == 0 {
        return jsonError(c, fiber.StatusNotFound, "Зона не найдена", nil)
    }

    return jsonOK(c, fiber.Map{"message": "Фото загружено"})
}

// GetZonePhoto — отдать фото зоны для <img>
func GetZonePhoto(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return c.Status(fiber.StatusBadRequest).SendString("Некорректный id зоны")
    }

	db := database.GetDB()
	var img []byte
	ctx, cancel := withDBTimeout()
	defer cancel()
	err = db.QueryRowContext(ctx, `SELECT "Фото" FROM "Зона" WHERE "id_зоны"=$1`, id).Scan(&img)
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
