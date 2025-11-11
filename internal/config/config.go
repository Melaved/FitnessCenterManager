package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config — корневая структура приложения, читается из YAML.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
}

// DatabaseConfig — настройки подключения к Postgres + параметры пула.
type DatabaseConfig struct {
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
	User    string `yaml:"user"`
	Password string `yaml:"password"` // берём из config.secret.yaml
	DBName  string `yaml:"dbname"`
	SSLMode string `yaml:"sslmode"`

	MaxOpenConns           int `yaml:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int `yaml:"conn_max_lifetime_minutes"`
	ConnMaxIdleMinutes     int `yaml:"conn_max_idle_minutes"`
	ConnectTimeoutSeconds  int `yaml:"connect_timeout_seconds"`
}

// DSN формирует keyword-DSN для github.com/lib/pq.
// Пример: host=... port=... user=... password=... dbname=... sslmode=...
func (d DatabaseConfig) DSN() string {
    user := url.QueryEscape(d.User)
    pass := url.QueryEscape(strings.TrimSpace(d.Password))

    // Поддержка UNIX-сокета: host начинается с "/" → формируем DSN через query-параметры
    if strings.HasPrefix(d.Host, "/") {
        // postgres://user:pass@/dbname?host=/var/run/postgresql&port=5432&sslmode=disable
        q := url.Values{}
        q.Set("host", d.Host)
        if d.Port != "" {
            q.Set("port", d.Port)
        }
        if d.SSLMode != "" {
            q.Set("sslmode", d.SSLMode)
        }
        return fmt.Sprintf("postgres://%s:%s@/%s?%s", user, pass, d.DBName, q.Encode())
    }

    // Обычное TCP-подключение
    return fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=%s",
        user, pass, d.Host, d.Port, d.DBName, d.SSLMode,
    )
}

// ServerConfig — базовые настройки веб-сервера и путей.
type ServerConfig struct {
    Port         string `yaml:"port"`
    TemplatePath string `yaml:"template_path"`
    StaticPath   string `yaml:"static_path"`
    UploadPath   string `yaml:"upload_path"`
    ProblemBaseURL string `yaml:"problem_base_url"`
}

// LoadConfig загружает конфигурацию из config.yaml и опционально из config.secret.yaml.
// Пароль БД подмешивается из секрета, если файл существует.
// Если пароля нет — выводится предупреждение.
func LoadConfig() *Config {
	cfg := &Config{}

	// 1) основной конфиг
	mainBytes, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Ошибка чтения config.yaml: %v", err)
	}
	if err := yaml.Unmarshal(mainBytes, cfg); err != nil {
		log.Fatalf("Ошибка парсинга config.yaml: %v", err)
	}

	// 2) секретный конфиг (опционально)
	secretBytes, err := os.ReadFile("config.secret.yaml")
	if err == nil {
		var secret struct {
			Database struct {
				Password string `yaml:"password"`
			} `yaml:"database"`
		}
		if err := yaml.Unmarshal(secretBytes, &secret); err != nil {
			log.Printf("⚠️  Ошибка парсинга config.secret.yaml: %v", err)
		} else if secret.Database.Password != "" {
			cfg.Database.Password = secret.Database.Password
		}
	} else {
		log.Println("⚠️  config.secret.yaml не найден — пароль БД не установлен (допустимо только в dev)")
	}

	// 3) базовая валидация и информативные логи
	if cfg.Database.Host == "" || cfg.Database.Port == "" || cfg.Database.User == "" || cfg.Database.DBName == "" {
		log.Fatal("Некорректный database-конфиг: host/port/user/dbname должны быть заданы в config.yaml")
	}
	if cfg.Server.Port == "" {
		log.Fatal("Некорректный server-конфиг: server.port должен быть задан в config.yaml (например, :3000)")
	}
	if cfg.Database.Password == "" {
		log.Println("⚠️  Пароль БД пуст — убедись, что это режим разработки и есть доступ без пароля")
	}

	log.Printf("✅ Конфигурация загружена: %s:%s/%s (sslmode=%s)",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.SSLMode)

	return cfg
}
