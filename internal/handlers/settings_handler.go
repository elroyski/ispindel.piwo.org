package handlers

import (
	"bufio"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
)

// SettingsHandler obsługuje ustawienia konta użytkownika i systemu
type SettingsHandler struct {
	userService *services.UserService
}

// NewSettingsHandler tworzy nową instancję handlera ustawień
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{
		userService: services.NewUserService(),
	}
}

// getSystemVersion pobiera wersję systemu z pliku go.mod
func getSystemVersion() string {
	defaultVersion := "1.0.0"

	// Otwórz plik go.mod
	file, err := os.Open("go.mod")
	if err != nil {
		return defaultVersion
	}
	defer file.Close()

	// Wyszukaj linię z wersją Go
	scanner := bufio.NewScanner(file)
	goVersionRegex := regexp.MustCompile(`go (\d+\.\d+\.\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "go ") {
			matches := goVersionRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				return matches[1] // Zwróć wersję Go jako wersję systemu
			}
		}
	}

	return defaultVersion
}

// Settings wyświetla stronę ustawień konta i systemu
func (h *SettingsHandler) Settings(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	// Pobierz wersję systemu
	version := getSystemVersion()

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"user":    user.(*models.User),
		"version": version,
	})
}

// ChangePassword obsługuje zmianę hasła użytkownika
func (h *SettingsHandler) ChangePassword(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}
	userModel := user.(*models.User)

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	// Pobierz wersję systemu
	version := getSystemVersion()

	// Sprawdzenie czy nowe hasło i potwierdzenie są takie same
	if newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"user":    userModel,
			"error":   "Nowe hasło i potwierdzenie hasła nie są identyczne",
			"version": version,
		})
		return
	}

	// Aktualizacja hasła w serwisie
	err := h.userService.ChangePassword(userModel.ID, currentPassword, newPassword)
	if err != nil {
		c.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"user":    userModel,
			"error":   "Nie udało się zmienić hasła: " + err.Error(),
			"version": version,
		})
		return
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"user":    userModel,
		"success": "Hasło zostało zmienione pomyślnie",
		"version": version,
	})
}

// DeleteAccount obsługuje żądanie usunięcia konta użytkownika
func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}
	userModel := user.(*models.User)

	// Jeśli użytkownik ma hasło (zalogowany standardową metodą), sprawdź je
	if userModel.Password != "" {
		// Pobierz i sprawdź hasło
		password := c.PostForm("password")
		if password == "" {
			c.HTML(http.StatusBadRequest, "settings.html", gin.H{
				"user":    userModel,
				"error":   "Hasło jest wymagane do usunięcia konta",
				"version": getSystemVersion(),
			})
			return
		}

		// Sprawdź poprawność hasła
		if err := bcrypt.CompareHashAndPassword([]byte(userModel.Password), []byte(password)); err != nil {
			c.HTML(http.StatusBadRequest, "settings.html", gin.H{
				"user":    userModel,
				"error":   "Nieprawidłowe hasło",
				"version": getSystemVersion(),
			})
			return
		}
	}

	// Usuń konto użytkownika
	if err := h.userService.DeleteUser(int64(userModel.ID)); err != nil {
		c.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"user":    userModel,
			"error":   "Nie udało się usunąć konta: " + err.Error(),
			"version": getSystemVersion(),
		})
		return
	}

	// Wyloguj użytkownika
	c.SetCookie("session", "", -1, "/", "", false, true)
	c.SetCookie("token", "", -1, "/", "", false, true)

	// Przekieruj na stronę logowania
	c.Redirect(http.StatusSeeOther, "/auth/login")
}
