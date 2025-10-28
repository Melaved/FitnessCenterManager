package handlers

import (
    "log"
    "fitness-center-manager/internal/database"
    "github.com/gofiber/fiber/v2"
)

func Dashboard(c *fiber.Ctx) error {
    db := database.GetDB()
    
    var clientCount, trainerCount, subscriptionCount, trainingCount int
    
    db.QueryRow(`SELECT COUNT(*) FROM "Клиент"`).Scan(&clientCount)
    db.QueryRow(`SELECT COUNT(*) FROM "Тренер" WHERE "Активен" = true`).Scan(&trainerCount)
    db.QueryRow(`SELECT COUNT(*) FROM "Абонемент"`).Scan(&subscriptionCount)
    db.QueryRow(`SELECT COUNT(*) FROM "Персональная_тренировка" WHERE "Статус" = 'Запланирована'`).Scan(&trainingCount)
    
    log.Printf("📊 Статистика: Клиенты=%d, Тренеры=%d, Абонементы=%d, Тренировки=%d",
        clientCount, trainerCount, subscriptionCount, trainingCount)
    
    return c.Render("dashboard", fiber.Map{
    "Title": "Главная панель",
    "Stats": fiber.Map{
        "Clients":       clientCount,
        "Trainers":      trainerCount,
        "Subscriptions": subscriptionCount,
        "Trainings":     trainingCount,
    },
})
}