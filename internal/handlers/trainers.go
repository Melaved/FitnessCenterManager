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

	rows, err := db.Query(`
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
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.FIO == "" || f.Phone == "" || f.HireDate == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "ФИО, телефон и дата найма обязательны"})
	}
	hire, err := time.Parse("2006-01-02", f.HireDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата найма"})
	}

	db := database.GetDB()
	var id int
	err = db.QueryRow(`
		INSERT INTO "Тренер" ("ФИО","Номер_телефона","Специализация","Дата_найма","Стаж_работы")
		VALUES ($1,$2,$3,$4,$5)
		RETURNING "id_тренера"
	`, f.FIO, f.Phone, f.Specialization, hire, f.Experience).Scan(&id)
	if err != nil {
		log.Printf("❌ create trainer: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка сохранения тренера"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Тренер добавлен", "id": id})
}

func GetTrainerByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	var t models.Trainer
	err = db.QueryRow(`
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
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Тренер не найден"})
	}
	if err != nil {
		log.Printf("❌ get trainer: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка БД"})
	}
	resp := fiber.Map{
		"id":             t.ID,
		"fio":            t.FIO,
		"phone":          t.Phone,
		"specialization": t.Specialization,
		"hire_date":      t.HireDate.Format("2006-01-02"),
		"experience":     t.Experience,
	}
	return c.JSON(fiber.Map{"success": true, "trainer": resp})
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
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.FIO == "" || f.Phone == "" || f.HireDate == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "ФИО, телефон и дата найма обязательны"})
	}
	hire, err := time.Parse("2006-01-02", f.HireDate)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная дата найма"})
	}

	db := database.GetDB()
	res, err := db.Exec(`
		UPDATE "Тренер"
		SET "ФИО"=$2, "Номер_телефона"=$3, "Специализация"=$4, "Дата_найма"=$5, "Стаж_работы"=$6
		WHERE "id_тренера"=$1
	`, id, f.FIO, f.Phone, f.Specialization, hire, f.Experience)
	if err != nil {
		log.Printf("❌ update trainer: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка обновления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Тренер не найден"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Данные тренера обновлены"})
}

// ======================= DELETE =======================
func DeleteTrainer(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()

	// Если на тренера ссылаются тренировки, тут может быть FK.
	res, err := db.Exec(`DELETE FROM "Тренер" WHERE "id_тренера"=$1`, id)
	if err != nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "error": "Невозможно удалить: есть связанные тренировки"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Тренер не найден"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Тренер удалён"})
}

func GetTrainersForSelect(c *fiber.Ctx) error {
	db := database.GetDB()

	rows, err := db.Query(`
		SELECT "id_тренера", "ФИО"
		FROM "Тренер"
		ORDER BY "ФИО"
	`)
	if err != nil {
		log.Printf("❌ trainers-for-select: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка чтения тренеров",
		})
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
	return c.JSON(fiber.Map{"success": true, "trainers": out})
}