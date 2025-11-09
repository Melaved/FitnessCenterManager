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

func tplScript(src string) template.HTML { // маленький помощник для layout'а
	return template.HTML(fmt.Sprintf(`<script src="%s"></script>`, src))
}

func GetTrainersPage(c *fiber.Ctx) error {
    db := database.GetDB()

    ctx, cancel := withDBTimeout()
    defer cancel()

    rows, err := db.QueryContext(ctx, `
        SELECT 
            "id_тренера",
            "ФИО",
            "Номер_телефона",
            "Специализация",
            "Дата_найма",
            "Стаж_работы"
        FROM "Тренер"
        ORDER BY "id_тренера" DESC
    `)
	if err != nil {
		log.Printf("❌ trainers list error: %v", err)
		return c.Render("trainers", fiber.Map{
			"Title":        "Тренеры",
			"Trainers":     []models.Trainer{},
			"Message":      "Не удалось загрузить список тренеров",
			"ExtraScripts": tplScript(`/static/js/trainers.js`),
		})
	}
	defer rows.Close()

	var list []models.Trainer
	for rows.Next() {
		var t models.Trainer
		if err := rows.Scan(
			&t.ID,
			&t.FIO,
			&t.Phone,
			&t.Specialization,
			&t.HireDate,
			&t.Experience,
		); err != nil {
			log.Printf("❌ scan trainer: %v", err)
			continue
		}
		list = append(list, t)
	}
	if err = rows.Err(); err != nil {
		log.Printf("❌ trainers rows err: %v", err)
	}

	return c.Render("trainers", fiber.Map{
		"Title":        "Тренеры",
		"Trainers":     list,
		"ExtraScripts": tplScript(`/static/js/trainers.js`),
	})
}

func CreateTrainer(c *fiber.Ctx) error {
	type formT struct {
		FIO            string `form:"fio"`
		Phone          string `form:"phone"`
		Specialization string `form:"specialization"`
		HireDate       string `form:"hire_date"` // YYYY-MM-DD
		Experience     int    `form:"experience"`
	}
	var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.FIO == "" || f.Phone == "" || f.HireDate == "" {
        return jsonError(c, 400, "ФИО, телефон и дата найма обязательны", nil)
    }
    hire, err := time.Parse("2006-01-02", f.HireDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата найма", err)
    }

	db := database.GetDB()
	var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
        INSERT INTO "Тренер" ("ФИО","Номер_телефона","Специализация","Дата_найма","Стаж_работы")
        VALUES ($1,$2,$3,$4,$5)
        RETURNING "id_тренера"
    `, f.FIO, f.Phone, f.Specialization, hire, f.Experience).Scan(&id)
    if err != nil {
        log.Printf("❌ create trainer: %v", err)
        return jsonError(c, 500, "Ошибка сохранения тренера", err)
    }
    return jsonOK(c, fiber.Map{"message": "Тренер добавлен", "id": id})
}

func GetTrainerByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()
    var t models.Trainer
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
        SELECT 
            "id_тренера",
            "ФИО",
            "Номер_телефона",
            "Специализация",
            "Дата_найма",
            "Стаж_работы"
        FROM "Тренер"
        WHERE "id_тренера"=$1
    `, id).Scan(
        &t.ID, &t.FIO, &t.Phone, &t.Specialization, &t.HireDate, &t.Experience,
    )
    if err == sql.ErrNoRows {
        return jsonError(c, 404, "Тренер не найден", nil)
    }
    if err != nil {
        log.Printf("❌ get trainer: %v", err)
        return jsonError(c, 500, "Ошибка БД", err)
    }
	resp := fiber.Map{
		"id":             t.ID,
		"fio":            t.FIO,
		"phone":          t.Phone,
		"specialization": t.Specialization,
		"hire_date":      t.HireDate.Format("2006-01-02"),
		"experience":     t.Experience,
	}
    return jsonOK(c, fiber.Map{"trainer": resp})
}

// ======================= UPDATE =======================
func UpdateTrainer(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}

	type formT struct {
		FIO            string `form:"fio"`
		Phone          string `form:"phone"`
		Specialization string `form:"specialization"`
		HireDate       string `form:"hire_date"` // YYYY-MM-DD
		Experience     int    `form:"experience"`
	}
	var f formT
    if err := c.BodyParser(&f); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.FIO == "" || f.Phone == "" || f.HireDate == "" {
        return jsonError(c, 400, "ФИО, телефон и дата найма обязательны", nil)
    }
    hire, err := time.Parse("2006-01-02", f.HireDate)
    if err != nil {
        return jsonError(c, 400, "Неверная дата найма", err)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Тренер"
        SET "ФИО"=$2, "Номер_телефона"=$3, "Специализация"=$4, "Дата_найма"=$5, "Стаж_работы"=$6
        WHERE "id_тренера"=$1
    `, id, f.FIO, f.Phone, f.Specialization, hire, f.Experience)
    if err != nil {
        log.Printf("❌ update trainer: %v", err)
        return jsonError(c, 500, "Ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Тренер не найден", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Данные тренера обновлены"})
}

// ======================= DELETE =======================
func DeleteTrainer(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
    if err != nil || id <= 0 {
        return jsonError(c, 400, "Некорректный id", err)
    }
    db := database.GetDB()

    // Если на тренера ссылаются тренировки, тут может быть FK.
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Тренер" WHERE "id_тренера"=$1`, id)
    if err != nil {
        return jsonError(c, 409, "Невозможно удалить: есть связанные тренировки", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Тренер не найден", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Тренер удалён"})
}

func GetTrainersForSelect(c *fiber.Ctx) error {
	db := database.GetDB()

    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT "id_тренера", "ФИО"
        FROM "Тренер"
        ORDER BY "ФИО"
    `)
    if err != nil {
        log.Printf("❌ trainers-for-select: %v", err)
        return jsonError(c, 500, "Ошибка чтения тренеров", err)
    }
	defer rows.Close()

	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var out []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Name); err == nil {
			out = append(out, it)
		}
	}
    return jsonOK(c, fiber.Map{"trainers": out})
}
