// models/trainings.go

package models

import "time"

// Групповые (view)
type GroupTrainingView struct {
    ID                int       `db:"id"`
    Title             string    `db:"title"`
    Description       string    `db:"description"`
    Level             string    `db:"level"`
    Max               int       `db:"max"`
    StartsAt          time.Time `db:"starts_at"`
    EndsAt            time.Time `db:"ends_at"`
    TrainerID         int       `db:"trainer_id"`
    TrainerName       string    `db:"trainer_name"`       // lookup
    ZoneID            int       `db:"zone_id"`
    ZoneName          string    `db:"zone_name"`          // lookup

    DurationMinutes   int       `db:"duration_minutes"`   // вычисляемая
    EnrolledCount     int       `db:"enrolled_count"`     // вычисляемая
    SlotsLeft         int       `db:"slots_left"`         // вычисляемая
    IsUpcoming        bool      `db:"is_upcoming"`        // вычисляемая
    IsRecent          bool      `db:"is_recent"`          // вычисляемая
    StatusTime        string    `db:"status_time"`        // подстановочная
    CapacityUsagePct  int       `db:"capacity_usage_pct"` // вычисляемая
}

// Персональные (view)
type PersonalTrainingView struct {
    ID              int       `db:"id"`
    SubscriptionID  int       `db:"subscription_id"`
    ClientFIO       string    `db:"client_fio"`      // lookup
    TrainerID       int       `db:"trainer_id"`
    TrainerFIO      string    `db:"trainer_fio"`     // lookup
    StartsAt        time.Time `db:"starts_at"`
    EndsAt          time.Time `db:"ends_at"`
    Status          string    `db:"status"`
    StatusBadge     string    `db:"status_badge"`    // подстановочная
    PriceEffective  float64   `db:"price_effective"` // вычисляемая
    DurationMinutes int       `db:"duration_minutes"`// вычисляемая
    IsUpcoming      bool      `db:"is_upcoming"`     // вычисляемая
    IsRecent        bool      `db:"is_recent"`       // вычисляемая
}
