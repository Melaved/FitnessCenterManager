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
		Views:       engine,
		AppName:     "FitnessCenterManager",
		ViewsLayout: "layouts/base",
		BodyLimit:   10 * 1024 * 1024, // до 10 МБ на запрос
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
	// страницы
	app.Get("/", handlers.Dashboard)
	app.Get("/about", handlers.About)

	// клиенты
	app.Get("/clients", handlers.GetClients)
	app.Post("/clients", handlers.CreateClient)
	app.Get("/clients/:id", handlers.GetClientByID)
	app.Put("/clients/:id", handlers.UpdateClient)
	app.Delete("/clients/:id", handlers.DeleteClient)

	// абонементы
	app.Get("/subscriptions", handlers.GetSubscriptionsPage)
	app.Post("/subscriptions", handlers.CreateSubscription)
	app.Get("/subscriptions/:id", handlers.GetSubscriptionByID)
	app.Put("/subscriptions/:id", handlers.UpdateSubscription)
	app.Delete("/subscriptions/:id", handlers.DeleteSubscription)

	// тренеры
	app.Get("/trainers", handlers.GetTrainersPage)      // HTML страница
	app.Post("/trainers", handlers.CreateTrainer)       // создать
	app.Get("/trainers/:id", handlers.GetTrainerByID)   // JSON для формы редактирования
	app.Put("/trainers/:id", handlers.UpdateTrainer)    // обновить
	app.Delete("/trainers/:id", handlers.DeleteTrainer) // удалить

	// групповые
	app.Get("/trainings", handlers.GetTrainingsPage)
	app.Get("/api/group-trainings/:id", handlers.GetGroupTrainingByID)
	app.Post("/group-trainings", handlers.CreateGroupTraining)
	app.Put("/group-trainings/:id", handlers.UpdateGroupTraining)
	app.Delete("/group-trainings/:id", handlers.DeleteGroupTraining)

	// персональные
	app.Get("/api/personal-trainings/:id", handlers.GetPersonalTrainingByID)
	app.Post("/personal-trainings", handlers.CreatePersonalTraining)
	app.Put("/personal-trainings/:id", handlers.UpdatePersonalTraining)
	app.Delete("/personal-trainings/:id", handlers.DeletePersonalTraining)

	// запись на групповую
	app.Get("/api/group-trainings/:id/enrollments", handlers.ListGroupEnrollments)
	app.Post("/group-enrollments", handlers.CreateGroupEnrollment)
	// API для селектов
	app.Get("/api/clients-for-select", handlers.GetClientsForSelect)
	app.Get("/api/tariffs-for-select", handlers.GetTariffsForSelect)
	app.Get("/api/trainers-for-select", handlers.GetTrainersForSelect)
	app.Get("/api/zones-for-select", handlers.GetZonesForSelect)
	app.Get("/api/subscriptions-for-select", handlers.GetSubscriptionsForSelect)
	// зоны
	app.Get("/zones", handlers.GetZones)
	app.Post("/zones", handlers.CreateZone)
	app.Get("/zones/:id/photo", handlers.GetZonePhoto)
	app.Post("/zones/:id/upload-photo", handlers.UploadZonePhoto)
	app.Delete("/zones/:id/photo", handlers.ClearZonePhoto)
	app.Delete("/zones/:id", handlers.DeleteZone)
	app.Get("/api/zones/:id", handlers.GetZoneByID)

	// страницы
	app.Get("/equipment", handlers.GetEquipmentPage)

	// API
	app.Get("/api/zones-for-select", handlers.GetZonesForSelect)
	app.Get("/api/equipment/:id", handlers.GetEquipmentByID)
	app.Get("/api/repairs/latest", handlers.GetLatestRepairs)

	// CRUD оборудования
	app.Post("/equipment", handlers.CreateEquipment)
	app.Put("/equipment/:id", handlers.UpdateEquipment)
	app.Delete("/equipment/:id", handlers.DeleteEquipment)

	// фото оборудования
	app.Post("/equipment/:id/upload-photo", handlers.UploadEquipmentPhoto)
	app.Get("/equipment/:id/photo", handlers.GetEquipmentPhoto)
	app.Delete("/equipment/:id/photo", handlers.DeleteEquipmentPhoto)

	// заявки на ремонт
	app.Post("/repairs", handlers.CreateRepairRequest)
	app.Get("/repairs/:id/photo", handlers.GetRepairPhoto)
	app.Delete("/repairs/:id", handlers.DeleteRepairRequest)

}
