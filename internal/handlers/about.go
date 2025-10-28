package handlers

import "github.com/gofiber/fiber/v2"

func About(c *fiber.Ctx) error {
    return c.Render("about", fiber.Map{
    "Title": "О программе",
})
}