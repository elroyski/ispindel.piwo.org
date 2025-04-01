package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
	"ispindel.piwo.org/pkg/database"

	"github.com/gin-gonic/gin"
)

// AdminHandler obsługuje żądania związane z panelem administracyjnym
type AdminHandler struct {
	UserService         *services.UserService
	IspindelService     *services.IspindelService
	FermentationService *services.FermentationService
	AdminEmail          string
}

// NewAdminHandler tworzy nowy handler administratora
func NewAdminHandler(userService *services.UserService, ispindelService *services.IspindelService, fermentationService *services.FermentationService) *AdminHandler {
	return &AdminHandler{
		UserService:         userService,
		IspindelService:     ispindelService,
		FermentationService: fermentationService,
		AdminEmail:          "elroyski@gmail.com", // Stały email administratora
	}
}

// AdminRequired to middleware sprawdzające, czy użytkownik ma uprawnienia administratora
func (h *AdminHandler) AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusSeeOther, "/auth/login")
			c.Abort()
			return
		}

		userModel := user.(*models.User)
		if userModel.Email != h.AdminEmail {
			c.HTML(http.StatusForbidden, "error.html", gin.H{
				"error": "Brak uprawnień do tej strony",
				"user":  userModel,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Dashboard wyświetla główny panel administracyjny
func (h *AdminHandler) Dashboard(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Pobierz podstawowe statystyki
	userCount, _ := h.UserService.GetUserCount()
	ispindelCount, _ := h.IspindelService.GetIspindelCount()
	fermentationCount, _ := h.FermentationService.GetFermentationCount()
	activeUserCount, _ := h.UserService.GetActiveUserCount()

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"user":              userModel,
		"userCount":         userCount,
		"ispindelCount":     ispindelCount,
		"fermentationCount": fermentationCount,
		"activeUserCount":   activeUserCount,
		"isAdmin":           true,
	})
}

// ListUsers wyświetla listę wszystkich użytkowników
func (h *AdminHandler) ListUsers(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Pobierz wszystkich użytkowników
	users, err := h.UserService.GetAllUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania listy użytkowników: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_users.html", gin.H{
		"user":    userModel,
		"users":   users,
		"isAdmin": true,
	})
}

// UserDetails wyświetla szczegóły użytkownika wraz z jego urządzeniami i fermentacjami
func (h *AdminHandler) UserDetails(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Pobierz ID użytkownika z URL
	userID := c.Param("id")

	// Konwertuj string ID na uint
	userIDuint, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator użytkownika",
			"user":  userModel,
		})
		return
	}

	// Pobierz szczegóły użytkownika
	targetUser, err := h.UserService.GetUserByID(uint(userIDuint))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Użytkownik nie znaleziony",
			"user":  userModel,
		})
		return
	}

	// Pobierz urządzenia użytkownika
	ispindels, err := h.IspindelService.GetIspindelsByUserID(targetUser.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania urządzeń: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	// Pobierz fermentacje użytkownika
	fermentations, err := h.FermentationService.GetFermentationsByUserID(targetUser.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania fermentacji: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_user_details.html", gin.H{
		"user":          userModel,
		"targetUser":    targetUser,
		"ispindels":     ispindels,
		"fermentations": fermentations,
		"isAdmin":       true,
	})
}

// ListIspindels wyświetla listę wszystkich urządzeń iSpindel
func (h *AdminHandler) ListIspindels(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Pobierz wszystkie urządzenia
	ispindels, err := h.IspindelService.GetAllIspindels()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania listy urządzeń: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_ispindels.html", gin.H{
		"user":      userModel,
		"ispindels": ispindels,
		"isAdmin":   true,
	})
}

// ListFermentations wyświetla listę wszystkich fermentacji
func (h *AdminHandler) ListFermentations(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Pobierz wszystkie fermentacje
	fermentations, err := h.FermentationService.GetAllFermentations()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania listy fermentacji: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	c.HTML(http.StatusOK, "admin_fermentations.html", gin.H{
		"user":          userModel,
		"fermentations": fermentations,
		"isAdmin":       true,
	})
}

// AdminDeleteIspindel umożliwia administratorowi usunięcie urządzenia iSpindel
func (h *AdminHandler) AdminDeleteIspindel(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Sprawdź czy użytkownik jest administratorem
	if userModel.Email != h.AdminEmail {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"error": "Brak uprawnień administratora",
			"user":  userModel,
		})
		return
	}

	// Pobierz ID urządzenia z parametrów URL
	ispindelIDStr := c.Param("id")
	ispindelID, err := strconv.ParseUint(ispindelIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error":   "Nieprawidłowy identyfikator urządzenia",
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Znajdź urządzenie
	var ispindel models.Ispindel
	if err := database.DB.First(&ispindel, ispindelID).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":   "Urządzenie nie znalezione: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Rozpocznij transakcję
	tx := database.DB.Begin()

	// Znajdź wszystkie aktywne fermentacje dla tego urządzenia (bez względu na właściciela)
	var fermentations []models.Fermentation
	id := uint(ispindelID)
	if err := tx.Where("ispindel_id = ? AND is_active = true", &id).Find(&fermentations).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error":   "Błąd podczas pobierania fermentacji: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Zakończ wszystkie aktywne fermentacje
	for _, fermentation := range fermentations {
		now := time.Now()
		fermentation.EndedAt = &now
		fermentation.IsActive = false
		fermentation.IspindelID = nil // Usuń powiązanie z urządzeniem

		// Dodaj komentarz o usunięciu urządzenia
		comment := "Zakończono - urządzenie pomiarowe zostało usunięte przez administratora"
		if fermentation.Description != "" {
			fermentation.Description = fermentation.Description + "\n\n" + comment
		} else {
			fermentation.Description = comment
		}

		if err := tx.Save(&fermentation).Error; err != nil {
			tx.Rollback()
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"error":   "Błąd podczas aktualizacji fermentacji: " + err.Error(),
				"user":    userModel,
				"isAdmin": true,
			})
			return
		}
	}

	// Usuń urządzenie
	if err := tx.Delete(&ispindel).Error; err != nil {
		tx.Rollback()
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error":   "Nie udało się usunąć urządzenia: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Zatwierdź transakcję
	if err := tx.Commit().Error; err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error":   "Błąd podczas zatwierdzania transakcji: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Przekieruj z powrotem do listy urządzeń
	c.Redirect(http.StatusSeeOther, "/admin/ispindels")
}

// AdminDeleteUser umożliwia administratorowi usunięcie konta użytkownika
func (h *AdminHandler) AdminDeleteUser(c *gin.Context) {
	user, _ := c.Get("user")
	userModel := user.(*models.User)

	// Sprawdź czy użytkownik jest administratorem
	if userModel.Email != h.AdminEmail {
		c.HTML(http.StatusForbidden, "error.html", gin.H{
			"error": "Brak uprawnień administratora",
			"user":  userModel,
		})
		return
	}

	// Pobierz ID użytkownika z parametrów URL
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error":   "Nieprawidłowy identyfikator użytkownika",
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Nie pozwól na usunięcie konta administratora
	targetUser, err := h.UserService.GetUserByID(uint(userID))
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error":   "Użytkownik nie znaleziony: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	if targetUser.Email == h.AdminEmail {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error":   "Nie można usunąć konta administratora",
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Usuń konto użytkownika
	if err := h.UserService.DeleteUser(userID); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error":   "Nie udało się usunąć konta: " + err.Error(),
			"user":    userModel,
			"isAdmin": true,
		})
		return
	}

	// Przekieruj z powrotem do listy użytkowników
	c.Redirect(http.StatusSeeOther, "/admin/users")
}
