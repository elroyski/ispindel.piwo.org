package services

import (
	"errors"
	"time"

	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/pkg/auth"
	"ispindel.piwo.org/pkg/database"
	"ispindel.piwo.org/pkg/mailer"
	"ispindel.piwo.org/pkg/utils"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Register(name, email, password, ip string) error {
	// Sprawdź czy użytkownik już istnieje
	var existingUser models.User
	if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
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

	if err := database.DB.Create(&user).Error; err != nil {
		return err
	}

	// Wyślij email z linkiem aktywacyjnym
	err = mailer.SendActivationEmail(email, name, activationToken)
	if err != nil {
		// Logujemy błąd, ale nie przerywamy rejestracji
		// W prawdziwej aplikacji można rozważyć jakiś mechanizm ponownego wysyłania
		return errors.New("konto zostało utworzone, ale nie mogliśmy wysłać e-maila aktywacyjnego. Skontaktuj się z administracją")
	}

	return nil
}

func (s *UserService) ActivateAccount(token string) error {
	var user models.User
	if err := database.DB.Where("activation_token = ?", token).First(&user).Error; err != nil {
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

	if err := database.DB.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

func (s *UserService) Login(email, password, ip string) (string, error) {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
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
		database.DB.Save(&user)
		return "", errors.New("nieprawidłowy email lub hasło")
	}

	// Resetuj licznik nieudanych prób
	user.FailedLogins = 0
	user.LastLoginAt = time.Now()
	database.DB.Save(&user)

	// Generuj token JWT
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ResendActivationEmail(email string) error {
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
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

	if err := database.DB.Save(&user).Error; err != nil {
		return err
	}

	// Wyślij email z nowym linkiem aktywacyjnym
	return mailer.SendActivationEmail(user.Email, user.Name, activationToken)
} 