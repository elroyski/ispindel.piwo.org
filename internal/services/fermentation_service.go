package services

import (
	"errors"
	"fmt"
	"time"

	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/pkg/database"
	"gorm.io/gorm"
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
	fermentation := &models.Fermentation{
		UserID:        userID,
		IspindelID:    ispindelID,
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
	
	// Sprawdź, czy fermentacja jest już zakończona
	if fermentation.IsActive {
		return errors.New("nie można usunąć aktywnej fermentacji - najpierw ją zakończ")
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

	// Przygotuj zapytanie bazowe
	query := database.DB.Where("ispindel_id = ?", fermentation.IspindelID)

	// Dodaj warunek na zakres dat
	query = query.Where("timestamp >= ?", fermentation.StartedAt)
	if !fermentation.IsActive && fermentation.EndedAt != nil {
		query = query.Where("timestamp <= ?", fermentation.EndedAt)
	}

	// Pobierz pomiary - sortowanie rosnąco po timestamp
	var measurements []models.Measurement
	err := query.Order("timestamp ASC").Find(&measurements).Error
	if err != nil {
		return nil, err
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