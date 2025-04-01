package services

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/pkg/database"
)

type FermentationService struct{}

func NewFermentationService() *FermentationService {
	return &FermentationService{}
}

// CreateFermentation tworzy nową fermentację
func (s *FermentationService) CreateFermentation(userID uint, name, style, styleID, styleCategory, description string, ispindelID uint) (*models.Fermentation, error) {
	// Sprawdź czy urządzenie istnieje i należy do tego użytkownika
	var ispindel models.Ispindel
	if err := database.DB.Where("id = ? AND user_id = ? AND is_active = ?", ispindelID, userID, true).First(&ispindel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("nie znaleziono aktywnego urządzenia iSpindel o podanym ID")
		}
		return nil, err
	}

	// Utwórz nową fermentację
	id := ispindelID // Tworzymy zmienną, aby móc przekazać jej adres
	fermentation := &models.Fermentation{
		UserID:        userID,
		IspindelID:    &id,
		Name:          name,
		Style:         style,
		StyleID:       styleID,
		StyleCategory: styleCategory,
		Description:   description,
		StartedAt:     time.Now(),
		IsActive:      true,
	}

	if err := database.DB.Create(fermentation).Error; err != nil {
		return nil, err
	}

	return fermentation, nil
}

// GetFermentationsByUserID pobiera wszystkie fermentacje użytkownika
func (s *FermentationService) GetFermentationsByUserID(userID uint) ([]models.Fermentation, error) {
	var fermentations []models.Fermentation
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&fermentations).Error; err != nil {
		return nil, err
	}
	return fermentations, nil
}

// GetActiveFermentationsByUserID pobiera aktywne fermentacje użytkownika
func (s *FermentationService) GetActiveFermentationsByUserID(userID uint) ([]models.Fermentation, error) {
	var fermentations []models.Fermentation
	if err := database.DB.Where("user_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&fermentations).Error; err != nil {
		return nil, err
	}
	return fermentations, nil
}

// GetFermentationByID pobiera fermentację po ID
func (s *FermentationService) GetFermentationByID(fermentationID, userID uint) (*models.Fermentation, error) {
	var fermentation models.Fermentation
	if err := database.DB.Where("id = ? AND user_id = ?", fermentationID, userID).First(&fermentation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("nie znaleziono fermentacji o podanym ID")
		}
		return nil, err
	}

	// Załaduj informacje o urządzeniu
	if err := database.DB.Model(&fermentation).Association("Ispindel").Find(&fermentation.Ispindel); err != nil {
		return nil, err
	}

	return &fermentation, nil
}

// EndFermentation kończy fermentację
func (s *FermentationService) EndFermentation(fermentationID, userID uint) error {
	return s.EndFermentationWithComment(fermentationID, userID, "")
}

// EndFermentationWithComment kończy fermentację i dodaje komentarz do pola opisu
func (s *FermentationService) EndFermentationWithComment(fermentationID, userID uint, comment string) error {
	fermentation, err := s.GetFermentationByID(fermentationID, userID)
	if err != nil {
		return err
	}

	if !fermentation.IsActive {
		return errors.New("fermentacja jest już zakończona")
	}

	now := time.Now()
	fermentation.EndedAt = &now
	fermentation.IsActive = false

	if comment != "" {
		if fermentation.Description != "" {
			fermentation.Description = fermentation.Description + "\n\n" + comment
		} else {
			fermentation.Description = comment
		}
	}

	return database.DB.Save(fermentation).Error
}

// GetActiveIspindelsForUser zwraca listę aktywnych urządzeń iSpindel dla użytkownika
// które nie są już używane w aktywnych fermentacjach
func (s *FermentationService) GetActiveIspindelsForUser(userID uint) ([]models.Ispindel, error) {
	var ispindels []models.Ispindel
	if err := database.DB.Where("user_id = ? AND is_active = ?", userID, true).Find(&ispindels).Error; err != nil {
		return nil, err
	}

	// Jeśli nie znaleziono urządzeń, zwróć pustą listę
	if len(ispindels) == 0 {
		return ispindels, nil
	}

	// Pobierz ID urządzeń używanych w aktywnych fermentacjach
	var usedIspindelIDs []uint
	if err := database.DB.Model(&models.Fermentation{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Pluck("ispindel_id", &usedIspindelIDs).Error; err != nil {
		return nil, err
	}

	// Jeśli nie ma aktywnych fermentacji, zwróć wszystkie aktywne urządzenia
	if len(usedIspindelIDs) == 0 {
		return ispindels, nil
	}

	// Filtruj urządzenia, które są już używane
	var availableIspindels []models.Ispindel
	for _, ispindel := range ispindels {
		isUsed := false
		for _, usedID := range usedIspindelIDs {
			if ispindel.ID == usedID {
				isUsed = true
				break
			}
		}
		if !isUsed {
			availableIspindels = append(availableIspindels, ispindel)
		}
	}

	return availableIspindels, nil
}

// DeleteFermentation usuwa fermentację
func (s *FermentationService) DeleteFermentation(fermentationID, userID uint) error {
	// Najpierw sprawdź, czy fermentacja istnieje i należy do użytkownika
	fermentation, err := s.GetFermentationByID(fermentationID, userID)
	if err != nil {
		return err
	}

	// Sprawdź, czy fermentacja jest już zakończona lub nie ma pomiarów
	if fermentation.IsActive {
		// Pobierz pomiary, aby sprawdzić, czy fermentacja ma jakieś dane
		measurements, err := s.GetAllMeasurements(fermentationID)
		if err != nil {
			return err
		}

		// Jeśli fermentacja ma pomiary, nie można jej usunąć bez zakończenia
		if len(measurements) > 0 {
			return errors.New("nie można usunąć aktywnej fermentacji z pomiarami - najpierw ją zakończ")
		}
	}

	// Usuń fermentację
	return database.DB.Delete(&models.Fermentation{}, fermentationID).Error
}

// GetFermentation pobiera szczegóły fermentacji po ID
func (s *FermentationService) GetFermentation(id uint, userID uint) (*models.Fermentation, error) {
	var fermentation models.Fermentation
	err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&fermentation).Error
	if err != nil {
		return nil, err
	}
	return &fermentation, nil
}

// GetAllMeasurements pobiera wszystkie pomiary dla danej fermentacji
func (s *FermentationService) GetAllMeasurements(fermentationID uint) ([]models.Measurement, error) {
	// Najpierw pobierz fermentację, aby uzyskać ispindel_id i zakres dat
	var fermentation models.Fermentation
	if err := database.DB.First(&fermentation, fermentationID).Error; err != nil {
		return nil, err
	}

	// Jeśli IspindelID jest NULL (urządzenie zostało usunięte), zwróć pustą listę pomiarów
	if fermentation.IspindelID == nil {
		return []models.Measurement{}, nil
	}

	// Przygotuj zapytanie bazowe
	query := database.DB.Where("ispindel_id = ?", fermentation.IspindelID)

	// Dodaj warunek na zakres dat
	query = query.Where("timestamp >= ?", fermentation.StartedAt)
	if !fermentation.IsActive && fermentation.EndedAt != nil {
		query = query.Where("timestamp <= ?", fermentation.EndedAt)
	}

	// Pobierz pomiary - sortowanie malejąco po timestamp (od najnowszych)
	var measurements []models.Measurement
	err := query.Order("timestamp DESC").Find(&measurements).Error
	if err != nil {
		return nil, err
	}

	return measurements, nil
}

// GetMeasurementsLast12Hours pobiera pomiary z ostatnich 12 godzin dla danej fermentacji
func (s *FermentationService) GetMeasurementsLast12Hours(fermentationID uint) ([]models.Measurement, error) {
	// Najpierw pobierz fermentację, aby uzyskać ispindel_id i zakres dat
	var fermentation models.Fermentation
	if err := database.DB.First(&fermentation, fermentationID).Error; err != nil {
		return nil, err
	}

	// Jeśli IspindelID jest NULL (urządzenie zostało usunięte), zwróć pustą listę pomiarów
	if fermentation.IspindelID == nil {
		return []models.Measurement{}, nil
	}

	// Przygotuj zapytanie bazowe
	query := database.DB.Where("ispindel_id = ?", fermentation.IspindelID)

	// Dodaj warunek na zakres dat - ostatnie 12 godzin
	twelveHoursAgo := time.Now().Add(-12 * time.Hour)
	query = query.Where("timestamp >= ?", twelveHoursAgo)

	// Pobierz pomiary
	var measurements []models.Measurement
	err := query.Order("timestamp ASC").Find(&measurements).Error
	if err != nil {
		return nil, err
	}

	return measurements, nil
}

// GetHourlyMeasurementsLast12Hours pobiera pomiary z ostatnich 12 godzin dla danej fermentacji, jeden pomiar na godzinę
func (s *FermentationService) GetHourlyMeasurementsLast12Hours(fermentationID uint) ([]models.Measurement, error) {
	// Najpierw pobierz fermentację, aby uzyskać ispindel_id
	var fermentation models.Fermentation
	if err := database.DB.First(&fermentation, fermentationID).Error; err != nil {
		return nil, err
	}

	// Jeśli IspindelID jest NULL (urządzenie zostało usunięte), zwróć pustą listę pomiarów
	if fermentation.IspindelID == nil {
		return []models.Measurement{}, nil
	}

	// Przygotuj zapytanie bazowe
	query := database.DB.Model(&models.Measurement{}).
		Where("ispindel_id = ?", fermentation.IspindelID)

	// Dodaj warunek na zakres dat - ostatnie 12 godzin
	twelveHoursAgo := time.Now().Add(-12 * time.Hour)
	query = query.Where("timestamp >= ?", twelveHoursAgo)

	// Grupuj po godzinie używając składni MySQL
	query = query.Select("MIN(id) as id").
		Group("DATE_FORMAT(timestamp, '%Y-%m-%d %H:00:00')")

	// Pobierz ID pomiarów
	var measurementIDs []uint
	if err := query.Pluck("id", &measurementIDs).Error; err != nil {
		return nil, err
	}

	// Pobierz pełne rekordy pomiarów
	var measurements []models.Measurement
	if len(measurementIDs) > 0 {
		err := database.DB.Where("id IN ?", measurementIDs).
			Order("timestamp ASC").
			Find(&measurements).Error
		if err != nil {
			return nil, err
		}
	}

	return measurements, nil
}

// FermentationDuration reprezentuje czas trwania fermentacji
type FermentationDuration struct {
	Days    int
	Hours   int
	Minutes int
}

// GetFermentationDuration oblicza czas trwania fermentacji
func (s *FermentationService) GetFermentationDuration(fermentation *models.Fermentation) FermentationDuration {
	var endTime time.Time
	if fermentation.IsActive {
		endTime = time.Now()
	} else if fermentation.EndedAt != nil {
		endTime = *fermentation.EndedAt
	} else {
		endTime = time.Now()
	}

	duration := endTime.Sub(fermentation.StartedAt)

	totalHours := int(duration.Hours())
	days := totalHours / 24
	hours := totalHours % 24
	minutes := int(duration.Minutes()) % 60

	return FermentationDuration{
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
	}
}

// GetFermentationDurationString zwraca sformatowany string z czasem trwania fermentacji
func (s *FermentationService) GetFermentationDurationString(fermentation *models.Fermentation) string {
	duration := s.GetFermentationDuration(fermentation)

	if duration.Days > 0 {
		if duration.Hours > 0 {
			return fmt.Sprintf("%d dni %d godz", duration.Days, duration.Hours)
		}
		return fmt.Sprintf("%d dni", duration.Days)
	}

	if duration.Hours > 0 {
		if duration.Minutes > 0 {
			return fmt.Sprintf("%d godz %d min", duration.Hours, duration.Minutes)
		}
		return fmt.Sprintf("%d godz", duration.Hours)
	}

	return fmt.Sprintf("%d min", duration.Minutes)
}

// GetAllMeasurementsChronological pobiera wszystkie pomiary dla fermentacji w kolejności chronologicznej
func (s *FermentationService) GetAllMeasurementsChronological(fermentationID uint) ([]models.Measurement, error) {
	var measurements []models.Measurement
	if err := database.DB.Where("fermentation_id = ?", fermentationID).Order("timestamp ASC").Find(&measurements).Error; err != nil {
		return nil, err
	}
	return measurements, nil
}

// SortMeasurementsChronologically sortuje pomiary chronologicznie (od najstarszych do najnowszych)
func SortMeasurementsChronologically(measurements []models.Measurement) {
	sort.Slice(measurements, func(i, j int) bool {
		return measurements[i].Timestamp.Before(measurements[j].Timestamp)
	})
}

// GetInitialMeasurements pobiera średnie wartości z pierwszych 3 pomiarów fermentacji
func (s *FermentationService) GetInitialMeasurements(fermentationID uint) (float64, float64, error) {
	// Najpierw pobierz fermentację, aby uzyskać ispindel_id i datę rozpoczęcia
	var fermentation models.Fermentation
	if err := database.DB.First(&fermentation, fermentationID).Error; err != nil {
		return 0, 0, err
	}

	// Jeśli IspindelID jest NULL (urządzenie zostało usunięte), zwróć błąd
	if fermentation.IspindelID == nil {
		return 0, 0, errors.New("urządzenie pomiarowe zostało usunięte")
	}

	// Pobierz pierwsze 3 pomiary posortowane rosnąco po timestamp
	var measurements []models.Measurement
	err := database.DB.Where("ispindel_id = ? AND timestamp >= ?", fermentation.IspindelID, fermentation.StartedAt).
		Order("timestamp ASC").
		Limit(3).
		Find(&measurements).Error
	if err != nil {
		return 0, 0, err
	}

	if len(measurements) == 0 {
		return 0, 0, errors.New("brak pomiarów")
	}

	// Oblicz średnie wartości
	var sumGravity, sumTemperature float64
	for _, m := range measurements {
		sumGravity += m.Gravity
		sumTemperature += m.Temperature
	}

	avgGravity := sumGravity / float64(len(measurements))
	avgTemperature := sumTemperature / float64(len(measurements))

	return avgGravity, avgTemperature, nil
}

// GetActiveGermentationsByIspindelID zwraca listę aktywnych fermentacji dla danego urządzenia iSpindel
func (s *FermentationService) GetActiveGermentationsByIspindelID(ispindelID uint) ([]models.Fermentation, error) {
	var fermentations []models.Fermentation
	id := ispindelID // Tworzymy zmienną, aby móc przekazać jej adres
	if err := database.DB.Where("ispindel_id = ? AND is_active = true", &id).Find(&fermentations).Error; err != nil {
		return nil, err
	}
	return fermentations, nil
}

// GetFermentationCount zwraca całkowitą liczbę fermentacji
func (s *FermentationService) GetFermentationCount() (int64, error) {
	var count int64
	if err := database.DB.Model(&models.Fermentation{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetAllFermentations zwraca wszystkie fermentacje w systemie
func (s *FermentationService) GetAllFermentations() ([]map[string]interface{}, error) {
	var fermentations []models.Fermentation
	if err := database.DB.Order("started_at desc").Find(&fermentations).Error; err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, f := range fermentations {
		// Pobierz dane użytkownika, aby uzyskać jego nazwę
		var user models.User
		if err := database.DB.First(&user, f.UserID).Error; err == nil {
			results = append(results, map[string]interface{}{
				"ID":        f.ID,
				"Name":      f.Name,
				"UserID":    f.UserID,
				"UserName":  user.Name,
				"Style":     f.Style,
				"StyleID":   f.StyleID,
				"StartedAt": f.StartedAt,
				"EndedAt":   f.EndedAt,
				"IsActive":  f.IsActive,
			})
		} else {
			// Jeśli nie udało się znaleźć użytkownika, dodaj fermentację bez nazwy użytkownika
			results = append(results, map[string]interface{}{
				"ID":        f.ID,
				"Name":      f.Name,
				"UserID":    f.UserID,
				"UserName":  "Nieznany",
				"Style":     f.Style,
				"StyleID":   f.StyleID,
				"StartedAt": f.StartedAt,
				"EndedAt":   f.EndedAt,
				"IsActive":  f.IsActive,
			})
		}
	}

	return results, nil
}
