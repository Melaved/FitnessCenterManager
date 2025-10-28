package handlers

import (
    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"
    "github.com/gofiber/fiber/v2"
)

func GetTrainers(c *fiber.Ctx) error {
    db := database.GetDB()
    
    rows, err := db.Query(`
        SELECT 
            "id_тренера", 
            "ФИО", 
            "Номер_телефона", 
            "Специализация", 
            "Дата_найма", 
            "Стаж_работы", 
        FROM "Тренер" 
        ORDER BY "id_тренера"
    `)
    if err != nil {
        return c.Status(500).SendString("Ошибка получения тренеров: " + err.Error())
    }
    defer rows.Close()
    
    var trainers []models.Trainer
    for rows.Next() {
        var trainer models.Trainer
        err := rows.Scan(
            &trainer.ID,
            &trainer.FIO,
            &trainer.Phone,
            &trainer.Specialization,
            &trainer.HireDate,
            &trainer.Experience,
        )
        if err != nil {
            return c.Status(500).SendString("Ошибка сканирования тренера: " + err.Error())
        }
        trainers = append(trainers, trainer)
    }
    
    return c.Render("trainers", fiber.Map{
        "Title":    "Тренеры",
        "Trainers": trainers,
    })
}