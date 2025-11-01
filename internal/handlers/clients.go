package handlers

import (
	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/models"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func GetClients(c *fiber.Ctx) error {
    db := database.GetDB()
    
    rows, err := db.Query(`
        SELECT 
            "id_клиента", 
            "ФИО", 
            "Номер_телефона", 
            "Дата_рождения", 
            "Дата_регистрации", 
            "Медицинские_данные"
        FROM "Клиент" 
        ORDER BY "id_клиента"
    `)
    if err != nil {
        return c.Status(500).SendString("Ошибка получения клиентов: " + err.Error())
    }
    defer rows.Close()
    
    var clients []models.Client
    for rows.Next() {
        var client models.Client
        err := rows.Scan(
            &client.ID,
            &client.FIO, 
            &client.Phone,
            &client.BirthDate,
            &client.RegisterDate,
            &client.MedicalData,
        )
        if err != nil {
            return c.Status(500).SendString("Ошибка сканирования клиента: " + err.Error())
        }
        clients = append(clients, client)
    }
    
    return c.Render("clients", fiber.Map{
        "Title":   "Клиенты",
        "Clients": clients,
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
    log.Printf("📝 Данные для сохранения: FIO=%s, Phone=%s, MedicalData='%s'", 
        form.FIO, form.Phone, form.MedicalData)
    
    var clientID int
    // Если MedicalData пустая строка, она сохранится как NULL
    err = db.QueryRow(`
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
    
    log.Printf("✅ Клиент создан! ID: %d, Мед.данные: '%s'", clientID, form.MedicalData)
    
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
    err := db.QueryRow(`
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
    
    result, err := db.Exec(`
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
    err = db.QueryRow(`SELECT COUNT(*) FROM Абонемент WHERE id_клиента = $1`, clientID).Scan(&subscriptionCount)
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

    result, err := db.Exec(`DELETE FROM Клиент WHERE id_клиента = $1`,clientID)
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