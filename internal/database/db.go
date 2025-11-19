package database

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"fitness-center-manager/internal/config"
	_ "github.com/lib/pq"
)

var (
	instance *sql.DB
	once     sync.Once
)

// GetDB возвращает singleton *sql.DB. При первом вызове инициализирует подключение.
func GetDB() *sql.DB {
	once.Do(func() {
		db, err := initDB()
		if err != nil {
			log.Fatalf("DB init error: %v", err)
		}
		instance = db
	})
	return instance
}

// Close аккуратно закрывает пул соединений
func Close() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}

// initDB создаёт подключение к Postgres c учётом таймаутов/пула из конфига.
func initDB() (*sql.DB, error) {
	cfg := config.LoadConfig()
	dsn := cfg.Database.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// ==== Таймауты/пул из конфига с дефолтами ====
	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 25
	}
	lifeMin := cfg.Database.ConnMaxLifetimeMinutes
	if lifeMin <= 0 {
		lifeMin = 5
	}
	idleMin := cfg.Database.ConnMaxIdleMinutes
	if idleMin <= 0 {
		idleMin = 1
	}
	pingSec := cfg.Database.ConnectTimeoutSeconds
	if pingSec <= 0 {
		pingSec = 5
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(lifeMin) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(idleMin) * time.Minute)

	// Быстрый ping с таймаутом, чтобы не зависать.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(pingSec)*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	log.Println("Успешное подключение к PostgreSQL")
	return db, nil
}
