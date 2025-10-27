package main

import (
    "log"
    "fitness-center-manager/internal/database"
    "fitness-center-manager/internal/models"
)

func testModels() {
    db := database.GetDB()
    
    // Тест 1: Проверяем структуру таблицы Клиенты
    log.Println("Тестирование структуры таблицы Клиенты...")
    rows, err := db.Query(`
        SELECT column_name, data_type, is_nullable 
        FROM information_schema.columns 
        WHERE table_name = 'клиент' 
        ORDER BY ordinal_position
    `)
    if err != nil {
        log.Printf("Ошибка проверки структуры клиент: %v", err)
        return
    }
    defer rows.Close()
    
    log.Println("Структура таблицы 'клиент':")
    for rows.Next() {
        var columnName, dataType, nullable string
        rows.Scan(&columnName, &dataType, &nullable)
        log.Printf("   %s: %s (%s)", columnName, dataType, nullable)
    }
    
    // Тест 2: Пробуем получить несколько клиентов
    log.Println("Тестирование выборки клиентов...")
    clientRows, err := db.Query("SELECT id_клиента, ФИО, Номер_телефона FROM Клиент LIMIT 3")
    if err != nil {
        log.Printf("Не удалось получить клиентов (возможно таблица пуста): %v", err)
        return
    }
    defer clientRows.Close()
    
    var clients []models.Client
    for clientRows.Next() {
        var client models.Client
        err := clientRows.Scan(&client.ID, &client.FIO, &client.Phone)
        if err != nil {
            log.Printf("Ошибка сканирования клиента: %v", err)
            continue
        }
        clients = append(clients, client)
    }
    
    log.Printf("Успешно загружено %d клиентов", len(clients))
    for _, client := range clients {
        log.Printf("    %d: %s (%s)", client.ID, client.FIO, client.Phone)
    }
}

func main() {
    log.Println("Запуск теста моделей и БД...")
    
    // Тестируем подключение к БД
    db := database.GetDB()
    log.Println("Подключение к БД установлено")

    // Выполняем тестовый запрос
    err := database.TestConnection()
    if err != nil {
        log.Fatalf("Ошибка тестирования БД: %v", err)
    }
    
    // Проверяем что наша база fitness_center существует и доступна
    var dbName string
    err = db.QueryRow("SELECT current_database()").Scan(&dbName)
    if err != nil {
        log.Fatalf("Ошибка получения имени БД: %v", err)
    }
    log.Printf("Используемая БД: %s", dbName)
    
    // Проверяем таблицы
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
    
    // Если таблицы есть - тестируем модели
    if tableCount > 0 {
        testModels()
    } else {
        log.Println("В БД нет таблиц. Нужно создать структуру БД.")
    }
    
    log.Println("Этап 2.5 завершен! Модели данных готовы")
    log.Println("Следующий этап: Создание веб-сервера на Fiber")
}