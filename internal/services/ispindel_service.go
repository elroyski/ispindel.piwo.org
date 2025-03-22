package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
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

	// Sprawdź i zaktualizuj stan aktywności każdego urządzenia
	for _, ispindel := range ispindels {
		if err := s.checkAndUpdateDeviceActivity(ispindel.ID); err != nil {
			log.Printf("Błąd podczas aktualizacji stanu aktywności urządzenia %d: %v", ispindel.ID, err)
		}
	}

	// Pobierz zaktualizowaną listę urządzeń
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
func (s *IspindelService) UpdateIspindel(ispindel *models.Ispindel) error {
	// Aktualizujemy pola name, description i is_active
	return database.DB.Model(ispindel).Updates(map[string]interface{}{
		"name":        ispindel.Name,
		"description": ispindel.Description,
		"is_active":   ispindel.IsActive,
	}).Error
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
	if err := database.DB.Where("api_key = ?", apiKey).First(&ispindel).Error; err != nil {
		return nil, err
	}
	return &ispindel, nil
}

// IsIspindelActive sprawdza, czy urządzenie jest aktywne
func (s *IspindelService) IsIspindelActive(ispindel *models.Ispindel) bool {
	return ispindel.IsActive
}

// shouldSaveMeasurement sprawdza czy powinniśmy zapisać nowy pomiar
func (s *IspindelService) shouldSaveMeasurement(ispindelID uint) (bool, error) {
	var lastMeasurement models.Measurement
	result := database.DB.Where("ispindel_id = ?", ispindelID).Order("timestamp desc").First(&lastMeasurement)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Jeśli nie ma wcześniejszych pomiarów, pozwól na zapis
			return true, nil
		}
		return false, result.Error
	}

	// Pobierz minimalny interwał z zmiennej środowiskowej
	minInterval := 900 // domyślna wartość 900 sekund (15 minut)
	if envInterval := os.Getenv("ISPINDEL_MIN_INTERVAL"); envInterval != "" {
		if interval, err := strconv.Atoi(envInterval); err == nil {
			minInterval = interval
		} else {
			log.Printf("Błąd podczas parsowania ISPINDEL_MIN_INTERVAL: %v, używam wartości domyślnej 900", err)
		}
	}

	// Sprawdź czy minął wymagany czas od ostatniego pomiaru
	timeSinceLastMeasurement := time.Since(lastMeasurement.Timestamp)
	return timeSinceLastMeasurement.Seconds() >= float64(minInterval), nil
}

// SaveMeasurement zapisuje pomiar z urządzenia iSpindel
func (s *IspindelService) SaveMeasurement(ispindelID uint, data map[string]interface{}) (*models.Measurement, error) {
	// Sprawdź czy powinniśmy zapisać nowy pomiar
	shouldSave, err := s.shouldSaveMeasurement(ispindelID)
	if err != nil {
		return nil, fmt.Errorf("błąd podczas sprawdzania czasu ostatniego pomiaru: %v", err)
	}

	if !shouldSave {
		return nil, fmt.Errorf("za częste pomiary - minimalny odstęp między pomiarami to 900 sekund (15 minut)")
	}

	// Kontynuuj normalny proces zapisu pomiaru...
	measurement := &models.Measurement{
		IspindelID: ispindelID,
		Timestamp:  time.Now(),
	}

	// Mapowanie pól z danych JSON na strukturę Measurement
	if name, ok := data["name"].(string); ok {
		measurement.Name = name
	}

	var deviceIDStr string
	if deviceID, ok := data["ID"].(float64); ok {
		deviceIDStr = fmt.Sprintf("%.0f", deviceID)
		measurement.DeviceID = deviceIDStr
	}

	if angle, ok := data["angle"].(float64); ok {
		measurement.Angle = angle
	}
	if temp, ok := data["temperature"].(float64); ok {
		measurement.Temperature = temp
	}
	if battery, ok := data["battery"].(float64); ok {
		measurement.Battery = battery
	}
	if gravity, ok := data["gravity"].(float64); ok {
		measurement.Gravity = gravity
	}
	if interval, ok := data["interval"].(float64); ok {
		measurement.Interval = int(interval)
	}
	if rssi, ok := data["RSSI"].(float64); ok {
		measurement.RSSI = int(rssi)
	}

	// Zapisz pomiar w bazie danych
	if err := database.DB.Create(measurement).Error; err != nil {
		return nil, fmt.Errorf("błąd podczas zapisywania pomiaru: %v", err)
	}

	// Aktualizuj informacje o urządzeniu
	updates := map[string]interface{}{
		"last_seen": time.Now(),
	}

	// Dodaj DeviceID do aktualizacji jeśli jest dostępne
	if deviceIDStr != "" {
		updates["device_id"] = deviceIDStr
	}

	// Dodaj Name do aktualizacji jeśli jest dostępne
	if name, ok := data["name"].(string); ok && name != "" {
		// Usuń ID z nazwy jeśli występuje (format: "ID/name")
		if parts := strings.Split(name, "/"); len(parts) > 1 {
			updates["name"] = parts[1]
		} else {
			updates["name"] = name
		}
	}

	if err := database.DB.Model(&models.Ispindel{}).Where("id = ?", ispindelID).Updates(updates).Error; err != nil {
		log.Printf("Błąd podczas aktualizacji informacji o urządzeniu: %v", err)
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

// GetMeasurementsForIspindelInRange pobiera pomiary dla danego urządzenia w określonym zakresie czasowym
func (s *IspindelService) GetMeasurementsForIspindelInRange(ispindelID uint, startTime, endTime time.Time, limit int) ([]models.Measurement, error) {
	var measurements []models.Measurement

	if limit <= 0 {
		limit = 100 // Domyślny limit
	}

	if err := database.DB.Where("ispindel_id = ? AND timestamp BETWEEN ? AND ?",
		ispindelID, startTime, endTime).
		Order("timestamp DESC").
		Limit(limit).
		Find(&measurements).Error; err != nil {
		return nil, err
	}

	return measurements, nil
}

// checkAndUpdateDeviceActivity sprawdza i aktualizuje stan aktywności urządzenia
func (s *IspindelService) checkAndUpdateDeviceActivity(ispindelID uint) error {
	var ispindel models.Ispindel
	if err := database.DB.First(&ispindel, ispindelID).Error; err != nil {
		return err
	}

	// Pobierz maksymalny czas nieaktywności z zmiennej środowiskowej
	inactivityTimeout := 6 // domyślna wartość 6 godzin
	if envTimeout := os.Getenv("ISPINDEL_INACTIVITY_TIMEOUT"); envTimeout != "" {
		if timeout, err := strconv.Atoi(envTimeout); err == nil {
			inactivityTimeout = timeout
		} else {
			log.Printf("Błąd podczas parsowania ISPINDEL_INACTIVITY_TIMEOUT: %v, używam wartości domyślnej 6", err)
		}
	}

	// Oblicz czas nieaktywności
	inactivityDuration := time.Since(ispindel.LastSeen)
	isInactive := inactivityDuration.Hours() >= float64(inactivityTimeout)

	// Logujemy informację o długiej nieaktywności
	if isInactive {
		log.Printf("Urządzenie iSpindel %d nie wysłało danych przez %v godzin", ispindelID, inactivityTimeout)
	}

	return nil
}
