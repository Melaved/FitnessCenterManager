package handlers

import (
    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"
    "html/template"
    "log"
    "strconv"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
)

func GetClients(c *fiber.Ctx) error {
    db := database.GetDB()

    // === параметры фильтра ===
    q := strings.TrimSpace(c.Query("q"))         // строка поиска
    onlyWithMed := c.Query("medical") == "1"     // чекбокс «только с мед. данными»
    recent30 := c.Query("recent") == "1"         // «за 30 дней»
    // пагинация
    page, _ := strconv.Atoi(c.Query("page"))
    size, _ := strconv.Atoi(c.Query("size"))
    if page <= 0 { page = 1 }
    if size <= 0 || size > 100 { size = 20 }

    // === динамический WHERE ===
    where := []string{}
    args := []any{}
    paramCount := 0

	nextPH := func() string {
		paramCount++
		return "$" + strconv.Itoa(paramCount)
	}

	if q != "" {
		like := "%" + q + "%"
		where = append(where, `(
			v."ФИО" ILIKE `+nextPH()+` OR
			v."Номер_телефона" ILIKE `+nextPH()+` OR
			CAST(v."id_клиента" AS TEXT) ILIKE `+nextPH()+`
		)`)
		args = append(args, like, like, like)
	}
	if onlyWithMed {
		// есть непустые медданные
		where = append(where, `COALESCE(NULLIF(c."Медицинские_данные", ''), NULL) IS NOT NULL`)
	}
	if recent30 {
		// зарегистрирован за последние 30 дней
		where = append(where, `c."Дата_регистрации" >= NOW()::date - INTERVAL '30 days'`)
	}

    // === базовый SELECT ===
    baseSelect := `
        SELECT
            v."id_клиента",
            v."ФИО",
            v."Номер_телефона",
            c."Дата_рождения",
            c."Дата_регистрации",
            c."Медицинские_данные",
            v.age,
            COALESCE(v.subs_total, 0) AS subscriptions_count,
            CASE WHEN v.subs_active > 0 THEN 'Активен' ELSE 'Неактивен' END AS active_status
        FROM public.view_client_enriched v
        JOIN public."Клиент" c USING ("id_клиента")
    `
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = " WHERE " + strings.Join(where, " AND ")
    }

    // === общее количество для пагинации ===
    countSQL := "SELECT COUNT(*) FROM (" + baseSelect + whereSQL + ") t"
    ctxCount, cancelCount := withDBTimeout()
    var total int
    if err := db.QueryRowContext(ctxCount, countSQL, args...).Scan(&total); err != nil {
        cancelCount()
        return c.Status(500).SendString("Ошибка подсчёта записей: " + err.Error())
    }
    cancelCount()

    // === финальный запрос с LIMIT/OFFSET ===
    query := baseSelect + whereSQL + ` ORDER BY v."ФИО" LIMIT $` + strconv.Itoa(paramCount+1) + ` OFFSET $` + strconv.Itoa(paramCount+2)
    args = append(args, size, (page-1)*size)

    ctx, cancel := withDBTimeout()
    defer cancel()

    rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		return c.Status(500).SendString("Ошибка получения клиентов: " + err.Error())
	}
	defer rows.Close()

	var clients []models.ClientEnriched
	for rows.Next() {
		var cl models.ClientEnriched
		if err := rows.Scan(
			&cl.ID,
			&cl.FIO,
			&cl.Phone,
			&cl.BirthDate,
			&cl.RegisterDate,
			&cl.MedicalData,
			&cl.Age,
			&cl.SubscriptionsCnt,
			&cl.ActiveStatus,
		); err != nil {
			log.Printf("Scan error: %v", err)
			return c.Status(500).SendString("Ошибка сканирования клиента: " + err.Error())
		}
		clients = append(clients, cl)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
		return c.Status(500).SendString("Ошибка при обработке результатов: " + err.Error())
	}

    return c.Render("clients", fiber.Map{
        "Title":   "Клиенты",
        "Clients": clients,
        "Filter": fiber.Map{
            "q":       q,
            "medical": onlyWithMed,
            "recent":  recent30,
        },
        "Pagination": fiber.Map{
            "page": page,
            "size": size,
            "total": total,
            "has_prev": page > 1,
            "has_next": page*size < total,
            "prev": page-1,
            "next": page+1,
        },
        "ExtraScripts": template.HTML(`<script src="/static/js/clients.js"></script>`),
    })
}

// APIv1ListClients — JSON-список клиентов с фильтрами/пагинацией
func APIv1ListClients(c *fiber.Ctx) error {
    db := database.GetDB()

    q := strings.TrimSpace(c.Query("q"))
    onlyWithMed := c.Query("medical") == "1"
    recent30 := c.Query("recent") == "1"
    page, _ := strconv.Atoi(c.Query("page"))
    size, _ := strconv.Atoi(c.Query("size"))
    if page <= 0 { page = 1 }
    if size <= 0 || size > 100 { size = 20 }

    where := []string{}
    args := []any{}
    paramCount := 0
    nextPH := func() string {
        paramCount++
        return "$" + strconv.Itoa(paramCount)
    }
    if q != "" {
        like := "%" + q + "%"
        where = append(where, `(
            v."ФИО" ILIKE `+nextPH()+` OR
            v."Номер_телефона" ILIKE `+nextPH()+` OR
            CAST(v."id_клиента" AS TEXT) ILIKE `+nextPH()+`
        )`)
        args = append(args, like, like, like)
    }
    if onlyWithMed {
        where = append(where, `COALESCE(NULLIF(c."Медицинские_данные", ''), NULL) IS NOT NULL`)
    }
    if recent30 {
        where = append(where, `c."Дата_регистрации" >= NOW()::date - INTERVAL '30 days'`)
    }

    baseSelect := `
        SELECT
            v."id_клиента",
            v."ФИО",
            v."Номер_телефона",
            c."Дата_рождения",
            c."Дата_регистрации",
            c."Медицинские_данные",
            v.age,
            COALESCE(v.subs_total, 0) AS subscriptions_count,
            CASE WHEN v.subs_active > 0 THEN 'Активен' ELSE 'Неактивен' END AS active_status
        FROM public.view_client_enriched v
        JOIN public."Клиент" c USING ("id_клиента")
    `
    whereSQL := ""
    if len(where) > 0 {
        whereSQL = " WHERE " + strings.Join(where, " AND ")
    }

    // count
    countSQL := "SELECT COUNT(*) FROM (" + baseSelect + whereSQL + ") t"
    ctxCount, cancelCount := withDBTimeout()
    var total int
    if err := db.QueryRowContext(ctxCount, countSQL, args...).Scan(&total); err != nil {
        cancelCount()
        return jsonError(c, 500, "Ошибка подсчёта записей", err)
    }
    cancelCount()

    // data
    query := baseSelect + whereSQL + ` ORDER BY v."ФИО" LIMIT $` + strconv.Itoa(paramCount+1) + ` OFFSET $` + strconv.Itoa(paramCount+2)
    args = append(args, size, (page-1)*size)

    ctx, cancel := withDBTimeout()
    defer cancel()
    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return jsonError(c, 500, "Ошибка получения клиентов", err)
    }
    defer rows.Close()

    type clientDTO struct {
        ID                  int    `json:"id"`
        FIO                 string `json:"fio"`
        Phone               string `json:"phone"`
        BirthDate           string `json:"birth_date"`
        RegisterDate        string `json:"register_date"`
        MedicalData         string `json:"medical_data"`
        Age                 int    `json:"age"`
        SubscriptionsCount  int    `json:"subscriptions_count"`
        ActiveStatus        string `json:"active_status"`
    }
    var list []clientDTO
    for rows.Next() {
        var cl models.ClientEnriched
        if err := rows.Scan(
            &cl.ID,
            &cl.FIO,
            &cl.Phone,
            &cl.BirthDate,
            &cl.RegisterDate,
            &cl.MedicalData,
            &cl.Age,
            &cl.SubscriptionsCnt,
            &cl.ActiveStatus,
        ); err != nil {
            return jsonError(c, 500, "Ошибка сканирования клиента", err)
        }
        list = append(list, clientDTO{
            ID:                 cl.ID,
            FIO:                cl.FIO,
            Phone:              cl.Phone,
            BirthDate:          cl.BirthDate.Format("2006-01-02"),
            RegisterDate:       cl.RegisterDate.Format("2006-01-02"),
            MedicalData:        cl.MedicalData.String,
            Age:                cl.Age,
            SubscriptionsCount: cl.SubscriptionsCnt,
            ActiveStatus:       cl.ActiveStatus,
        })
    }
    if err := rows.Err(); err != nil {
        return jsonError(c, 500, "Ошибка при обработке результатов", err)
    }

    return jsonOK(c, fiber.Map{
        "clients": list,
        "pagination": fiber.Map{
            "page": page,
            "size": size,
            "total": total,
            "has_prev": page > 1,
            "has_next": page*size < total,
            "prev": page-1,
            "next": page+1,
        },
        "filter": fiber.Map{
            "q": q,
            "medical": onlyWithMed,
            "recent": recent30,
        },
    })
}


// CreateClient создает нового клиента
func CreateClient(c *fiber.Ctx) error {
    log.Println("🎯 Создание нового клиента...")
    
    type ClientForm struct {
        FIO         string `form:"fio"`
        Phone       string `form:"phone"`
        BirthDate   string `form:"birth_date"`
        MedicalData string `form:"medical_data"`
    }
    
    var form ClientForm
    if err := c.BodyParser(&form); err != nil {
        log.Printf("❌ Ошибка парсинга формы: %v", err)
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    
    // Валидация данных
    if form.FIO == "" || form.Phone == "" || form.BirthDate == "" {
        return jsonError(c, 400, "Все обязательные поля должны быть заполнены", nil)
    }
    
    // Парсим дату рождения
    birthDate, err := time.Parse("2006-01-02", form.BirthDate)
    if err != nil {
        return jsonError(c, 400, "Неверный формат даты", err)
    }
    
    // Проверка возраста
    age := time.Since(birthDate).Hours() / 24 / 365
    if age < 16 {
        return jsonError(c, 400, "Клиент должен быть старше 16 лет", nil)
    }
    
    db := database.GetDB()
    
    // ОТЛАДОЧНАЯ ИНФОРМАЦИЯ
    // redact sensitive medical data in logs
    log.Printf("📝 Данные для сохранения: FIO=%s, Phone=%s", 
        form.FIO, form.Phone)
    
    var clientID int
    // Если MedicalData пустая строка, она сохранится как NULL
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `
        INSERT INTO "Клиент" ("ФИО", "Номер_телефона", "Дата_рождения", "Медицинские_данные")
        VALUES ($1, $2, $3, $4)
        RETURNING "id_клиента"
    `, form.FIO, form.Phone, birthDate, form.MedicalData).Scan(&clientID)
    
    if err != nil {
        log.Printf("❌ Ошибка сохранения клиента: %v", err)
        return jsonError(c, 500, "Ошибка сохранения в базу данных", err)
    }
    
    log.Printf("✅ Клиент создан! ID: %d", clientID)
    
    return c.JSON(fiber.Map{
        "success": true,
        "message": "Клиент успешно создан",
        "client_id": clientID,
    })
}

// GetClientByID возвращает клиента по ID для редактирования
func GetClientByID(c *fiber.Ctx) error {
    id := c.Params("id")
    
    db := database.GetDB()
    
    var client models.Client
    ctx, cancel := withDBTimeout()
    defer cancel()
    err := db.QueryRowContext(ctx, `
        SELECT 
            "id_клиента", 
            "ФИО", 
            "Номер_телефона", 
            "Дата_рождения", 
            "Дата_регистрации", 
            "Медицинские_данные"
        FROM "Клиент" 
        WHERE "id_клиента" = $1
    `, id).Scan(
        &client.ID,
        &client.FIO, 
        &client.Phone,
        &client.BirthDate,
        &client.RegisterDate,
        &client.MedicalData,
    )

    if err != nil {
        return jsonError(c, 404, "Клиент не найден", err)
    }
    
    return jsonOK(c, fiber.Map{
        "client": fiber.Map{
            "id": client.ID,
            "fio": client.FIO,
            "phone": client.Phone,
            "birth_date": client.BirthDate.Format("2006-01-02"),
            "medical_data": client.MedicalData.String,
        },
    })
}

// UpdateClient обновляет данные клиента
func UpdateClient(c *fiber.Ctx) error {
    id := c.Params("id")
    
    type ClientForm struct {
        FIO         string `form:"fio"`
        Phone       string `form:"phone"`
        BirthDate   string `form:"birth_date"`
        MedicalData string `form:"medical_data"`
    }
    
    var form ClientForm
    if err := c.BodyParser(&form); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    
    if form.FIO == "" || form.Phone == "" || form.BirthDate == "" {
        return jsonError(c, 400, "Все обязательные поля должны быть заполнены", nil)
    }
    
    birthDate, err := time.Parse("2006-01-02", form.BirthDate)
    if err != nil {
        return jsonError(c, 400, "Неверный формат даты", err)
    }
    
    db := database.GetDB()
    
    ctx, cancel := withDBTimeout()
    defer cancel()
    result, err := db.ExecContext(ctx, `
        UPDATE "Клиент" 
        SET "ФИО" = $1, "Номер_телефона" = $2, "Дата_рождения" = $3, "Медицинские_данные" = $4
        WHERE "id_клиента" = $5
    `, form.FIO, form.Phone, birthDate, form.MedicalData, id)
    
    if err != nil {
        return jsonError(c, 500, "Ошибка обновления", err)
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return jsonError(c, 404, "Клиент не найден", nil)
    }
    
    return c.JSON(fiber.Map{
        "success": true,
        "message": "Клиент успешно обновлен",
    })
}

func DeleteClient(c *fiber.Ctx) error{
    id := c.Params(("id"))

    clientID, err := strconv.Atoi(id)
    if err != nil{
        return jsonError(c, 400, "Неверный Id клиента", err)
    }

    db := database.GetDB()
    var subscriptionCount int

    //Проверка абонементов
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM Абонемент WHERE id_клиента = $1`, clientID).Scan(&subscriptionCount)
    if err != nil {
        return jsonError(c, 500, "Ошибка проверки данных клиента", err)
    }
    if subscriptionCount > 0{
        return jsonError(c, 400, "Невозможно удалить клиента: есть активные абонементы", nil)
    }

    ctx, cancel = withDBTimeout()
    defer cancel()
    result, err := db.ExecContext(ctx, `DELETE FROM Клиент WHERE id_клиента = $1`,clientID)
    if err != nil{
        return jsonError(c, 500, "Ошибка удаления клиента", err)
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0{
        return jsonError(c, 404, "Клиент не найден", nil)
    }

    return jsonOK(c, fiber.Map{"message": "Клиент успешно удален"})
}

// APIv1CreateClient — создание клиента с 201/Location
func APIv1CreateClient(c *fiber.Ctx) error {
    type ClientForm struct {
        FIO         string `form:"fio"`
        Phone       string `form:"phone"`
        BirthDate   string `form:"birth_date"`
        MedicalData string `form:"medical_data"`
    }
    var form ClientForm
    if err := c.BodyParser(&form); err != nil {
        return jsonError(c, 400, "Неверные данные формы", err)
    }
    if form.FIO == "" || form.Phone == "" || form.BirthDate == "" {
        return jsonError(c, 400, "Все обязательные поля должны быть заполнены", nil)
    }
    birthDate, err := time.Parse("2006-01-02", form.BirthDate)
    if err != nil {
        return jsonError(c, 400, "Неверный формат даты", err)
    }
    // возраст >= 16
    age := time.Since(birthDate).Hours() / 24 / 365
    if age < 16 {
        return jsonError(c, 400, "Клиент должен быть старше 16 лет", nil)
    }

    db := database.GetDB()
    ctx, cancel := withDBTimeout()
    defer cancel()
    var clientID int
    if err := db.QueryRowContext(ctx, `
        INSERT INTO "Клиент" ("ФИО", "Номер_телефона", "Дата_рождения", "Медицинские_данные")
        VALUES ($1,$2,$3,$4)
        RETURNING "id_клиента"
    `, form.FIO, form.Phone, birthDate, form.MedicalData).Scan(&clientID); err != nil {
        return jsonError(c, 500, "Ошибка сохранения в базу данных", err)
    }

    c.Set("Location", "/api/v1/clients/"+strconv.Itoa(clientID))
    return c.Status(fiber.StatusCreated).JSON(fiber.Map{
        "success": true,
        "message": "Клиент успешно создан",
        "client_id": clientID,
    })
}

func GetClientsForSelect(c *fiber.Ctx) error {
	db := database.GetDB()
    rows, err := db.Query(`SELECT "id_клиента","ФИО" FROM "Клиент" ORDER BY "id_клиента"`)
    if err != nil {
        return jsonError(c, 500, "Ошибка чтения клиентов", err)
    }
	defer rows.Close()

	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var list []item
	for rows.Next() {
		var v item
		if err := rows.Scan(&v.ID, &v.Name); err == nil {
			list = append(list, v)
		}
	}
    return jsonOK(c, fiber.Map{"clients": list})
}
