package handlers

import (
    "github.com/gofiber/fiber/v2"
)

func GetTrainings(c *fiber.Ctx) error {
    return c.Render("trainings", fiber.Map{
        "Title": "Тренировки", 
        "Message": "Страница в разработке",
    })
}

func GetZones(c *fiber.Ctx) error {
    return c.Render("zones", fiber.Map{
        "Title": "Зоны",
        "Message": "Страница в разработке",
    })
}

func GetEquipment(c *fiber.Ctx) error {
    return c.Render("equipment", fiber.Map{
        "Title": "Оборудование",
        "Message": "Страница в разработке", 
    })
}