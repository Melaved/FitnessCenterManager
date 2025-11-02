package main

import (
	"log"
	"time"

	"fitness-center-manager/internal/config"
	"fitness-center-manager/internal/database"
	"fitness-center-manager/internal/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
		Views:     engine,
		AppName:   "FitnessCenterManager",
		BodyLimit: 10 * 1024 * 1024, // до 10 МБ на запрос
	})

	// -------------------------------
	// Middleware: безопасность и логика
	// -------------------------------

	app.Use(recover.New())  // Перехватывает паники, возвращает 500 вместо краша
	app.Use(helmet.New())   // Добавляет HTTP security-заголовки
	app.Use(compress.New()) // Сжимает ответы gzip/br
	app.Use(logger.New())   // Логи запросов
	app.Use(limiter.New(limiter.Config{
		Max:        120,         // 120 запросов
		Expiration: time.Minute, // за минуту
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("Слишком много запросов. Попробуйте позже.")
		},
	}))
	app.Use(etag.New()) // Ускоряет GET-запросы через кэширование по ETag

	// -------------------------------
	// Статика и маршруты
	// -------------------------------
	app.Static("/static", cfg.Server.StaticPath)

	setupRoutes(app)

	log.Printf("🚀 Сервер запущен на http://localhost%s", cfg.Server.Port)
	log.Printf("📊 Главная: http://localhost%s/", cfg.Server.Port)
	log.Printf("👥 Клиенты: http://localhost%s/clients", cfg.Server.Port)

	log.Fatal(app.Listen(cfg.Server.Port))
}

// setupRoutes — маршруты приложения
func setupRoutes(app *fiber.App) {
	// Главная
	app.Get("/", handlers.Dashboard)
	app.Get("/about", handlers.About)

	// Клиенты
	app.Get("/clients", handlers.GetClients)
	app.Post("/clients", handlers.CreateClient)
	app.Get("/clients/:id", handlers.GetClientByID)
	app.Put("/clients/:id", handlers.UpdateClient)
	app.Delete("/clients/:id", handlers.DeleteClient)

	// Абонементы
	app.Get("/subscriptions", handlers.GetSubscriptions)
	app.Get("/api/clients-for-select", handlers.GetClientsForSelect)
	app.Get("/api/trainers-for-select", handlers.GetTrainersForSelect)

	// Зоны с загрузкой фото
	// Зоны
	app.Get("/zones", handlers.GetZones)                          // страница
	app.Get("/api/zones/:id", handlers.GetZoneByID)               // получить одну зону (JSON)
	app.Post("/zones", handlers.CreateZone)                       // создать (JSON)
	app.Post("/zones/:id/upload-photo", handlers.UploadZonePhoto) // загрузка фото (JSON)
	app.Put("/zones/:id", handlers.UpdateZone)                    // обновить (JSON)
	app.Delete("/zones/:id", handlers.DeleteZone)                 // удалить (JSON)
	app.Delete("/zones/:id/photo", handlers.ClearZonePhoto)       // очистить фото (JSON)
	app.Get("/zones/:id/photo", handlers.GetZonePhoto)            // отдача картинки для <img>

	// Остальные
	app.Get("/trainers", handlers.GetTrainers)
	app.Get("/trainings", handlers.GetTrainings)
	app.Get("/equipment", handlers.GetEquipment)
}
