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

    // Настройка base URL для Problem Details (если задан в конфиге)
    if cfg.Server.ProblemBaseURL != "" {
        handlers.SetProblemBaseURL(cfg.Server.ProblemBaseURL)
    }

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
    // отчетность: запросы и операции (без добавления новых файлов)
    app.Post("/about/query/clients-after-date", handlers.ReportClientsAfterDate)
    app.Post("/about/query/subscriptions-by-status", handlers.ReportSubscriptionsByStatus)
    app.Post("/about/query/revenue-by-tariff", handlers.ReportRevenueByTariff)
    app.Post("/about/query/zones-min-equip", handlers.ReportZonesWithMinEquipment)
    app.Post("/about/op/insert-zone", handlers.ReportInsertZone)
    app.Post("/about/op/update-zone-status", handlers.ReportUpdateZoneStatus)
    app.Post("/about/op/delete-zone", handlers.ReportDeleteZone)

	// клиенты
	app.Get("/clients", handlers.GetClients)
	app.Post("/clients", handlers.CreateClient)
	app.Get("/clients/:id", handlers.GetClientByID)
	app.Put("/clients/:id", handlers.UpdateClient)
	app.Delete("/clients/:id", handlers.DeleteClient)

	// API v1 — клиенты (JSON)
	app.Get("/api/v1/clients", handlers.APIv1ListClients)
	app.Post("/api/v1/clients", handlers.APIv1CreateClient)
	app.Get("/api/v1/clients/:id", handlers.GetClientByID)
	app.Put("/api/v1/clients/:id", handlers.UpdateClient)
	app.Delete("/api/v1/clients/:id", handlers.DeleteClient)

	// абонементы
	app.Get("/subscriptions", handlers.GetSubscriptionsPage)
	app.Post("/subscriptions", handlers.CreateSubscription)
	app.Get("/subscriptions/:id", handlers.GetSubscriptionByID)
	app.Put("/subscriptions/:id", handlers.UpdateSubscription)
	app.Delete("/subscriptions/:id", handlers.DeleteSubscription)
	// API v1 — абонементы (JSON + алиасы)
	app.Get("/api/v1/subscriptions", handlers.APIv1ListSubscriptions)
	app.Post("/api/v1/subscriptions", handlers.APIv1CreateSubscription)
	app.Get("/api/v1/subscriptions/:id", handlers.GetSubscriptionByID)
	app.Put("/api/v1/subscriptions/:id", handlers.UpdateSubscription)
	app.Delete("/api/v1/subscriptions/:id", handlers.DeleteSubscription)

	// тренеры
	app.Get("/trainers", handlers.GetTrainersPage)  
	app.Post("/trainers", handlers.CreateTrainer)
	app.Get("/trainers/:id", handlers.GetTrainerByID) 
	app.Put("/trainers/:id", handlers.UpdateTrainer)
	app.Delete("/trainers/:id", handlers.DeleteTrainer)
	// API v1 — тренеры (JSON + алиасы)
	app.Get("/api/v1/trainers", handlers.APIv1ListTrainers)
	app.Post("/api/v1/trainers", handlers.APIv1CreateTrainer)
	app.Get("/api/v1/trainers/:id", handlers.GetTrainerByID)
	app.Put("/api/v1/trainers/:id", handlers.UpdateTrainer)
	app.Delete("/api/v1/trainers/:id", handlers.DeleteTrainer)

	// групповые
	app.Get("/trainings", handlers.GetTrainingsPage)
	app.Get("/api/group-trainings/:id", handlers.GetGroupTrainingByID)
	app.Post("/group-trainings", handlers.CreateGroupTraining)
	app.Put("/group-trainings/:id", handlers.UpdateGroupTraining)
	app.Delete("/group-trainings/:id", handlers.DeleteGroupTraining)
	// API v1 — групповые тренировки (алиасы)
	app.Get("/api/v1/group-trainings", handlers.APIv1ListGroupTrainings)
	app.Get("/api/v1/group-trainings/:id", handlers.GetGroupTrainingByID)
	app.Post("/api/v1/group-trainings", handlers.CreateGroupTraining)
	app.Put("/api/v1/group-trainings/:id", handlers.UpdateGroupTraining)
	app.Delete("/api/v1/group-trainings/:id", handlers.DeleteGroupTraining)

	// персональные
	app.Get("/api/personal-trainings/:id", handlers.GetPersonalTrainingByID)
	app.Post("/personal-trainings", handlers.CreatePersonalTraining)
	app.Put("/personal-trainings/:id", handlers.UpdatePersonalTraining)
	app.Delete("/personal-trainings/:id", handlers.DeletePersonalTraining)
	// API v1 — персональные тренировки (алиасы)
	app.Get("/api/v1/personal-trainings", handlers.APIv1ListPersonalTrainings)
	app.Get("/api/v1/personal-trainings/:id", handlers.GetPersonalTrainingByID)
	app.Post("/api/v1/personal-trainings", handlers.CreatePersonalTraining)
	app.Put("/api/v1/personal-trainings/:id", handlers.UpdatePersonalTraining)
	app.Delete("/api/v1/personal-trainings/:id", handlers.DeletePersonalTraining)

	// запись на групповую
	app.Get("/api/group-trainings/:id/enrollments", handlers.ListGroupEnrollments)
	app.Post("/group-enrollments", handlers.CreateGroupEnrollment)
	// API v1 — записи на групповые (алиасы)
	app.Get("/api/v1/group-trainings/:id/enrollments", handlers.ListGroupEnrollments)
	app.Post("/api/v1/group-enrollments", handlers.CreateGroupEnrollment)
	// API для селектов
	app.Get("/api/clients-for-select", handlers.GetClientsForSelect)
	app.Get("/api/tariffs-for-select", handlers.GetTariffsForSelect)
	app.Get("/api/trainers-for-select", handlers.GetTrainersForSelect)
	app.Get("/api/zones-for-select", handlers.GetZonesForSelect)
	app.Get("/api/subscriptions-for-select", handlers.GetSubscriptionsForSelect)
	// API v1 — селекты (алиасы)
	app.Get("/api/v1/clients-for-select", handlers.GetClientsForSelect)
	app.Get("/api/v1/tariffs-for-select", handlers.GetTariffsForSelect)
	app.Get("/api/v1/trainers-for-select", handlers.GetTrainersForSelect)
	app.Get("/api/v1/zones-for-select", handlers.GetZonesForSelect)
	app.Get("/api/v1/subscriptions-for-select", handlers.GetSubscriptionsForSelect)
	// зоны
	app.Get("/zones", handlers.GetZones)
	app.Post("/zones", handlers.CreateZone)
	app.Get("/zones/:id/photo", handlers.GetZonePhoto)
	app.Post("/zones/:id/upload-photo", handlers.UploadZonePhoto)
	app.Delete("/zones/:id/photo", handlers.ClearZonePhoto)
	app.Delete("/zones/:id", handlers.DeleteZone)
	app.Get("/api/zones/:id", handlers.GetZoneByID)

	// API v1 — фото зон (RESTful-алиасы)
	app.Get("/api/v1/zones/:id/photo", handlers.GetZonePhoto)
	app.Put("/api/v1/zones/:id/photo", handlers.UploadZonePhoto)
	app.Delete("/api/v1/zones/:id/photo", handlers.ClearZonePhoto)

	// страницы
	app.Get("/equipment", handlers.GetEquipmentPage)

	// API
	app.Get("/api/zones-for-select", handlers.GetZonesForSelect)
	app.Get("/api/equipment/:id", handlers.GetEquipmentByID)
	app.Get("/api/repairs/latest", handlers.GetLatestRepairs)
	// API v1 — алиасы
	app.Get("/api/v1/equipment", handlers.APIv1ListEquipment)
	app.Get("/api/v1/equipment/:id", handlers.GetEquipmentByID)
	app.Get("/api/v1/repairs/latest", handlers.GetLatestRepairs)

	// API v1 — отчёты (алиасы под REST-префиксом)
	app.Post("/api/v1/reports/clients-after-date", handlers.ReportClientsAfterDate)
	app.Post("/api/v1/reports/subscriptions-by-status", handlers.ReportSubscriptionsByStatus)
	app.Post("/api/v1/reports/revenue-by-tariff", handlers.ReportRevenueByTariff)
	app.Post("/api/v1/reports/zones-min-equip", handlers.ReportZonesWithMinEquipment)
	app.Post("/api/v1/reports/ops/insert-zone", handlers.ReportInsertZone)
	app.Post("/api/v1/reports/ops/update-zone-status", handlers.ReportUpdateZoneStatus)
	app.Post("/api/v1/reports/ops/delete-zone", handlers.ReportDeleteZone)

	// CRUD оборудования
	app.Post("/equipment", handlers.CreateEquipment)
	app.Put("/equipment/:id", handlers.UpdateEquipment)
	app.Delete("/equipment/:id", handlers.DeleteEquipment)
	// API v1 — CRUD оборудования (алиасы)
	app.Post("/api/v1/equipment", handlers.CreateEquipment)
	app.Put("/api/v1/equipment/:id", handlers.UpdateEquipment)
	app.Delete("/api/v1/equipment/:id", handlers.DeleteEquipment)

	// фото оборудования
	app.Post("/equipment/:id/upload-photo", handlers.UploadEquipmentPhoto)
	app.Get("/equipment/:id/photo", handlers.GetEquipmentPhoto)
	app.Delete("/equipment/:id/photo", handlers.DeleteEquipmentPhoto)

	// API v1 — фото оборудования (RESTful-алиасы)
	app.Get("/api/v1/equipment/:id/photo", handlers.GetEquipmentPhoto)
	app.Put("/api/v1/equipment/:id/photo", handlers.UploadEquipmentPhoto)
	app.Delete("/api/v1/equipment/:id/photo", handlers.DeleteEquipmentPhoto)

	// заявки на ремонт
	app.Post("/repairs", handlers.CreateRepairRequest)
	app.Get("/repairs/:id/photo", handlers.GetRepairPhoto)
	app.Post("/repairs/:id/upload-photo", handlers.UploadRepairPhoto)
	app.Put("/repairs/:id", handlers.UpdateRepairRequest)
	app.Delete("/repairs/:id", handlers.DeleteRepairRequest)
	// API v1 — заявки на ремонт (алиасы)
	app.Post("/api/v1/repairs", handlers.CreateRepairRequest)
	app.Get("/api/v1/repairs/:id/photo", handlers.GetRepairPhoto)
	app.Put("/api/v1/repairs/:id/photo", handlers.UploadRepairPhoto)
	app.Put("/api/v1/repairs/:id", handlers.UpdateRepairRequest)
	app.Delete("/api/v1/repairs/:id", handlers.DeleteRepairRequest)

}
