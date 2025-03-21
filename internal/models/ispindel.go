package models

import (
	"time"

	"gorm.io/gorm"
)

// Ispindel reprezentuje urządzenie iSpindel powiązane z użytkownikiem
type Ispindel struct {
	gorm.Model
	UserID      uint   `gorm:"index" json:"user_id"`
	Name        string `gorm:"size:100;not null" json:"name"`
	DeviceID    string `gorm:"size:100" json:"device_id"` // Może być pusty, używany do identyfikacji urządzenia
	APIKey      string `gorm:"size:100;uniqueIndex" json:"api_key"`
	Description string `gorm:"size:255" json:"description"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
	LastSeen    time.Time `json:"last_seen"`
	
	// Relacja z pomiarami
	Measurements []Measurement `json:"-"`
}

// Measurement reprezentuje pojedynczy pomiar wysłany przez iSpindel
type Measurement struct {
	gorm.Model
	IspindelID   uint      `gorm:"index" json:"ispindel_id"`
	DeviceID     uint      `json:"device_id"`
	Name         string    `gorm:"size:100" json:"name"`
	Angle        float64   `json:"angle"`
	Temperature  float64   `json:"temperature"`
	TempUnits    string    `gorm:"size:10" json:"temp_units"`
	Battery      float64   `json:"battery"`
	Gravity      float64   `json:"gravity"`
	Interval     int       `json:"interval"`
	RSSI         int       `json:"rssi"`
	Timestamp    time.Time `json:"timestamp"`
	ReceivedAt   time.Time `json:"received_at"`
} 