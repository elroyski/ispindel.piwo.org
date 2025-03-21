package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/pkg/database"
)

type IspindelService struct{}

func NewIspindelService() *IspindelService {
	return &IspindelService{}
}

// Generuj unikalny klucz API
func generateAPIKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateIspindel tworzy nowe urządzenie iSpindel dla użytkownika
func (s *IspindelService) CreateIspindel(userID uint, name, description string) (*models.Ispindel, error) {
	// Sprawdź czy użytkownik ma już 4 urządzenia
	var count int64
	if err := database.DB.Model(&models.Ispindel{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return nil, err
	}

	if count >= 4 {
		return nil, errors.New("osiągnięto maksymalną liczbę urządzeń (4)")
	}

	// Generuj unikalny klucz API
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// Utwórz nowe urządzenie
	ispindel := &models.Ispindel{
		UserID:      userID,
		Name:        name,
		Description: description,
		APIKey:      apiKey,
		IsActive:    true,
	}

	if err := database.DB.Create(ispindel).Error; err != nil {
		return nil, err
	}

	return ispindel, nil
}

// GetIspindelsByUserID pobiera wszystkie urządzenia użytkownika
func (s *IspindelService) GetIspindelsByUserID(userID uint) ([]models.Ispindel, error) {
	var ispindels []models.Ispindel
	if err := database.DB.Where("user_id = ?", userID).Find(&ispindels).Error; err != nil {
		return nil, err
	}
	return ispindels, nil
}

// GetIspindelByID pobiera urządzenie po ID
func (s *IspindelService) GetIspindelByID(ispindelID, userID uint) (*models.Ispindel, error) {
	var ispindel models.Ispindel
	if err := database.DB.Where("id = ? AND user_id = ?", ispindelID, userID).First(&ispindel).Error; err != nil {
		return nil, err
	}
	return &ispindel, nil
}

// UpdateIspindel aktualizuje dane urządzenia
func (s *IspindelService) UpdateIspindel(ispindelID, userID uint, name, description string, isActive bool) (*models.Ispindel, error) {
	ispindel, err := s.GetIspindelByID(ispindelID, userID)
	if err != nil {
		return nil, err
	}

	ispindel.Name = name
	ispindel.Description = description
	ispindel.IsActive = isActive

	if err := database.DB.Save(ispindel).Error; err != nil {
		return nil, err
	}

	return ispindel, nil
}

// DeleteIspindel usuwa urządzenie
func (s *IspindelService) DeleteIspindel(ispindelID, userID uint) error {
	result := database.DB.Where("id = ? AND user_id = ?", ispindelID, userID).Delete(&models.Ispindel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("nie znaleziono urządzenia")
	}
	return nil
}

// RegenerateAPIKey generuje nowy klucz API dla urządzenia
func (s *IspindelService) RegenerateAPIKey(ispindelID, userID uint) (string, error) {
	ispindel, err := s.GetIspindelByID(ispindelID, userID)
	if err != nil {
		return "", err
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	ispindel.APIKey = apiKey
	if err := database.DB.Save(ispindel).Error; err != nil {
		return "", err
	}

	return apiKey, nil
}

// FindIspindelByAPIKey znajduje urządzenie na podstawie klucza API
func (s *IspindelService) FindIspindelByAPIKey(apiKey string) (*models.Ispindel, error) {
	var ispindel models.Ispindel
	if err := database.DB.Where("api_key = ? AND is_active = ?", apiKey, true).First(&ispindel).Error; err != nil {
		return nil, err
	}
	return &ispindel, nil
}

// SaveMeasurement zapisuje pomiar z urządzenia iSpindel
func (s *IspindelService) SaveMeasurement(ispindelID uint, data map[string]interface{}) (*models.Measurement, error) {
	// Pobierz urządzenie aby upewnić się, że istnieje
	var ispindel models.Ispindel
	if err := database.DB.First(&ispindel, ispindelID).Error; err != nil {
		return nil, errors.New("nie znaleziono urządzenia")
	}

	// Aktualizuj pole LastSeen
	now := time.Now()
	ispindel.LastSeen = now
	if err := database.DB.Save(&ispindel).Error; err != nil {
		return nil, err
	}

	// Przygotuj pomiar do zapisania
	measurement := &models.Measurement{
		IspindelID: ispindelID,
		ReceivedAt: now,
	}

	// Pobierz dane z mapy
	if val, ok := data["ID"].(float64); ok {
		measurement.DeviceID = uint(val)
	}
	
	if val, ok := data["name"].(string); ok {
		measurement.Name = val
	}

	if val, ok := data["angle"].(float64); ok {
		measurement.Angle = val
	}

	if val, ok := data["temperature"].(float64); ok {
		measurement.Temperature = val
	}

	if val, ok := data["temp_units"].(string); ok {
		measurement.TempUnits = val
	}

	if val, ok := data["battery"].(float64); ok {
		measurement.Battery = val
	}

	if val, ok := data["gravity"].(float64); ok {
		measurement.Gravity = val
	}

	if val, ok := data["interval"].(float64); ok {
		measurement.Interval = int(val)
	}

	if val, ok := data["RSSI"].(float64); ok {
		measurement.RSSI = int(val)
	}

	// Przetwarzanie timestampu
	if val, ok := data["timestamp"].(string); ok {
		t, err := time.Parse("2006-01-02 15:04:05", val)
		if err == nil {
			measurement.Timestamp = t
		} else {
			measurement.Timestamp = now
		}
	} else {
		measurement.Timestamp = now
	}

	// Zapisz pomiar
	if err := database.DB.Create(measurement).Error; err != nil {
		return nil, err
	}

	return measurement, nil
}

// GetLatestMeasurements pobiera najnowsze pomiary dla danego urządzenia
func (s *IspindelService) GetLatestMeasurements(ispindelID uint, limit int) ([]models.Measurement, error) {
	var measurements []models.Measurement
	
	if limit <= 0 {
		limit = 10 // Domyślny limit
	}

	if err := database.DB.Where("ispindel_id = ?", ispindelID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&measurements).Error; err != nil {
		return nil, err
	}

	return measurements, nil
} 