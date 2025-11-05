package handlers

import (
	"database/sql"
	"fitness-center-manager/internal/database"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// для записи на групповую: нужен список абонементов (id + «ФИО (абонемент #)»)
func GetSubscriptionsForSelect(c *fiber.Ctx) error {
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT s."id_абонемента", c."ФИО", s."Статус"
		FROM "Абонемент" s
		JOIN "Клиент" c ON c."id_клиента" = s."id_клиента"
		ORDER BY s."id_абонемента" DESC
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка загрузки абонементов"})
	}
	defer rows.Close()
	type item struct{ ID int; Label string }
	var out []item
	for rows.Next() {
		var id int
		var fio, st string
		if err := rows.Scan(&id, &fio, &st); err == nil {
			out = append(out, item{ID: id, Label: fmt.Sprintf("%s (абонемент #%d, %s)", fio, id, st)})
		}
	}
	return c.JSON(fiber.Map{"success": true, "subscriptions": out})
}

// ====== Страница тренировок (сводка) ======

func GetTrainingsPage(c *fiber.Ctx) error {
	db := database.GetDB()

	// ---- параметры фильтра ----
	q := strings.TrimSpace(c.Query("q"))                 // общий поиск
	qTrainer := strings.TrimSpace(c.Query("trainer_id")) // ID тренера
	qZone := strings.TrimSpace(c.Query("zone_id"))       // ID зоны (групповые)
	qLevel := strings.TrimSpace(c.Query("level"))        // Начальный|Средний|Продвинутый (групповые)
	qStatus := strings.TrimSpace(c.Query("status"))      // Запланирована|Завершена|Отменена (персональные)
	qFrom := strings.TrimSpace(c.Query("from"))          // 2006-01-02 / 2006-01-02T15:04
	qTo := strings.TrimSpace(c.Query("to"))
	onlyUpcoming := c.Query("upcoming") == "1"
	recent30 := c.Query("recent") == "1"

	// ================== ГРУППОВЫЕ ==================
	whereG := []string{}
	argsG := []any{}
	nextPHG := func(n int) []string {
		start := len(argsG) + 1
		ph := make([]string, n)
		for i := 0; i < n; i++ {
			ph[i] = "$" + strconv.Itoa(start+i)
		}
		return ph
	}

	queryG := `
		SELECT g."id_групповой_тренировки",
		       g."Название",
		       COALESCE(g."Описание",''),
		       g."Максимум_участников",
		       g."Время_начала",
		       g."Время_окончания",
		       COALESCE(g."Уровень_сложности",''),
		       t."ФИО" AS trainer_name, t."id_тренера",
		       z."Название" AS zone_name,  z."id_зоны"
		FROM "Групповая_тренировка" g
		JOIN "Тренер" t ON t."id_тренера" = g."id_тренера"
		JOIN "Зона"   z ON z."id_зоны"   = g."id_зоны"
	`

	if q != "" {
		like := "%" + q + "%"
		ph := nextPHG(3)
		whereG = append(whereG, `(
			g."Название" ILIKE `+ph[0]+` OR
			t."ФИО"      ILIKE `+ph[1]+` OR
			z."Название" ILIKE `+ph[2]+`
		)`)
		argsG = append(argsG, like, like, like)
	}
	if qTrainer != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `g."id_тренера" = `+ph[0]+`::int`)
		argsG = append(argsG, qTrainer)
	}
	if qZone != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `g."id_зоны" = `+ph[0]+`::int`)
		argsG = append(argsG, qZone)
	}
	if qLevel != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `g."Уровень_сложности" = `+ph[0])
		argsG = append(argsG, qLevel)
	}
	if qFrom != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `g."Время_начала" >= `+ph[0]+`::timestamp`)
		argsG = append(argsG, qFrom)
	}
	if qTo != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `g."Время_начала" <= `+ph[0]+`::timestamp`)
		argsG = append(argsG, qTo)
	}
	if onlyUpcoming {
		ph := nextPHG(1)
		whereG = append(whereG, `g."Время_начала" >= `+ph[0]+`::timestamp`)
		argsG = append(argsG, time.Now())
	}
	if recent30 {
		whereG = append(whereG, `g."Время_начала" >= NOW() - INTERVAL '30 days'`)
	}

	if len(whereG) > 0 {
		queryG += " WHERE " + strings.Join(whereG, " AND ")
	}
	queryG += ` ORDER BY g."Время_начала" DESC, g."id_групповой_тренировки" DESC`

	var groups []fiber.Map
	if gr, err := db.Query(queryG, argsG...); err != nil {
		log.Printf("groups list err: %v", err)
	} else {
		defer gr.Close()
		for gr.Next() {
			var (
				id, max int
				title, desc, level, trainerName, zoneName string
				start, end time.Time
				trainerID, zoneID int
			)
			if err := gr.Scan(&id, &title, &desc, &max, &start, &end, &level, &trainerName, &trainerID, &zoneName, &zoneID); err == nil {
				groups = append(groups, fiber.Map{
					"ID": id, "Title": title, "Description": desc, "Max": max,
					"Start": start, "End": end, "Level": level,
					"TrainerName": trainerName, "TrainerID": trainerID,
					"ZoneName": zoneName, "ZoneID": zoneID,
				})
			}
		}
	}

	// ================== ПЕРСОНАЛЬНЫЕ ==================
	whereP := []string{}
	argsP := []any{}
	nextPHP := func(n int) []string {
		start := len(argsP) + 1
		ph := make([]string, n)
		for i := 0; i < n; i++ {
			ph[i] = "$" + strconv.Itoa(start+i)
		}
		return ph
	}

	queryP := `
		SELECT p."id_персональной_тренировки",
		       p."Время_начала", p."Время_окончания",
		       p."Статус", COALESCE(p."Стоимость",0),
		       p."id_абонемента", a."id_клиента", c."ФИО",
		       t."id_тренера", t."ФИО"
		FROM "Персональная_тренировка" p
		JOIN "Абонемент" a ON a."id_абонемента" = p."id_абонемента"
		JOIN "Клиент"    c ON c."id_клиента"    = a."id_клиента"
		JOIN "Тренер"    t ON t."id_тренера"    = p."id_тренера"
	`

	if q != "" {
		like := "%" + q + "%"
		ph := nextPHP(2)
		whereP = append(whereP, `(
			c."ФИО" ILIKE `+ph[0]+` OR
			t."ФИО" ILIKE `+ph[1]+`
		)`)
		argsP = append(argsP, like, like)
	}
	if qTrainer != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `p."id_тренера" = `+ph[0]+`::int`)
		argsP = append(argsP, qTrainer)
	}
	if qStatus != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `p."Статус" = `+ph[0])
		argsP = append(argsP, qStatus)
	}
	if qFrom != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `p."Время_начала" >= `+ph[0]+`::timestamp`)
		argsP = append(argsP, qFrom)
	}
	if qTo != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `p."Время_начала" <= `+ph[0]+`::timestamp`)
		argsP = append(argsP, qTo)
	}
	if onlyUpcoming {
		ph := nextPHP(1)
		whereP = append(whereP, `p."Время_начала" >= `+ph[0]+`::timestamp`)
		argsP = append(argsP, time.Now())
	}
	if recent30 {
		whereP = append(whereP, `p."Время_начала" >= NOW() - INTERVAL '30 days'`)
	}

	if len(whereP) > 0 {
		queryP += " WHERE " + strings.Join(whereP, " AND ")
	}
	queryP += ` ORDER BY p."Время_начала" DESC, p."id_персональной_тренировки" DESC`

	var personal []fiber.Map
	if pr, err := db.Query(queryP, argsP...); err == nil {
		defer pr.Close()
		for pr.Next() {
			var (
				id int
				start, end time.Time
				status string
				price float64
				subID, clientID, trainerID int
				clientFIO, trainerFIO string
			)
			if err := pr.Scan(&id, &start, &end, &status, &price, &subID, &clientID, &clientFIO, &trainerID, &trainerFIO); err == nil {
				personal = append(personal, fiber.Map{
					"ID": id, "Start": start, "End": end, "Status": status, "Price": price,
					"SubscriptionID": subID, "ClientID": clientID, "ClientFIO": clientFIO,
					"TrainerID": trainerID, "TrainerFIO": trainerFIO,
				})
			}
		}
	} else {
		log.Printf("personal list err: %v", err)
	}

	return c.Render("trainings", fiber.Map{
		"Title":    "Тренировки",
		"Groups":   groups,
		"Personal": personal,
		"Filter": fiber.Map{
			"q":          q,
			"trainer_id": qTrainer,
			"zone_id":    qZone,
			"level":      qLevel,
			"status":     qStatus,
			"from":       qFrom,
			"to":         qTo,
			"upcoming":   onlyUpcoming,
			"recent":     recent30,
		},
		"ExtraScripts": templateScript("/static/js/trainings.js"),
	})
}

// ====== CRUD: Групповые ======

func GetGroupTrainingByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	var (
		title, desc, level string
		max, trainerID, zoneID int
		start, end time.Time
	)
	err := db.QueryRow(`
		SELECT "Название", COALESCE("Описание",''), COALESCE("Уровень_сложности",''), 
		       COALESCE("Максимум_участников",0),
		       "Время_начала","Время_окончания",
		       "id_тренера","id_зоны"
		FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1
	`, id).Scan(&title, &desc, &level, &max, &start, &end, &trainerID, &zoneID)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка БД"})
	}
	return c.JSON(fiber.Map{"success": true, "item": fiber.Map{
		"ID": id, "Title": title, "Description": desc, "Level": level, "Max": max,
		"Date": start.Format("2006-01-02"),
		"StartTime": start.Format("15:04"),
		"EndTime":   end.Format("15:04"),
		"TrainerID": trainerID, "ZoneID": zoneID,
	}})
}

func CreateGroupTraining(c *fiber.Ctx) error {
	type fT struct {
		Title   string `form:"title"`
		Desc    string `form:"description"`
		Max     int    `form:"max"`
		Level   string `form:"level"` // Начальный | Средний | Продвинутый
		Date    string `form:"date"`
		Start   string `form:"start_time"`
		End     string `form:"end_time"`
		Trainer int    `form:"trainer_id"`
		Zone    int    `form:"zone_id"`
	}
	var f fT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.Title == "" || f.Date == "" || f.Start == "" || f.End == "" || f.Trainer <= 0 || f.Zone <= 0 || f.Max <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}
	switch f.Level {
	case "", "Начальный", "Средний", "Продвинутый":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверный уровень сложности"})
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
	if err1 != nil || err2 != nil || !end.After(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректное время начала/окончания"})
	}
	db := database.GetDB()
	var id int
	err := db.QueryRow(`
		INSERT INTO "Групповая_тренировка"
		("id_тренера","id_зоны","Название","Описание","Максимум_участников","Время_начала","Время_окончания","Уровень_сложности")
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING "id_групповой_тренировки"
	`, f.Trainer, f.Zone, f.Title, nullIfEmpty(f.Desc), f.Max, start, end, nullIfEmpty(f.Level)).Scan(&id)
	if err != nil {
		log.Printf("create group err: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка сохранения"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id, "message": "Групповая тренировка создана"})
}

func UpdateGroupTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	type fT struct {
		Title   string `form:"title"`
		Desc    string `form:"description"`
		Max     int    `form:"max"`
		Level   string `form:"level"`
		Date    string `form:"date"`
		Start   string `form:"start_time"`
		End     string `form:"end_time"`
		Trainer int    `form:"trainer_id"`
		Zone    int    `form:"zone_id"`
	}
	var f fT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.Title == "" || f.Date == "" || f.Start == "" || f.End == "" || f.Trainer <= 0 || f.Zone <= 0 || f.Max <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}
	switch f.Level {
	case "", "Начальный", "Средний", "Продвинутый":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверный уровень сложности"})
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
	if err1 != nil || err2 != nil || !end.After(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректное время начала/окончания"})
	}

	db := database.GetDB()
	res, err := db.Exec(`
		UPDATE "Групповая_тренировка"
		SET "id_тренера"=$2,"id_зоны"=$3,"Название"=$4,"Описание"=$5,"Максимум_участников"=$6,
		    "Время_начала"=$7,"Время_окончания"=$8,"Уровень_сложности"=$9
		WHERE "id_групповой_тренировки"=$1
	`, id, f.Trainer, f.Zone, f.Title, nullIfEmpty(f.Desc), f.Max, start, end, nullIfEmpty(f.Level))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка обновления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Обновлено"})
}

func DeleteGroupTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`DELETE FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1`, id)
	if err != nil {
		// связанные записи в "Запись_на_групповую_тренировку" могут мешать, но там ON DELETE CASCADE — должно удалиться
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка удаления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Удалено"})
}

// ====== CRUD: Персональные ======

func GetPersonalTrainingByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	var (
		start, end time.Time
		status string
		price sql.NullFloat64
		subID, trainerID int
	)
	err := db.QueryRow(`
		SELECT "Время_начала","Время_окончания","Статус",COALESCE("Стоимость",0),"id_абонемента","id_тренера"
		FROM "Персональная_тренировка" WHERE "id_персональной_тренировки"=$1
	`, id).Scan(&start, &end, &status, &price, &subID, &trainerID)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка БД"})
	}
	return c.JSON(fiber.Map{"success": true, "item": fiber.Map{
		"ID": id,
		"Date": start.Format("2006-01-02"),
		"StartTime": start.Format("15:04"),
		"EndTime": end.Format("15:04"),
		"Status": status,
		"Price": fmt.Sprintf("%.2f", price.Float64),
		"SubscriptionID": subID,
		"TrainerID": trainerID,
	}})
}

func CreatePersonalTraining(c *fiber.Ctx) error {
	type fT struct {
		Subscription int    `form:"subscription_id"`
		Trainer      int    `form:"trainer_id"`
		Date         string `form:"date"`
		Start        string `form:"start_time"`
		End          string `form:"end_time"`
		Status       string `form:"status"` // Запланирована | Завершена | Отменена
		Price        string `form:"price"`
	}
	var f fT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.Subscription <= 0 || f.Trainer <= 0 || f.Date == "" || f.Start == "" || f.End == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}
	switch f.Status {
	case "", "Запланирована", "Завершена", "Отменена":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверный статус"})
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
	if err1 != nil || err2 != nil || !end.After(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректное время начала/окончания"})
	}
	var price *float64
	if f.Price != "" {
		if p, err := strconv.ParseFloat(f.Price, 64); err == nil {
			price = &p
		} else {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная стоимость"})
		}
	}
	db := database.GetDB()
	var id int
	err := db.QueryRow(`
		INSERT INTO "Персональная_тренировка"
		("id_абонемента","id_тренера","Время_начала","Время_окончания","Статус","Стоимость")
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING "id_персональной_тренировки"
	`, f.Subscription, f.Trainer, start, end, coalesceStr(f.Status, "Запланирована"), nullablePrice(price)).Scan(&id)
	if err != nil {
		log.Printf("create personal err: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка сохранения"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id, "message": "Персональная тренировка создана"})
}

func UpdatePersonalTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	type fT struct {
		Subscription int    `form:"subscription_id"`
		Trainer      int    `form:"trainer_id"`
		Date         string `form:"date"`
		Start        string `form:"start_time"`
		End          string `form:"end_time"`
		Status       string `form:"status"`
		Price        string `form:"price"`
	}
	var f fT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.Subscription <= 0 || f.Trainer <= 0 || f.Date == "" || f.Start == "" || f.End == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Заполните обязательные поля"})
	}
	switch f.Status {
	case "Запланирована", "Завершена", "Отменена":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверный статус"})
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
	if err1 != nil || err2 != nil || !end.After(start) {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректное время начала/окончания"})
	}
	var price *float64
	if f.Price != "" {
		if p, err := strconv.ParseFloat(f.Price, 64); err == nil {
			price = &p
		} else {
			return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверная стоимость"})
		}
	}
	db := database.GetDB()
	res, err := db.Exec(`
		UPDATE "Персональная_тренировка"
		SET "id_абонемента"=$2,"id_тренера"=$3,"Время_начала"=$4,"Время_окончания"=$5,"Статус"=$6,"Стоимость"=$7
		WHERE "id_персональной_тренировки"=$1
	`, id, f.Subscription, f.Trainer, start, end, f.Status, nullablePrice(price))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка обновления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Обновлено"})
}

func DeletePersonalTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	if id <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Некорректный id"})
	}
	db := database.GetDB()
	res, err := db.Exec(`DELETE FROM "Персональная_тренировка" WHERE "id_персональной_тренировки"=$1`, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка удаления"})
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "Не найдено"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Удалено"})
}

// ====== Запись на групповую ======

func CreateGroupEnrollment(c *fiber.Ctx) error {
	type fT struct {
		GroupID int `form:"group_id"`
		SubID   int `form:"subscription_id"`
		Status  string `form:"status"` // опц., по умолчанию 'Записан'
	}
	var f fT
	if err := c.BodyParser(&f); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверные данные формы"})
	}
	if f.GroupID <= 0 || f.SubID <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Выберите тренировку и абонемент"})
	}
	switch f.Status {
	case "", "Записан", "Посетил", "Отменил":
	default:
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Неверный статус записи"})
	}

	db := database.GetDB()
	// проверим, что групповая существует
	var exists int
	if err := db.QueryRow(`SELECT 1 FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1`, f.GroupID).Scan(&exists); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Групповая тренировка не найдена"})
	}
	// и абонемент существует
	if err := db.QueryRow(`SELECT 1 FROM "Абонемент" WHERE "id_абонемента"=$1`, f.SubID).Scan(&exists); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "Абонемент не найден"})
	}

	var id int
	err := db.QueryRow(`
		INSERT INTO "Запись_на_групповую_тренировку"
		("id_групповой_тренировки","id_абонемента","Статус")
		VALUES ($1,$2,$3)
		RETURNING "id_записи"
	`, f.GroupID, f.SubID, coalesceStr(f.Status, "Записан")).Scan(&id)
	if err != nil {
		log.Printf("enrollment err: %v", err)
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Не удалось создать запись (возможно, дубликат)"})
	}
	return c.JSON(fiber.Map{"success": true, "id": id, "message": "Запись создана"})
}

// ====== helpers ======

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func coalesceStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
func nullablePrice(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
