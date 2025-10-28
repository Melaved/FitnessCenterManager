package handlers

import (
	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/models"
	"log"

	"github.com/gofiber/fiber/v2"
)

// GetSubscriptions отображает страницу абонементов
func GetSubscriptions(c *fiber.Ctx) error {
    db := database.GetDB()
    
    log.Println("🔍 Получение абонементов из БД...")
    
    // Используем СМЕШАННЫЙ регистр: id_ с маленькой, остальное с большой
    rows, err := db.Query(`
        SELECT 
            a."id_абонемента",
            a."id_клиента", 
            a."id_тарифа",
            a."Дата_начала",
            a."Дата_окончания", 
            a."Статус",
            a."Цена",
            k."ФИО" as "ФИО_клиента",
            t."Название_тарифа" as "Название_тарифа"
        FROM "Абонемент" a
        JOIN "Клиент" k ON a."id_клиента" = k."id_клиента"
        JOIN "Тариф" t ON a."id_тарифа" = t."id_тарифа"
        ORDER BY a."id_абонемента"
    `)
    
    if err != nil {
        log.Printf("❌ Ошибка получения абонементов: %v", err)
        return c.Render("subscriptions", fiber.Map{
            "Title":         "Абонементы",
            "Subscriptions": []models.Subscription{},
        })
    }
    defer rows.Close()
    
    var subscriptions []models.Subscription
    for rows.Next() {
        var sub models.Subscription
        err := rows.Scan(
            &sub.ID,
            &sub.ClientID,
            &sub.TariffID, 
            &sub.StartDate,
            &sub.EndDate,
            &sub.Status,
            &sub.Price,
            &sub.ClientName,
            &sub.TariffName,
        )
        if err != nil {
            log.Printf("❌ Ошибка сканирования абонемента: %v", err)
            continue
        }
        subscriptions = append(subscriptions, sub)
    }
    
    log.Printf("✅ Загружено %d абонементов", len(subscriptions))
    
    return c.Render("subscriptions", fiber.Map{
        "Title":         "Абонементы", 
        "Subscriptions": subscriptions,
    })
}

// GetClientsForSelect возвращает список клиентов для ComboBox
func GetClientsForSelect(c *fiber.Ctx) error {
    db := database.GetDB()
    
    log.Println("🔍 GetClientsForSelect: получение клиентов для ComboBox...")
    
    // Используем СМЕШАННЫЙ регистр
    rows, err := db.Query(`
        SELECT "id_клиента", "ФИО" 
        FROM "Клиент" 
        ORDER BY "ФИО"
    `)
    if err != nil {
        log.Printf("❌ GetClientsForSelect ошибка: %v", err)
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка получения клиентов: " + err.Error(),
        })
    }
    defer rows.Close()
    
    var clients []fiber.Map
    for rows.Next() {
        var id int
        var name string
        err := rows.Scan(&id, &name)
        if err != nil {
            log.Printf("❌ Ошибка сканирования клиента: %v", err)
            continue
        }
        clients = append(clients, fiber.Map{
            "id":   id,
            "name": name,
        })
    }
    
    log.Printf("✅ GetClientsForSelect: загружено %d клиентов", len(clients))
    
    return c.JSON(fiber.Map{
        "success": true,
        "clients": clients,
    })
}

// GetTrainersForSelect возвращает список тренеров для ComboBox
func GetTrainersForSelect(c *fiber.Ctx) error {
    db := database.GetDB()
    
    rows, err := db.Query(`
        SELECT "id_тренера", "ФИО" 
        FROM "Тренер" 
        WHERE "Активен" = true 
        ORDER BY "ФИО"
    `)
    if err != nil {
        log.Printf("❌ Ошибка получения тренеров: %v", err)
        return c.JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка получения тренеров",
        })
    }
    defer rows.Close()
    
    var trainers []fiber.Map
    for rows.Next() {
        var id int
        var name string
        err := rows.Scan(&id, &name)
        if err != nil {
            continue
        }
        trainers = append(trainers, fiber.Map{
            "id":   id,
            "name": name,
        })
    }
    
    return c.JSON(fiber.Map{
        "success":  true,
        "trainers": trainers,
    })
}