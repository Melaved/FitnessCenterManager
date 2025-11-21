package models

import (
	"database/sql"
	"time"
)

type ClientEnriched struct {
	ID               int            `db:"id_клиента"`
	FIO              string         `db:"ФИО"`
	Phone            string         `db:"Номер_телефона"`
	BirthDate        time.Time      `db:"Дата_рождения"`
	RegisterDate     time.Time      `db:"Дата_регистрации"`
	MedicalData      sql.NullString `db:"Медицинские_данные"`
	Age              int            `db:"age"`                 // вычисляемая из view
	SubscriptionsCnt int            `db:"subscriptions_count"` // вычисляемая из view
	ActiveStatus     string         `db:"active_status"`       // подстановочная из view
}
