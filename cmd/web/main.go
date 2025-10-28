package main

import (
    "log"
    "fitness-center-manager/internal/config"
    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/handlers"
    
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/template/html/v2"
)

func main() {
    // Загрузка конфигурации
    cfg := config.LoadConfig()
    
    // Инициализация базы данных
    _ = database.GetDB()
    
    // Инициализация шаблонов
    engine := html.New(cfg.Server.TemplatePath, ".html")
    
    // Создание приложения Fiber
    app := fiber.New(fiber.Config{
        Views: engine,
        // ViewsLayout: "base",
    })
    
    // Middleware
    app.Use(logger.New())
    app.Static("/static", cfg.Server.StaticPath)
    
    // Настройка маршрутов
    setupRoutes(app)
    
    log.Printf("🚀 Сервер запущен на http://localhost%s", cfg.Server.Port)
    log.Printf("📊 Главная: http://localhost%s/", cfg.Server.Port)
    log.Printf("👥 Клиенты: http://localhost%s/clients", cfg.Server.Port)
    log.Printf("🏃 Тренеры: http://localhost%s/trainers", cfg.Server.Port)
    
    log.Fatal(app.Listen(cfg.Server.Port))
}

func setupRoutes(app *fiber.App) {
    // Главная страница
    app.Get("/", handlers.Dashboard)
    
    // Клиенты
    app.Get("/clients", handlers.GetClients)
    app.Post("/clients", handlers.CreateClient)
    app.Get("/clients/:id", handlers.GetClientByID)
    app.Put("/clients/:id", handlers.UpdateClient)

     // Абонементы
    app.Get("/subscriptions", handlers.GetSubscriptions)
    app.Get("/api/clients-for-select", handlers.GetClientsForSelect)
    app.Get("/api/trainers-for-select", handlers.GetTrainersForSelect)
    // Основные разделы
    app.Get("/trainers", handlers.GetTrainers)
    app.Get("/subscriptions", handlers.GetSubscriptions)
    app.Get("/trainings", handlers.GetTrainings)
    app.Get("/zones", handlers.GetZones)
    app.Get("/equipment", handlers.GetEquipment)
    
    // Сервисные страницы
    app.Get("/about", handlers.About)
}