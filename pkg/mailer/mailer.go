package mailer

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"net/smtp"
)

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

var config EmailConfig

// InitMailer inicjalizuje konfigurację e-mail z zmiennych środowiskowych
func InitMailer() {
	config = EmailConfig{
		Host:     getEnvOrDefault("SMTP_HOST", "mail18.mydevil.net"),
		Port:     getEnvIntOrDefault("SMTP_PORT", 587),
		Username: getEnvOrDefault("SMTP_USER", "ispindel@piwo.org"),
		Password: getEnvOrDefault("SMTP_PASSWORD", "Kochanapysia1"),
		From:     getEnvOrDefault("SMTP_FROM", "ispindel@piwo.org"),
	}

	log.Printf("Konfiguracja poczty: %s:%d, użytkownik: %s", config.Host, config.Port, config.Username)
}

// SendActivationEmail wysyła e-mail z linkiem aktywacyjnym
func SendActivationEmail(to, name, token string) error {
	subject := "Aktywacja konta iSpindel"
	
	baseURL := getEnvOrDefault("APP_URL", "http://ispindle.piwo.org:49330")
	activationURL := fmt.Sprintf("%s/auth/activate?token=%s", baseURL, token)
	
	body := fmt.Sprintf(`Cześć %s,

Dziękujemy za rejestrację na platformie iSpindel. Aby aktywować swoje konto, kliknij na poniższy link:

%s

Link jest ważny przez 24 godziny.

Jeśli nie zarejestrowałeś się na naszej platformie, zignoruj tę wiadomość.

Pozdrawiamy,
Zespół iSpindel
`, name, activationURL)

	return sendEmail(to, subject, body)
}

// SendPasswordResetEmail wysyła e-mail z linkiem do resetowania hasła
func SendPasswordResetEmail(to, name, token string) error {
	subject := "Reset hasła iSpindel"
	
	baseURL := getEnvOrDefault("APP_URL", "http://ispindle.piwo.org:49330")
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, token)
	
	body := fmt.Sprintf(`Cześć %s,

Otrzymaliśmy prośbę o zresetowanie hasła na platformie iSpindel. Aby zresetować hasło, kliknij na poniższy link:

%s

Link jest ważny przez 1 godzinę.

Jeśli nie zgłaszałeś prośby o reset hasła, zignoruj tę wiadomość.

Pozdrawiamy,
Zespół iSpindel
`, name, resetURL)

	return sendEmail(to, subject, body)
}

// sendEmail wysyła e-mail używając konfiguracji SMTP
func sendEmail(to, subject, body string) error {
	// Buduj nagłówki wiadomości
	message := fmt.Sprintf("From: %s\r\n", config.From)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// Adres SMTP w formacie host:port
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// Uwierzytelnienie
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// Wysłanie wiadomości
	err := smtp.SendMail(addr, auth, config.From, []string{to}, []byte(message))
	if err != nil {
		log.Printf("Błąd wysyłania e-maila: %v", err)
		return err
	}

	log.Printf("E-mail wysłany pomyślnie do %s", to)
	return nil
}

// Funkcje pomocnicze do pobierania zmiennych środowiskowych
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
} 