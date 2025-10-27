package config

import (
    "os"
    "log"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Database struct {
        Host     string `yaml:"host"`
        Port     string `yaml:"port"`
        User     string `yaml:"user"`
        Password string `yaml:"password"`
        DBName   string `yaml:"dbname"`
        SSLMode  string `yaml:"sslmode"`
    } `yaml:"database"`
    Server struct {
        Port         string `yaml:"port"`
        TemplatePath string `yaml:"template_path"`
        StaticPath   string `yaml:"static_path"`
        UploadPath   string `yaml:"upload_path"`
    } `yaml:"server"`
}

// LoadConfig загружает конфигурацию из файлов
func LoadConfig() *Config {
    config := &Config{}
    
    // 1. Загружаем основной конфиг (без пароля)
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Fatalf("Ошибка чтения config.yaml: %v", err)
    }
    
    err = yaml.Unmarshal(data, config)
    if err != nil {
        log.Fatalf("Ошибка парсинга config.yaml: %v", err)
    }
    
    // 2. Загружаем секретный конфиг (с паролем)
    secretData, err := os.ReadFile("config.secret.yaml")
    if err != nil {
        log.Fatalf("Ошибка чтения config.secret.yaml: %v", err)
    }
    
    var secretConfig struct {
        Database struct {
            Password string `yaml:"password"`
        } `yaml:"database"`
    }
    
    err = yaml.Unmarshal(secretData, &secretConfig)
    if err != nil {
        log.Fatalf("Ошибка парсинга config.secret.yaml: %v", err)
    }
    
    // 3. Объединяем конфиги - берем пароль из секретного файла
    config.Database.Password = secretConfig.Database.Password
    
    if config.Database.Password == "" {
        log.Fatal("Database password is required in config.secret.yaml")
    }
    
    log.Println("Конфигурация успешно загружена")
    return config
}