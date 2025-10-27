package models

import (
    "database/sql"
    "time"
)

type Client struct {
    ID           int            `json:"id_клиента"`
    FIO          string         `json:"фио"`
    Phone        string         `json:"номер_телефона"`
    BirthDate    time.Time      `json:"дата_рождения"`
    RegisterDate time.Time      `json:"дата_регистрации"`
    MedicalData  sql.NullString `json:"медицинские_данные"`
}

type Tariff struct {
    ID                   int     `json:"id_тарифа"`
    Name                 string  `json:"название_тарифа"`
    Description          string  `json:"описание"`
    Price                float64 `json:"стоимость"`
    AccessTime           string  `json:"время_доступа"`
    HasGroupTrainings    bool    `json:"наличие_групповых_тренировок"`
    HasPersonalTrainings bool    `json:"наличие_персональных_тренировок"`
}

type Trainer struct {
    ID             int       `json:"id_тренера"`
    FIO            string    `json:"фио"`
    Phone          string    `json:"номер_телефона"`
    Specialization string    `json:"специализация"`
    HireDate       time.Time `json:"дата_найма"`
    Experience     int       `json:"стаж_работы"`
    Active         bool      `json:"активен"`
}

type Subscription struct {
    ID          int       `json:"id_абонемента"`
    ClientID    int       `json:"id_клиента"`
    TariffID    int       `json:"id_тарифа"`
    StartDate   time.Time `json:"дата_начала"`
    EndDate     time.Time `json:"дата_окончания"`
    Status      string    `json:"статус"`
    Price       float64   `json:"цена"`
    ClientName  string    `json:"фио_клиента"`  // Для JOIN запросов
    TariffName  string    `json:"название_тарифа"` // Для JOIN запросов
}

type Zone struct {
    ID          int            `json:"id_зоны"`
    Name        string         `json:"название"`
    Description string         `json:"описание"`
    Capacity    int            `json:"вместимость"`
    PhotoPath   sql.NullString `json:"фото_path"`
    Status      string         `json:"статус"`
}

type Equipment struct {
    ID              int       `json:"id_оборудования"`
    ZoneID          int       `json:"id_зоны"`
    Name            string    `json:"название"`
    PurchaseDate    time.Time `json:"дата_покупки"`
    LastServiceDate time.Time `json:"дата_последнего_то"`
    Status          string    `json:"статус"`
    PhotoPath       string    `json:"фото_path"`
    ZoneName        string    `json:"название_зоны"` // Для JOIN запросов
}

type PersonalTraining struct {
    ID             int       `json:"id_персональной_тренировки"`
    SubscriptionID int       `json:"id_абонемента"`
    TrainerID      int       `json:"id_тренера"`
    StartTime      time.Time `json:"время_начала"`
    EndTime        time.Time `json:"время_окончания"`
    Status         string    `json:"статус"`
    Price          float64   `json:"стоимость"`
    ClientName     string    `json:"фио_клиента"`  // Для JOIN запросов
    TrainerName    string    `json:"фио_тренера"`  // Для JOIN запросов
}

type GroupTraining struct {
    ID              int       `json:"id_групповой_тренировки"`
    TrainerID       int       `json:"id_тренера"`
    ZoneID          int       `json:"id_зоны"`
    Name            string    `json:"название"`
    Description     string    `json:"описание"`
    MaxParticipants int       `json:"максимум_участников"`
    StartTime       time.Time `json:"время_начала"`
    EndTime         time.Time `json:"время_окончания"`
    DifficultyLevel string    `json:"уровень_сложности"`
    PhotoPath       string    `json:"фото_path"`
    TrainerName     string    `json:"фио_тренера"` // Для JOIN запросов
    ZoneName        string    `json:"название_зоны"` // Для JOIN запросов
}

type GroupTrainingRegistration struct {
    ID              int    `json:"id_записи"`
    GroupTrainingID int    `json:"id_групповой_тренировки"`
    SubscriptionID  int    `json:"id_абонемента"`
    Status          string `json:"статус"`
    ClientName      string `json:"фио_клиента"`    // Для JOIN запросов
    TrainingName    string `json:"название_тренировки"` // Для JOIN запросов
}

type RepairRequest struct {
    ID            int       `json:"id_заявки"`
    EquipmentID   int       `json:"id_оборудования"`
    CreateDate    time.Time `json:"дата_создания"`
    ProblemDesc   string    `json:"описание_проблемы"`
    Status        string    `json:"статус"`
    Priority      string    `json:"приоритет"`
	PhotoPath       string    `json:"фото_path"`
    EquipmentName string    `json:"название_оборудования"` // Для JOIN запросов
    ZoneName      string    `json:"название_зоны"`         // Для JOIN запросов
}