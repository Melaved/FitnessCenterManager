package handlers

import (
    "log"

    "github.com/gofiber/fiber/v2"
)

// jsonError logs the internal error and returns a safe JSON payload to the client.
func jsonError(c *fiber.Ctx, status int, publicMsg string, err error) error {
    if err != nil {
        log.Printf("handler error: %v", err)
    }
    return c.Status(status).JSON(fiber.Map{
        "success": false,
        "error":   publicMsg,
    })
}

// jsonOK ensures a success flag and returns JSON.
func jsonOK(c *fiber.Ctx, payload fiber.Map) error {
    if payload == nil {
        payload = fiber.Map{}
    }
    payload["success"] = true
    return c.JSON(payload)
}

