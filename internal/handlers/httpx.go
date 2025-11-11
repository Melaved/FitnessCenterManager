package handlers

import (
    "log"
    "strings"

    "github.com/gofiber/fiber/v2"
)

var problemBaseURL string

// SetProblemBaseURL задаёт базовый URL для поля type Problem Details.
// Пример: https://fitness-center-manager.dev/problem
func SetProblemBaseURL(base string) {
    problemBaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
}

// jsonError — единый ответ об ошибке в формате RFC 7807 (application/problem+json)
// Для обратной совместимости добавляет поля success=false и error.
func jsonError(c *fiber.Ctx, status int, publicMsg string, err error) error {
    if err != nil {
        log.Printf("handler error: %v", err)
    }
    if publicMsg == "" {
        publicMsg = fiber.ErrInternalServerError.Message
    }
    pType := problemType(publicMsg, status, c.OriginalURL())
    problem := fiber.Map{
        "type":     pType,
        "title":    publicMsg,
        "status":   status,
        "instance": c.OriginalURL(),
    }
    if err != nil {
        problem["detail"] = err.Error()
    }
    // backward-compat fields
    problem["success"] = false
    problem["error"] = publicMsg

    c.Type("application/problem+json")
    return c.Status(status).JSON(problem)
}

func jsonOK(c *fiber.Ctx, payload fiber.Map) error {
    if payload == nil {
        payload = fiber.Map{}
    }
    payload["success"] = true
    c.Type("application/json")
    return c.JSON(payload)
}

// problemType возвращает осмысленный URI для поля "type" Problem Details.
// Базовая схема использует URN, чтобы не зависеть от внешнего домена.
func problemType(title string, status int, _path string) string {
    t := strings.ToLower(strings.TrimSpace(title))
    code := ""
    // Частные случаи по тексту сообщения → код
    switch {
    case strings.Contains(t, "некорректный id"):
        code = "invalid-id"
    case strings.Contains(t, "неверные данные формы"):
        code = "invalid-form"
    case strings.Contains(t, "заполните обязательные поля"):
        code = "missing-required-fields"
    case strings.Contains(t, "неверный формат даты") || strings.Contains(t, "неверная дата"):
        code = "invalid-date"
    case strings.Contains(t, "дата окончания раньше даты начала"):
        code = "invalid-date-range"
    case strings.Contains(t, "не найден") || strings.Contains(t, "не найдено"):
        code = "not-found"
    case strings.Contains(t, "ошибка бд") || strings.Contains(t, "db: ошибка") || strings.Contains(t, "ошибка сохранения в бд"):
        code = "database-error"
    case strings.Contains(t, "невозможно удалить"):
        code = "conflict"
    case strings.Contains(t, "файл пустой") || strings.Contains(t, "превышает 5 мб") || strings.Contains(t, "больше 5 мб"):
        code = "file-too-large"
    case strings.Contains(t, "jpeg/png/webp") || strings.Contains(t, "разрешены jpeg") || strings.Contains(t, "только jpeg"):
        code = "invalid-image-type"
    case strings.Contains(t, "недопустимый статус") || strings.Contains(t, "неверный статус"):
        code = "invalid-status"
    }
    if code == "" {
        // Общее соответствие по HTTP-статусу
        switch status {
        case fiber.StatusBadRequest:
            code = "validation-error"
        case fiber.StatusUnauthorized:
            code = "unauthorized"
        case fiber.StatusForbidden:
            code = "forbidden"
        case fiber.StatusNotFound:
            code = "not-found"
        case fiber.StatusConflict:
            code = "conflict"
        case fiber.StatusRequestEntityTooLarge:
            code = "request-entity-too-large"
        default:
            code = "internal-error"
        }
    }
    if problemBaseURL != "" && (strings.HasPrefix(problemBaseURL, "http://") || strings.HasPrefix(problemBaseURL, "https://")) {
        return problemBaseURL + "/" + code
    }
    return "urn:fitness-center-manager:problem:" + code
}
