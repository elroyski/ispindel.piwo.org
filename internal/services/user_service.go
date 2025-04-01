package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/pkg/auth"
	"ispindel.piwo.org/pkg/database"
	"ispindel.piwo.org/pkg/mailer"
	"ispindel.piwo.org/pkg/utils"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: database.DB,
	}
}

func (s *UserService) Register(name, email, password, ip string) error {
	// Sprawdź czy użytkownik już istnieje
	var existingUser models.User
	if err := s.db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return errors.New("użytkownik o podanym adresie email już istnieje")
	}

	// Hashuj hasło
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	// Generuj token aktywacyjny
	activationToken := utils.GenerateActivationToken()
	activationExpires := time.Now().Add(24 * time.Hour)

	// Utwórz nowego użytkownika
	user := models.User{
		Name:              name,
		Email:             email,
		Password:          hashedPassword,
		RegistrationIP:    ip,
		IsActive:          false, // Użytkownik nie jest aktywny do czasu potwierdzenia
		ActivationToken:   activationToken,
		ActivationExpires: activationExpires,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return err
	}

	// Wyślij email z linkiem aktywacyjnym
	err = mailer.SendActivationEmail(email, name, activationToken)
	if err != nil {
		// Logujemy błąd, ale nie przerywamy rejestracji
		log.Printf("Błąd wysyłania e-maila aktywacyjnego do %s: %v", email, err)
		return errors.New("konto zostało utworzone, ale nie mogliśmy wysłać e-maila aktywacyjnego. Możesz poprosić o ponowne wysłanie e-maila z linkiem aktywacyjnym")
	}

	return nil
}

func (s *UserService) ActivateAccount(token string) error {
	var user models.User
	if err := s.db.Where("activation_token = ?", token).First(&user).Error; err != nil {
		return errors.New("nieprawidłowy token aktywacyjny")
	}

	// Sprawdź czy token nie wygasł
	if user.ActivationExpires.Before(time.Now()) {
		return errors.New("token aktywacyjny wygasł")
	}

	// Sprawdź czy konto nie zostało już aktywowane
	if user.ActivationCompleted {
		return errors.New("konto zostało już aktywowane")
	}

	// Aktywuj konto
	user.IsActive = true
	user.ActivationCompleted = true
	user.ActivationToken = "" // Wyczyść token

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func (s *UserService) Login(email, password, ip string) (string, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return "", errors.New("nieprawidłowy email lub hasło")
	}

	// Sprawdź czy konto jest zablokowane lub nieaktywne
	if !user.IsActive {
		return "", errors.New("konto nie zostało aktywowane - sprawdź swój e-mail")
	}

	if user.LockedUntil.After(time.Now()) {
		return "", errors.New("konto jest tymczasowo zablokowane")
	}

	// Sprawdź hasło
	if !auth.CheckPassword(password, user.Password) {
		// Zwiększ licznik nieudanych prób
		user.FailedLogins++
		if user.FailedLogins >= 5 {
			user.LockedUntil = time.Now().Add(15 * time.Minute)
		}
		s.db.Save(&user)
		return "", errors.New("nieprawidłowy email lub hasło")
	}

	// Resetuj licznik nieudanych prób
	user.FailedLogins = 0
	user.LastLoginAt = time.Now()
	s.db.Save(&user)

	// Generuj token JWT
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) GetUser(userID int64) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ResendActivationEmail(email string) error {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return errors.New("nie znaleziono użytkownika o podanym adresie email")
	}

	// Sprawdź czy konto nie zostało już aktywowane
	if user.ActivationCompleted || user.IsActive {
		return errors.New("konto zostało już aktywowane")
	}

	// Wygeneruj nowy token i ustaw czas wygaśnięcia
	activationToken := utils.GenerateActivationToken()
	activationExpires := time.Now().Add(24 * time.Hour)

	user.ActivationToken = activationToken
	user.ActivationExpires = activationExpires

	if err := s.db.Save(&user).Error; err != nil {
		return err
	}

	// Wyślij email z nowym linkiem aktywacyjnym
	return mailer.SendActivationEmail(user.Email, user.Name, activationToken)
}

func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) UpdateLastLogin(userID uint) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error
}

func (s *UserService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	// Pobierz użytkownika
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	// Sprawdź czy aktualne hasło jest poprawne
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("aktualne hasło jest niepoprawne")
	}

	// Haszuj nowe hasło
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("nie udało się zahaszować nowego hasła")
	}

	// Aktualizuj hasło
	return s.db.Model(&user).Update("password", string(hashedPassword)).Error
}

// DeleteUser usuwa konto użytkownika wraz ze wszystkimi powiązanymi danymi
func (s *UserService) DeleteUser(userID int64) error {
	// Rozpocznij transakcję
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("nie można rozpocząć transakcji: %v", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Usuń wszystkie pomiary powiązane z urządzeniami użytkownika
	if err := tx.Exec(`
		DELETE FROM measurements 
		WHERE ispindel_id IN (
			SELECT id FROM ispindels WHERE user_id = ?
		)`, userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nie można usunąć pomiarów: %v", err)
	}

	// Usuń wszystkie fermentacje użytkownika
	if err := tx.Exec("DELETE FROM fermentations WHERE user_id = ?", userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nie można usunąć fermentacji: %v", err)
	}

	// Usuń wszystkie urządzenia iSpindel użytkownika
	if err := tx.Exec("DELETE FROM ispindels WHERE user_id = ?", userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nie można usunąć urządzeń: %v", err)
	}

	// Na końcu usuń konto użytkownika
	if err := tx.Exec("DELETE FROM users WHERE id = ?", userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("nie można usunąć użytkownika: %v", err)
	}

	// Zatwierdź transakcję
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("nie można zatwierdzić transakcji: %v", err)
	}

	return nil
}

// GetUserByGoogleID pobiera użytkownika po identyfikatorze Google
func (s *UserService) GetUserByGoogleID(googleID string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("google_id = ?", googleID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByPiwoID pobiera użytkownika po identyfikatorze piwo.org
func (s *UserService) GetUserByPiwoID(piwoID int) (*models.User, error) {
	var user models.User
	if err := s.db.Where("piwo_id = ?", piwoID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser tworzy nowego użytkownika
func (s *UserService) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

// UpdateUser aktualizuje dane użytkownika
func (s *UserService) UpdateUser(user *models.User) error {
	return s.db.Save(user).Error
}

// GetUserCount zwraca całkowitą liczbę użytkowników
func (s *UserService) GetUserCount() (int64, error) {
	var count int64
	if err := s.db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetActiveUserCount zwraca liczbę aktywnych użytkowników
func (s *UserService) GetActiveUserCount() (int64, error) {
	var count int64
	if err := s.db.Model(&models.User{}).Where("is_active = ?", true).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetAllUsers zwraca wszystkich użytkowników systemu
func (s *UserService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := s.db.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
