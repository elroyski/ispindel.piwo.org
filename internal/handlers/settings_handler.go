package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

// Settings wyświetla stronę ustawień konta i systemu
func (h *SettingsHandler) Settings(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	// Definicja wersji systemu
	version := "1.0.0"

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

	// Sprawdzenie czy nowe hasło i potwierdzenie są takie same
	if newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"user":    userModel,
			"error":   "Nowe hasło i potwierdzenie hasła nie są identyczne",
			"version": "1.0.0",
		})
		return
	}

	// Aktualizacja hasła w serwisie
	err := h.userService.ChangePassword(userModel.ID, currentPassword, newPassword)
	if err != nil {
		c.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"user":    userModel,
			"error":   "Nie udało się zmienić hasła: " + err.Error(),
			"version": "1.0.0",
		})
		return
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"user":    userModel,
		"success": "Hasło zostało zmienione pomyślnie",
		"version": "1.0.0",
	})
}
