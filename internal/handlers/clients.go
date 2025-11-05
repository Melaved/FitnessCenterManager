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

	q := strings.TrimSpace(c.Query("q"))     // строка поиска
	onlyWithMed := c.Query("medical") == "1" // чекбокс «только с мед. данными»
	recent30 := c.Query("recent") == "1"     // чекбокс «за 30 дней»

	// динамический WHERE
	where := []string{}
	args := []any{}

	// отдельный счётчик плейсхолдеров ($1, $2, ...)
	n := 1
	nextPH := func() string {
		s := "$" + strconv.Itoa(n)
		n++
		return s
	}

	if q != "" {
		like := "%" + q + "%"
		// получаем ТРИ РАЗНЫХ плейсхолдера ($1, $2, $3)
		p1 := nextPH()
		p2 := nextPH()
		p3 := nextPH()
		where = append(where, `(
			"ФИО" ILIKE `+p1+` OR
			"Номер_телефона" ILIKE `+p2+` OR
			CAST("id_клиента" AS TEXT) ILIKE `+p3+`
		)`)
		args = append(args, like, like, like)
	}

	if onlyWithMed {
		where = append(where, `COALESCE(NULLIF("Медицинские_данные", ''), NULL) IS NOT NULL`)
	}

	if recent30 {
		where = append(where, `"Дата_регистрации" >= NOW() - INTERVAL '30 days'`)
	}

	query := `
		SELECT 
			"id_клиента", 
			"ФИО", 
			"Номер_телефона", 
			"Дата_рождения", 
			"Дата_регистрации", 
			"Медицинские_данные"
		FROM "Клиент"
	`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += ` ORDER BY "id_клиента"`

	ctx, cancel := withDBTimeout()
	defer cancel()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return c.Status(500).SendString("Ошибка получения клиентов: " + err.Error())
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(
			&client.ID,
			&client.FIO,
			&client.Phone,
			&client.BirthDate,
			&client.RegisterDate,
			&client.MedicalData,
		); err != nil {
			return c.Status(500).SendString("Ошибка сканирования клиента: " + err.Error())
		}
		clients = append(clients, client)
	}

	return c.Render("clients", fiber.Map{
		"Title":   "Клиенты",
		"Clients": clients,
		"Filter": fiber.Map{
			"q":       q,
			"medical": onlyWithMed,
			"recent":  recent30,
		},
		"ExtraScripts": template.HTML(`<script src="/static/js/clients.js?v=2"></script>`),
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
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Неверные данные формы",
        })
    }
    
    // Валидация данных
    if form.FIO == "" || form.Phone == "" || form.BirthDate == "" {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Все обязательные поля должны быть заполнены",
        })
    }
    
    // Парсим дату рождения
    birthDate, err := time.Parse("2006-01-02", form.BirthDate)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Неверный формат даты",
        })
    }
    
    // Проверка возраста
    age := time.Since(birthDate).Hours() / 24 / 365
    if age < 16 {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Клиент должен быть старше 16 лет",
        })
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
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка сохранения в базу данных: " + err.Error(),
        })
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
        return c.Status(404).JSON(fiber.Map{
            "success": false,
            "error":   "Клиент не найден",
        })
    }
    
    return c.JSON(fiber.Map{
        "success": true,
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
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Неверные данные формы",
        })
    }
    
    if form.FIO == "" || form.Phone == "" || form.BirthDate == "" {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Все обязательные поля должны быть заполнены",
        })
    }
    
    birthDate, err := time.Parse("2006-01-02", form.BirthDate)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Неверный формат даты",
        })
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
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка обновления: " + err.Error(),
        })
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return c.Status(404).JSON(fiber.Map{
            "success": false,
            "error":   "Клиент не найден",
        })
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
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error": "Неверный Id клиента",
        })
    }

    db := database.GetDB()
    var subscriptionCount int

    //Проверка абонементов
    ctx, cancel := withDBTimeout()
    defer cancel()
    err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM Абонемент WHERE id_клиента = $1`, clientID).Scan(&subscriptionCount)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error": "Ошибка проверки данных клиента",
        })
    }
    if subscriptionCount > 0{
        return c.Status(400).JSON(fiber.Map{
            "success":false,
            "error": "Невозможно удалить клиента у него есть активные абонементы: Сначала удалите абонементы",
        })
    }

    ctx, cancel = withDBTimeout()
    defer cancel()
    result, err := db.ExecContext(ctx, `DELETE FROM Клиент WHERE id_клиента = $1`,clientID)
    if err != nil{
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error": "Ошибка удаления клиента" + err.Error(),
        })
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0{
        return c.Status(404).JSON(fiber.Map{
            "success": false,
            "error": "Клиент не найден",
        })
    }

    return c.JSON(fiber.Map{
        "success": true,
        "message": "Клиент успешно удален",
    })
}

func GetClientsForSelect(c *fiber.Ctx) error {
	db := database.GetDB()
	rows, err := db.Query(`SELECT "id_клиента","ФИО" FROM "Клиент" ORDER BY "id_клиента"`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": "Ошибка чтения клиентов"})
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
	return c.JSON(fiber.Map{"success": true, "clients": list})
}