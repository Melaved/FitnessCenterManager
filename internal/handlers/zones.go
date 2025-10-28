package handlers

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "fitness-center-manager/internal/config"
    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"
    "github.com/gofiber/fiber/v2"
)

// GetZones отображает страницу зон
func GetZones(c *fiber.Ctx) error {
    db := database.GetDB()
    
    log.Println("🔍 Получение зон из БД...")
    
    rows, err := db.Query(`
        SELECT 
            "id_зоны",
            "Название", 
            "Описание",
            "Вместимость",
            "Статус",
            "Фото"
        FROM "Зона" 
        ORDER BY "id_зоны"
    `)
    
    if err != nil {
        log.Printf("❌ Ошибка получения зон: %v", err)
        return c.Render("zones", fiber.Map{
            "Title": "Зоны",
            "Zones": []models.Zone{},
            "Error": "Не удалось загрузить данные зон: " + err.Error(),
        })
    }
    defer rows.Close()
    
    var zones []models.Zone
    for rows.Next() {
        var zone models.Zone
        err := rows.Scan(
            &zone.ID,
            &zone.Name,
            &zone.Description, 
            &zone.Capacity,
            &zone.Status,
            &zone.PhotoPath,
        )
        if err != nil {
            log.Printf("❌ Ошибка сканирования зоны: %v", err)
            continue
        }
        zones = append(zones, zone)
    }
    
    if err = rows.Err(); err != nil {
        log.Printf("❌ Ошибка после итерации по зонам: %v", err)
    }
    
    log.Printf("✅ Загружено %d зон из БД", len(zones))
    
    return c.Render("zones", fiber.Map{
        "Title": "Зоны",
        "Zones": zones,
    })
}

// CreateZone создает новую зону
func CreateZone(c *fiber.Ctx) error {
    log.Println("🎯 Создание новой зоны...")
    
    type ZoneForm struct {
        Name        string `form:"name"`
        Description string `form:"description"` 
        Capacity    int    `form:"capacity"`
        Status      string `form:"status"`
    }
    
    var form ZoneForm
    if err := c.BodyParser(&form); err != nil {
        log.Printf("❌ Ошибка парсинга формы: %v", err)
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Неверные данные формы: " + err.Error(),
        })
    }
    
    // Валидация
    if form.Name == "" {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Название зоны обязательно",
        })
    }
    
    if form.Capacity <= 0 {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Вместимость должна быть положительным числом",
        })
    }
    
    db := database.GetDB()
    
    log.Printf("📝 Данные для сохранения: Name=%s, Description=%s, Capacity=%d, Status=%s", 
        form.Name, form.Description, form.Capacity, form.Status)
    
    var zoneID int
    err := db.QueryRow(`
        INSERT INTO "Зона" ("Название", "Описание", "Вместимость", "Статус")
        VALUES ($1, $2, $3, $4)
        RETURNING "id_зоны"
    `, form.Name, form.Description, form.Capacity, form.Status).Scan(&zoneID)
    
    if err != nil {
        log.Printf("❌ Ошибка создания зоны в БД: %v", err)
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка создания зоны в базе данных: " + err.Error(),
        })
    }
    
    log.Printf("✅ Создана новая зона: %s (ID: %d)", form.Name, zoneID)
    
    return c.JSON(fiber.Map{
        "success": true,
        "message": "Зона успешно создана",
        "zone_id": zoneID,
    })
}

// UploadZonePhoto загружает фото для зоны
func UploadZonePhoto(c *fiber.Ctx) error {
    zoneID := c.Params("id")
    log.Printf("🎯 Загрузка фото для зоны ID: %s", zoneID)
    
    // Получаем файл из формы
    file, err := c.FormFile("photo")
    if err != nil {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Файл не получен: " + err.Error(),
        })
    }
    
    // Проверяем тип файла
    allowedTypes := map[string]bool{
        "image/jpeg": true,
        "image/png":  true,
    }
    
    if !allowedTypes[file.Header.Get("Content-Type")] {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Разрешены только JPEG, PNG файлы",
        })
    }
    
    // Проверяем размер файла (максимум 5MB)
    if file.Size > 5*1024*1024 {
        return c.Status(400).JSON(fiber.Map{
            "success": false,
            "error":   "Размер файла не должен превышать 5MB",
        })
    }
    
    cfg := config.LoadConfig()
    
    // Создаем папку для загрузок если не существует
    uploadPath := cfg.Server.UploadPath
    if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
        os.MkdirAll(uploadPath, 0755)
        log.Printf("📁 Создана папка для загрузок: %s", uploadPath)
    }
    
    // Генерируем уникальное имя файла
    fileExt := filepath.Ext(file.Filename)
    fileName := fmt.Sprintf("zone_%s%s", zoneID, fileExt)
    filePath := filepath.Join(uploadPath, fileName)
    
    // Сохраняем файл
    if err := c.SaveFile(file, filePath); err != nil {
        log.Printf("❌ Ошибка сохранения файла: %v", err)
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка сохранения файла: " + err.Error(),
        })
    }
    
    // Обновляем путь к фото в БД
    db := database.GetDB()
    _, err = db.Exec(`
        UPDATE "Зона" 
        SET "Фото" = $1 
        WHERE "id_зоны" = $2
    `, fileName, zoneID)
    
    if err != nil {
        // Удаляем файл если не удалось обновить БД
        os.Remove(filePath)
        return c.Status(500).JSON(fiber.Map{
            "success": false,
            "error":   "Ошибка обновления базы данных: " + err.Error(),
        })
    }
    
    log.Printf("✅ Фото загружено: %s", fileName)
    
    return c.JSON(fiber.Map{
        "success": true,
        "message": "Фото успешно загружено",
        "file_name": fileName,
    })
}

// GetZonePhoto возвращает фото зоны
func GetZonePhoto(c *fiber.Ctx) error {
    fileName := c.Params("filename")
    cfg := config.LoadConfig()
    
    filePath := filepath.Join(cfg.Server.UploadPath, fileName)
    
    // Проверяем существует ли файл
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return c.Status(404).SendString("Файл не найден")
    }
    
    return c.SendFile(filePath)
}