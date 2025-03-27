package models

import (
	"time"

	"gorm.io/gorm"
)

// Fermentation reprezentuje proces fermentacji
type Fermentation struct {
	gorm.Model
	UserID        uint       `gorm:"index" json:"user_id"`
	IspindelID    *uint      `gorm:"index" json:"ispindel_id"`       // Powiązanie z urządzeniem
	Name          string     `gorm:"size:100;not null" json:"name"`  // Nazwa warki
	Style         string     `gorm:"size:100" json:"style"`          // Styl piwa
	StyleID       string     `gorm:"size:20" json:"style_id"`        // Identyfikator stylu (np. "1A")
	StyleCategory string     `gorm:"size:100" json:"style_category"` // Kategoria stylu
	Description   string     `gorm:"size:500" json:"description"`    // Dodatkowe informacje
	StartedAt     time.Time  `json:"started_at"`                     // Data rozpoczęcia
	EndedAt       *time.Time `json:"ended_at"`                       // Data zakończenia (opcjonalna)
	IsActive      bool       `gorm:"default:true" json:"is_active"`  // Czy fermentacja jest aktywna

	// Relacje
	Ispindel Ispindel `json:"-"`
}

// GetDurationInDays zwraca czas trwania fermentacji w dniach
func (f *Fermentation) GetDurationInDays() int {
	if f.StartedAt.IsZero() {
		return 0
	}

	endTime := time.Now()
	if f.EndedAt != nil {
		endTime = *f.EndedAt
	}

	duration := endTime.Sub(f.StartedAt)
	return int(duration.Hours() / 24)
}

// IsCompleted sprawdza, czy fermentacja została zakończona
func (f *Fermentation) IsCompleted() bool {
	return f.EndedAt != nil
}
