package main

import (
    "log"
    "fitness-center-manager/internal/config"
)

func main() {
    log.Println("Запуск теста конфигурации...")
    
    cfg := config.LoadConfig()
    
    log.Printf("Конфиг загружен успешно!")
    log.Printf("База данных: %s@%s:%s", 
        cfg.Database.User, cfg.Database.Host, cfg.Database.DBName)
    log.Printf("Порт сервера: %s", cfg.Server.Port)
    log.Printf("Путь к шаблонам: %s", cfg.Server.TemplatePath)
    
}