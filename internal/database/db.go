package database

import (
    "database/sql"
    "fmt"
    "log"
    "sync"
    "fitness-center-manager/internal/config"
    _ "github.com/lib/pq"
)

var (
    instance *sql.DB
    once     sync.Once
)



func GetDB() *sql.DB{
	once.Do(func(){
		cfg := config.LoadConfig()
		dbConfig := cfg.Database
		
        connectionStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
            dbConfig.Host, 
            dbConfig.Port, 
            dbConfig.User, 
            dbConfig.Password,
            dbConfig.DBName, 
            dbConfig.SSLMode)
        
        var err error

        instance, err = sql.Open("postgres", connectionStr)
        if err != nil{
            log.Fatal("Ошибка подключения к БД:", err)
        }

        if err = instance.Ping(); err != nil{
            log.Fatal("Ошибка ping БД:", err)
        }

        instance.SetMaxOpenConns(25)
        instance.SetMaxIdleConns(25)
        instance.SetConnMaxLifetime(5 * 60)
        log.Println("Успешное подключение к PostgreSQL")
	})

    return instance
}

func TestConnection() error {
    db := GetDB()
    
    var result int
    err := db.QueryRow("SELECT 1").Scan(&result)
    if err != nil {
        return fmt.Errorf("ошибка тестового запроса: %v", err)
    }
    
    if result != 1 {
        return fmt.Errorf("неожиданный результат теста: %d", result)
    }
    
    log.Println("Тестовый запрос к БД выполнен успешно")
    return nil
}