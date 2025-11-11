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
        return jsonError(c, 500, "Ошибка загрузки абонементов", err)
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
	q          := strings.TrimSpace(c.Query("q"))          // общий поиск
	qTrainer   := strings.TrimSpace(c.Query("trainer_id")) // ID тренера
	qZone      := strings.TrimSpace(c.Query("zone_id"))    // ID зоны (групповые)
	qLevel     := strings.TrimSpace(c.Query("level"))      // Начальный|Средний|Продвинутый (групповые)
	qStatus    := strings.TrimSpace(c.Query("status"))     // Запланирована|Завершена|Отменена (персональные)
	qFrom      := strings.TrimSpace(c.Query("from"))       // 2006-01-02 / 2006-01-02T15:04
	qTo        := strings.TrimSpace(c.Query("to"))
	onlyUpcoming := c.Query("upcoming") == "1"
	recent30     := c.Query("recent") == "1"

	// ================== ГРУППОВЫЕ (из vw_group_training_with_slots) ==================
	whereG := []string{}
	argsG  := []any{}
	nextPHG := func(n int) []string {
		start := len(argsG) + 1
		ph := make([]string, n)
		for i := 0; i < n; i++ {
			ph[i] = "$" + strconv.Itoa(start+i)
		}
		return ph
	}

	queryG := `
		SELECT 
			v."id_групповой_тренировки",
			v."Название",
			COALESCE(v."Описание",'')               AS description,
			COALESCE(v."Максимум_участников",0)     AS max,
			v."Время_начала",
			v."Время_окончания",
			COALESCE(v."Уровень_сложности",'')      AS level,
			v.trainer_name,
			v."id_тренера",
			v.zone_name,
			v."id_зоны",
			v.free_slots                             -- вычисляемая
		FROM vw_group_training_with_slots v
	`

	if q != "" {
		like := "%" + q + "%"
		ph := nextPHG(3)
		whereG = append(whereG, `(
			v."Название"    ILIKE `+ph[0]+` OR
			v.trainer_name  ILIKE `+ph[1]+` OR
			v.zone_name     ILIKE `+ph[2]+`
		)`)
		argsG = append(argsG, like, like, like)
	}
	if qTrainer != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `v."id_тренера" = `+ph[0]+`::int`)
		argsG = append(argsG, qTrainer)
	}
	if qZone != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `v."id_зоны" = `+ph[0]+`::int`)
		argsG = append(argsG, qZone)
	}
	if qLevel != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `v."Уровень_сложности" = `+ph[0])
		argsG = append(argsG, qLevel)
	}
	if qFrom != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `v."Время_начала" >= `+ph[0]+`::timestamp`)
		argsG = append(argsG, qFrom)
	}
	if qTo != "" {
		ph := nextPHG(1)
		whereG = append(whereG, `v."Время_начала" <= `+ph[0]+`::timestamp`)
		argsG = append(argsG, qTo)
	}
	if onlyUpcoming {
		ph := nextPHG(1)
		whereG = append(whereG, `v."Время_начала" >= `+ph[0]+`::timestamp`)
		argsG = append(argsG, time.Now())
	}
	if recent30 {
		whereG = append(whereG, `v."Время_начала" >= NOW() - INTERVAL '30 days'`)
	}

	if len(whereG) > 0 {
		queryG += " WHERE " + strings.Join(whereG, " AND ")
	}
	queryG += ` ORDER BY v."Время_начала" DESC, v."id_групповой_тренировки" DESC`

	var groups []fiber.Map
	{
		ctx, cancel := withDBTimeout()
		defer cancel()
		rows, err := db.QueryContext(ctx, queryG, argsG...)
		if err != nil {
			log.Printf("groups list err: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var (
					id, max, trainerID, zoneID int
					title, desc, level, trainerName, zoneName string
					start, end time.Time
					freeSlots int
				)
				if err := rows.Scan(
					&id, &title, &desc, &max, &start, &end, &level,
					&trainerName, &trainerID, &zoneName, &zoneID, &freeSlots,
				); err == nil {
					groups = append(groups, fiber.Map{
						"ID": id, "Title": title, "Description": desc, "Max": max,
						"Start": start, "End": end, "Level": level,
						"TrainerName": trainerName, "TrainerID": trainerID,
						"ZoneName": zoneName, "ZoneID": zoneID,
						"FreeSlots": freeSlots, // можно вывести в UI при желании
					})
				}
			}
		}
	}

	// ================== ПЕРСОНАЛЬНЫЕ (из vw_personal_training_enriched) ==================
	whereP := []string{}
	argsP  := []any{}
	nextPHP := func(n int) []string {
		start := len(argsP) + 1
		ph := make([]string, n)
		for i := 0; i < n; i++ {
			ph[i] = "$" + strconv.Itoa(start+i)
		}
		return ph
	}

	queryP := `
		SELECT 
			v."id_персональной_тренировки",
			v."Время_начала",
			v."Время_окончания",
			v."Статус",
			COALESCE(v."Стоимость",0)     AS price,
			v."id_абонемента",
			v."id_клиента",
			v.client_fio,
			v."id_тренера",
			v.trainer_fio
			-- v.duration_minutes, v.is_upcoming   -- есть во вью, если захочешь — добери
		FROM vw_personal_training_enriched v
	`

	if q != "" {
		like := "%" + q + "%"
		ph := nextPHP(2)
		whereP = append(whereP, `(
			v.client_fio  ILIKE `+ph[0]+` OR
			v.trainer_fio ILIKE `+ph[1]+`
		)`)
		argsP = append(argsP, like, like)
	}
	if qTrainer != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `v."id_тренера" = `+ph[0]+`::int`)
		argsP = append(argsP, qTrainer)
	}
	if qStatus != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `v."Статус" = `+ph[0])
		argsP = append(argsP, qStatus)
	}
	if qFrom != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `v."Время_начала" >= `+ph[0]+`::timestamp`)
		argsP = append(argsP, qFrom)
	}
	if qTo != "" {
		ph := nextPHP(1)
		whereP = append(whereP, `v."Время_начала" <= `+ph[0]+`::timestamp`)
		argsP = append(argsP, qTo)
	}
	if onlyUpcoming {
		ph := nextPHP(1)
		whereP = append(whereP, `v."Время_начала" >= `+ph[0]+`::timestamp`)
		argsP = append(argsP, time.Now())
	}
	if recent30 {
		whereP = append(whereP, `v."Время_начала" >= NOW() - INTERVAL '30 days'`)
	}

	if len(whereP) > 0 {
		queryP += " WHERE " + strings.Join(whereP, " AND ")
	}
	queryP += ` ORDER BY v."Время_начала" DESC, v."id_персональной_тренировки" DESC`

	var personal []fiber.Map
	{
		ctx, cancel := withDBTimeout()
		defer cancel()
		rows, err := db.QueryContext(ctx, queryP, argsP...)
		if err != nil {
			log.Printf("personal list err: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var (
					id, subID, clientID, trainerID int
					start, end time.Time
					status string
					price float64
					clientFIO, trainerFIO string
				)
				if err := rows.Scan(
					&id, &start, &end, &status, &price,
					&subID, &clientID, &clientFIO, &trainerID, &trainerFIO,
				); err == nil {
					personal = append(personal, fiber.Map{
						"ID": id, "Start": start, "End": end, "Status": status, "Price": price,
						"SubscriptionID": subID, "ClientID": clientID, "ClientFIO": clientFIO,
						"TrainerID": trainerID, "TrainerFIO": trainerFIO,
					})
				}
			}
		}
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

// ---------------- API v1: Списки тренировок (JSON) ----------------

// APIv1ListGroupTrainings — JSON-список групповых тренировок с фильтрами
func APIv1ListGroupTrainings(c *fiber.Ctx) error {
    db := database.GetDB()
    q := strings.TrimSpace(c.Query("q"))
    qTrainer := strings.TrimSpace(c.Query("trainer_id"))
    qZone := strings.TrimSpace(c.Query("zone_id"))
    qLevel := strings.TrimSpace(c.Query("level"))
    qFrom := strings.TrimSpace(c.Query("from"))
    qTo := strings.TrimSpace(c.Query("to"))
    onlyUpcoming := c.Query("upcoming") == "1"
    recent30 := c.Query("recent") == "1"

    where := []string{}
    args := []any{}
    nextPH := func(n int) []string {
        start := len(args) + 1
        ph := make([]string, n)
        for i := 0; i < n; i++ { ph[i] = "$" + strconv.Itoa(start+i) }
        return ph
    }

    query := `
        SELECT 
            v."id_групповой_тренировки",
            v."Название",
            COALESCE(v."Описание",'')               AS description,
            COALESCE(v."Максимум_участников",0)     AS max,
            v."Время_начала",
            v."Время_окончания",
            COALESCE(v."Уровень_сложности",'')      AS level,
            v.trainer_name,
            v."id_тренера",
            v.zone_name,
            v."id_зоны",
            v.free_slots
        FROM vw_group_training_with_slots v`

    if q != "" {
        like := "%" + q + "%"
        ph := nextPH(3)
        where = append(where, `(
            v."Название" ILIKE `+ph[0]+` OR
            v.trainer_name ILIKE `+ph[1]+` OR
            v.zone_name ILIKE `+ph[2]+`
        )`)
        args = append(args, like, like, like)
    }
    if qTrainer != "" {
        ph := nextPH(1)
        where = append(where, `v."id_тренера" = `+ph[0]+`::int`)
        args = append(args, qTrainer)
    }
    if qZone != "" {
        ph := nextPH(1)
        where = append(where, `v."id_зоны" = `+ph[0]+`::int`)
        args = append(args, qZone)
    }
    if qLevel != "" {
        ph := nextPH(1)
        where = append(where, `v."Уровень_сложности" = `+ph[0])
        args = append(args, qLevel)
    }
    if qFrom != "" {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" >= `+ph[0]+`::timestamp`)
        args = append(args, qFrom)
    }
    if qTo != "" {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" <= `+ph[0]+`::timestamp`)
        args = append(args, qTo)
    }
    if onlyUpcoming {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" >= `+ph[0]+`::timestamp`)
        args = append(args, time.Now())
    }
    if recent30 {
        where = append(where, `v."Время_начала" >= NOW() - INTERVAL '30 days'`)
    }
    if len(where) > 0 {
        query += " WHERE " + strings.Join(where, " AND ")
    }
    query += ` ORDER BY v."Время_начала" DESC, v."id_групповой_тренировки" DESC`

    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil { return jsonError(c, 500, "Ошибка загрузки групповых тренировок", err) }
    defer rows.Close()

    type dto struct {
        ID          int       `json:"id"`
        Title       string    `json:"title"`
        Description string    `json:"description"`
        Max         int       `json:"max"`
        Start       time.Time `json:"start"`
        End         time.Time `json:"end"`
        Level       string    `json:"level"`
        TrainerName string    `json:"trainer_name"`
        TrainerID   int       `json:"trainer_id"`
        ZoneName    string    `json:"zone_name"`
        ZoneID      int       `json:"zone_id"`
        FreeSlots   int       `json:"free_slots"`
    }
    var list []dto
    for rows.Next() {
        var (
            id, max, trainerID, zoneID, free int
            title, desc, level, tname, zname string
            start, end time.Time
        )
        if err := rows.Scan(&id, &title, &desc, &max, &start, &end, &level, &tname, &trainerID, &zname, &zoneID, &free); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        list = append(list, dto{ID: id, Title: title, Description: desc, Max: max, Start: start, End: end, Level: level, TrainerName: tname, TrainerID: trainerID, ZoneName: zname, ZoneID: zoneID, FreeSlots: free})
    }
    if err := rows.Err(); err != nil { return jsonError(c, 500, "Ошибка курсора", err) }
    return jsonOK(c, fiber.Map{"groups": list})
}

// APIv1ListPersonalTrainings — JSON-список персональных тренировок с фильтрами
func APIv1ListPersonalTrainings(c *fiber.Ctx) error {
    db := database.GetDB()
    q := strings.TrimSpace(c.Query("q"))
    qTrainer := strings.TrimSpace(c.Query("trainer_id"))
    qStatus := strings.TrimSpace(c.Query("status"))
    qFrom := strings.TrimSpace(c.Query("from"))
    qTo := strings.TrimSpace(c.Query("to"))
    onlyUpcoming := c.Query("upcoming") == "1"
    recent30 := c.Query("recent") == "1"

    where := []string{}
    args := []any{}
    nextPH := func(n int) []string {
        start := len(args) + 1
        ph := make([]string, n)
        for i := 0; i < n; i++ { ph[i] = "$" + strconv.Itoa(start+i) }
        return ph
    }

    query := `
        SELECT 
            v."id_персональной_тренировки",
            v."Время_начала",
            v."Время_окончания",
            v."Статус",
            COALESCE(v."Стоимость",0)     AS price,
            v."id_абонемента",
            v."id_клиента",
            v.client_fio,
            v."id_тренера",
            v.trainer_fio
        FROM vw_personal_training_enriched v`

    if q != "" {
        like := "%" + q + "%"
        ph := nextPH(2)
        where = append(where, `(
            v.client_fio  ILIKE `+ph[0]+` OR
            v.trainer_fio ILIKE `+ph[1]+`
        )`)
        args = append(args, like, like)
    }
    if qTrainer != "" {
        ph := nextPH(1)
        where = append(where, `v."id_тренера" = `+ph[0]+`::int`)
        args = append(args, qTrainer)
    }
    if qStatus != "" {
        ph := nextPH(1)
        where = append(where, `v."Статус" = `+ph[0])
        args = append(args, qStatus)
    }
    if qFrom != "" {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" >= `+ph[0]+`::timestamp`)
        args = append(args, qFrom)
    }
    if qTo != "" {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" <= `+ph[0]+`::timestamp`)
        args = append(args, qTo)
    }
    if onlyUpcoming {
        ph := nextPH(1)
        where = append(where, `v."Время_начала" >= `+ph[0]+`::timestamp`)
        args = append(args, time.Now())
    }
    if recent30 {
        where = append(where, `v."Время_начала" >= NOW() - INTERVAL '30 days'`)
    }
    if len(where) > 0 {
        query += " WHERE " + strings.Join(where, " AND ")
    }
    query += ` ORDER BY v."Время_начала" DESC, v."id_персональной_тренировки" DESC`

    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil { return jsonError(c, 500, "Ошибка загрузки персональных тренировок", err) }
    defer rows.Close()

    type dto struct {
        ID            int       `json:"id"`
        Start         time.Time `json:"start"`
        End           time.Time `json:"end"`
        Status        string    `json:"status"`
        Price         float64   `json:"price"`
        SubscriptionID int      `json:"subscription_id"`
        ClientID      int       `json:"client_id"`
        ClientFIO     string    `json:"client_fio"`
        TrainerID     int       `json:"trainer_id"`
        TrainerFIO    string    `json:"trainer_fio"`
    }
    var list []dto
    for rows.Next() {
        var (
            id, subID, clientID, trainerID int
            start, end time.Time
            status string
            price float64
            clientFIO, trainerFIO string
        )
        if err := rows.Scan(&id, &start, &end, &status, &price, &subID, &clientID, &clientFIO, &trainerID, &trainerFIO); err != nil {
            return jsonError(c, 500, "Ошибка чтения строки", err)
        }
        list = append(list, dto{ID: id, Start: start, End: end, Status: status, Price: price, SubscriptionID: subID, ClientID: clientID, ClientFIO: clientFIO, TrainerID: trainerID, TrainerFIO: trainerFIO})
    }
    if err := rows.Err(); err != nil { return jsonError(c, 500, "Ошибка курсора", err) }
    return jsonOK(c, fiber.Map{"personal": list})
}


// ====== CRUD: Групповые ======

func GetGroupTrainingByID(c *fiber.Ctx) error {
    id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
    }
	db := database.GetDB()
	var (
		title, desc, level string
		max, trainerID, zoneID int
		start, end time.Time
	)
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        SELECT "Название", COALESCE("Описание",''), COALESCE("Уровень_сложности",''), 
               COALESCE("Максимум_участников",0),
               "Время_начала","Время_окончания",
               "id_тренера","id_зоны"
        FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1
    `, id).Scan(&title, &desc, &level, &max, &start, &end, &trainerID, &zoneID)
    if err == sql.ErrNoRows {
        return jsonError(c, 404, "Не найдено", nil)
    }
    if err != nil {
        return jsonError(c, 500, "Ошибка БД", err)
    }
    return jsonOK(c, fiber.Map{"item": fiber.Map{
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.Title == "" || f.Date == "" || f.Start == "" || f.End == "" || f.Trainer <= 0 || f.Zone <= 0 || f.Max <= 0 {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }
	switch f.Level {
	case "", "Начальный", "Средний", "Продвинутый":
	default:
        return jsonError(c, 400, "Неверный уровень сложности", nil)
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
    if err1 != nil || err2 != nil || !end.After(start) {
        return jsonError(c, 400, "Некорректное время начала/окончания", nil)
    }
	db := database.GetDB()
	var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Групповая_тренировка"
        ("id_тренера","id_зоны","Название","Описание","Максимум_участников","Время_начала","Время_окончания","Уровень_сложности")
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING "id_групповой_тренировки"
    `, f.Trainer, f.Zone, f.Title, nullIfEmpty(f.Desc), f.Max, start, end, nullIfEmpty(f.Level)).Scan(&id)
    if err != nil {
        log.Printf("create group err: %v", err)
        return jsonError(c, 500, "Ошибка сохранения", err)
    }
    return jsonOK(c, fiber.Map{"id": id, "message": "Групповая тренировка создана"})
}

func UpdateGroupTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
	if f.Title == "" || f.Date == "" || f.Start == "" || f.End == "" || f.Trainer <= 0 || f.Zone <= 0 || f.Max <= 0 {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
	}
	switch f.Level {
	case "", "Начальный", "Средний", "Продвинутый":
	default:
        return jsonError(c, 400, "Неверный уровень сложности", nil)
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
    if err1 != nil || err2 != nil || !end.After(start) {
        return jsonError(c, 400, "Некорректное время начала/окончания", nil)
    }

	db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Групповая_тренировка"
        SET "id_тренера"=$2,"id_зоны"=$3,"Название"=$4,"Описание"=$5,"Максимум_участников"=$6,
            "Время_начала"=$7,"Время_окончания"=$8,"Уровень_сложности"=$9
        WHERE "id_групповой_тренировки"=$1
    `, id, f.Trainer, f.Zone, f.Title, nullIfEmpty(f.Desc), f.Max, start, end, nullIfEmpty(f.Level))
    if err != nil {
        return jsonError(c, 500, "Ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Обновлено"})
}

func DeleteGroupTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
    }
	db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1`, id)
    if err != nil {
        // связанные записи в "Запись_на_групповую_тренировку" могут мешать, но там ON DELETE CASCADE — должно удалиться
        return jsonError(c, 500, "Ошибка удаления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Удалено"})
}

// ====== CRUD: Персональные ======

func GetPersonalTrainingByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
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
        return jsonError(c, 404, "Не найдено", nil)
    }
    if err != nil {
        return jsonError(c, 500, "Ошибка БД", err)
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.Subscription <= 0 || f.Trainer <= 0 || f.Date == "" || f.Start == "" || f.End == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }
	switch f.Status {
	case "", "Запланирована", "Завершена", "Отменена":
    default:
        return jsonError(c, 400, "Неверный статус", nil)
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
    if err1 != nil || err2 != nil || !end.After(start) {
        return jsonError(c, 400, "Некорректное время начала/окончания", nil)
    }
	var price *float64
	if f.Price != "" {
		if p, err := strconv.ParseFloat(f.Price, 64); err == nil {
			price = &p
		} else {
            return jsonError(c, 400, "Неверная стоимость", err)
        }
    }
    db := database.GetDB()
    var id int
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Персональная_тренировка"
        ("id_абонемента","id_тренера","Время_начала","Время_окончания","Статус","Стоимость")
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING "id_персональной_тренировки"
    `, f.Subscription, f.Trainer, start, end, coalesceStr(f.Status, "Запланирована"), nullablePrice(price)).Scan(&id)
    if err != nil {
        log.Printf("create personal err: %v", err)
        return jsonError(c, 500, "Ошибка сохранения", err)
    }
    return jsonOK(c, fiber.Map{"id": id, "message": "Персональная тренировка создана"})
}

func UpdatePersonalTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.Subscription <= 0 || f.Trainer <= 0 || f.Date == "" || f.Start == "" || f.End == "" {
        return jsonError(c, 400, "Заполните обязательные поля", nil)
    }
	switch f.Status {
	case "Запланирована", "Завершена", "Отменена":
    default:
        return jsonError(c, 400, "Неверный статус", nil)
	}
	start, err1 := time.Parse("2006-01-02 15:04", f.Date+" "+f.Start)
	end,   err2 := time.Parse("2006-01-02 15:04", f.Date+" "+f.End)
    if err1 != nil || err2 != nil || !end.After(start) {
        return jsonError(c, 400, "Некорректное время начала/окончания", nil)
    }
	var price *float64
	if f.Price != "" {
		if p, err := strconv.ParseFloat(f.Price, 64); err == nil {
			price = &p
		} else {
            return jsonError(c, 400, "Неверная стоимость", err)
        }
    }
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `
        UPDATE "Персональная_тренировка"
        SET "id_абонемента"=$2,"id_тренера"=$3,"Время_начала"=$4,"Время_окончания"=$5,"Статус"=$6,"Стоимость"=$7
        WHERE "id_персональной_тренировки"=$1
    `, id, f.Subscription, f.Trainer, start, end, f.Status, nullablePrice(price))
    if err != nil {
        return jsonError(c, 500, "Ошибка обновления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Обновлено"})
}

func DeletePersonalTraining(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id", nil)
    }
    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    res, err := db.ExecContext(ctx, `DELETE FROM "Персональная_тренировка" WHERE "id_персональной_тренировки"=$1`, id)
    if err != nil {
        return jsonError(c, 500, "Ошибка удаления", err)
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return jsonError(c, 404, "Не найдено", nil)
    }
    return jsonOK(c, fiber.Map{"message": "Удалено"})
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
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if f.GroupID <= 0 || f.SubID <= 0 {
        return jsonError(c, 400, "Выберите тренировку и абонемент", nil)
    }
	switch f.Status {
	case "", "Записан", "Посетил", "Отменил":
    default:
        return jsonError(c, 400, "Неверный статус записи", nil)
	}

    db := database.GetDB()
    // проверим, что групповая существует
    var exists int
    ctx, cancel := withDBTimeout()
    defer cancel()
    if err := db.QueryRowContext(ctx, `SELECT 1 FROM "Групповая_тренировка" WHERE "id_групповой_тренировки"=$1`, f.GroupID).Scan(&exists); err != nil {
        return jsonError(c, 400, "Групповая тренировка не найдена", err)
    }
    // и абонемент существует
    if err := db.QueryRowContext(ctx, `SELECT 1 FROM "Абонемент" WHERE "id_абонемента"=$1`, f.SubID).Scan(&exists); err != nil {
        return jsonError(c, 400, "Абонемент не найден", err)
    }

    var id int
    err := db.QueryRowContext(ctx, `
        INSERT INTO "Запись_на_групповую_тренировку"
        ("id_групповой_тренировки","id_абонемента","Статус")
        VALUES ($1,$2,$3)
        RETURNING "id_записи"
    `, f.GroupID, f.SubID, coalesceStr(f.Status, "Записан")).Scan(&id)
    if err != nil {
        log.Printf("enrollment err: %v", err)
        return jsonError(c, 500, "Не удалось создать запись (возможно, дубликат)", err)
    }
    return jsonOK(c, fiber.Map{"id": id, "message": "Запись создана"})
}

func ListGroupEnrollments(c *fiber.Ctx) error {
    id, _ := strconv.Atoi(c.Params("id"))
    if id <= 0 {
        return jsonError(c, 400, "Некорректный id тренировки", nil)
    }

	db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, `
        SELECT 
            e."id_записи",
            e."Статус",
            s."id_абонемента",
            c."id_клиента",
            c."ФИО"
        FROM "Запись_на_групповую_тренировку" e
        JOIN "Абонемент" s ON s."id_абонемента" = e."id_абонемента"
        JOIN "Клиент"    c ON c."id_клиента"    = s."id_клиента"
        WHERE e."id_групповой_тренировки" = $1
        ORDER BY e."id_записи" DESC
    `, id)
    if err != nil {
        return jsonError(c, 500, "Ошибка загрузки записей", err)
    }
	defer rows.Close()

	type item struct {
		ID            int    `json:"id"`
		Status        string `json:"status"`
		Subscription  int    `json:"subscription_id"`
		ClientID      int    `json:"client_id"`
		ClientFIO     string `json:"client_fio"`
	}
	var list []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Status, &it.Subscription, &it.ClientID, &it.ClientFIO); err == nil {
			list = append(list, it)
		}
	}

    return jsonOK(c, fiber.Map{"enrollments": list})
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

