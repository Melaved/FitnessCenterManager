package main

import (
	"fitness-center-manager/internal/config"
	"fitness-center-manager/internal/database"
	"log"
)

func main() {
    log.Println("Запуск теста подключения к БД...")
    
    cfg := config.LoadConfig()
    log.Printf("Настройки БД: %s@%s:%s", 
        cfg.Database.User, cfg.Database.Host, cfg.Database.DBName)
    
    db := database.GetDB()
    log.Println("Подключение к БД установлено")
    
    err := database.TestConnection()
    if err != nil {
        log.Fatalf("Ошибка тестирования БД: %v", err)
    }
    
    var dbName string
    err = db.QueryRow("SELECT current_database()").Scan(&dbName)
    if err != nil {
        log.Fatalf("Ошибка получения имени БД: %v", err)
    }
    log.Printf("Используемая БД: %s", dbName)
    
    var tableCount int
    err = db.QueryRow(`
        SELECT COUNT(*) 
        FROM information_schema.tables 
        WHERE table_schema = 'public'
    `).Scan(&tableCount)
    if err != nil {
        log.Fatalf("Ошибка проверки таблиц: %v", err)
    }
    log.Printf("Количество таблиц в БД: %d", tableCount)
    
    log.Println("Этап 2 завершен! БД готова к работе")
    
}