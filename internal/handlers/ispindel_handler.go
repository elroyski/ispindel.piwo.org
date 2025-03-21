package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
)

type IspindelHandler struct {
	ispindelService *services.IspindelService
}

func NewIspindelHandler() *IspindelHandler {
	return &IspindelHandler{
		ispindelService: services.NewIspindelService(),
	}
}

// Strona z listą urządzeń iSpindel użytkownika
func (h *IspindelHandler) ListIspindels(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userModel := user.(*models.User)
	ispindels, err := h.ispindelService.GetIspindelsByUserID(userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Nie udało się pobrać listy urządzeń: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "ispindels.html", gin.H{
		"user":     userModel,
		"ispindels": ispindels,
	})
}

// Formularz dodawania nowego urządzenia
func (h *IspindelHandler) NewIspindelForm(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	c.HTML(http.StatusOK, "ispindel_form.html", gin.H{
		"user":  user.(*models.User),
		"title": "Dodaj nowe urządzenie iSpindel",
	})
}

// Dodawanie nowego urządzenia
func (h *IspindelHandler) CreateIspindel(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userModel := user.(*models.User)
	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.HTML(http.StatusBadRequest, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nazwa urządzenia jest wymagana",
			"title":       "Dodaj nowe urządzenie iSpindel",
			"name":        name,
			"description": description,
		})
		return
	}

	ispindel, err := h.ispindelService.CreateIspindel(userModel.ID, name, description)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nie udało się dodać urządzenia: " + err.Error(),
			"title":       "Dodaj nowe urządzenie iSpindel",
			"name":        name,
			"description": description,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/ispindels/"+strconv.FormatUint(uint64(ispindel.ID), 10))
}

// Szczegóły urządzenia
func (h *IspindelHandler) IspindelDetails(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userModel := user.(*models.User)
	ispindelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator urządzenia",
			"user":  userModel,
		})
		return
	}

	ispindel, err := h.ispindelService.GetIspindelByID(uint(ispindelID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Nie znaleziono urządzenia",
			"user":  userModel,
		})
		return
	}

	// Pobierz ostatnie pomiary
	measurements, err := h.ispindelService.GetLatestMeasurements(ispindel.ID, 20)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Nie udało się pobrać pomiarów: " + err.Error(),
			"user":  userModel,
		})
		return
	}

	// Przygotuj dane do wykresu
	var timestamps []string
	var temperatures []float64
	var gravities []float64
	var batteries []float64

	for i := len(measurements) - 1; i >= 0; i-- {
		m := measurements[i]
		timestamps = append(timestamps, m.Timestamp.Format("2006-01-02 15:04:05"))
		temperatures = append(temperatures, m.Temperature)
		gravities = append(gravities, m.Gravity)
		batteries = append(batteries, m.Battery)
	}

	c.HTML(http.StatusOK, "ispindel_details.html", gin.H{
		"user":         userModel,
		"ispindel":     ispindel,
		"measurements": measurements,
		"timestamps":   timestamps,
		"temperatures": temperatures,
		"gravities":    gravities,
		"batteries":    batteries,
		"hasData":      len(measurements) > 0,
		"apiEndpoint":  "/api/ispindel/" + ispindel.APIKey,
	})
}

// Formularz edycji urządzenia
func (h *IspindelHandler) EditIspindelForm(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userModel := user.(*models.User)
	ispindelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator urządzenia",
			"user":  userModel,
		})
		return
	}

	ispindel, err := h.ispindelService.GetIspindelByID(uint(ispindelID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Nie znaleziono urządzenia",
			"user":  userModel,
		})
		return
	}

	c.HTML(http.StatusOK, "ispindel_form.html", gin.H{
		"user":        userModel,
		"title":       "Edytuj urządzenie iSpindel",
		"ispindel":    ispindel,
		"name":        ispindel.Name,
		"description": ispindel.Description,
		"isActive":    ispindel.IsActive,
		"isEdit":      true,
	})
}

// Aktualizacja urządzenia
func (h *IspindelHandler) UpdateIspindel(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	userModel := user.(*models.User)
	ispindelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator urządzenia",
			"user":  userModel,
		})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	isActive := c.PostForm("is_active") == "on"

	if name == "" {
		c.HTML(http.StatusBadRequest, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nazwa urządzenia jest wymagana",
			"title":       "Edytuj urządzenie iSpindel",
			"ispindel":    &models.Ispindel{Model: gorm.Model{ID: uint(ispindelID)}},
			"name":        name,
			"description": description,
			"isActive":    isActive,
			"isEdit":      true,
		})
		return
	}

	ispindel, err := h.ispindelService.UpdateIspindel(uint(ispindelID), userModel.ID, name, description, isActive)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nie udało się zaktualizować urządzenia: " + err.Error(),
			"title":       "Edytuj urządzenie iSpindel",
			"ispindel":    &models.Ispindel{Model: gorm.Model{ID: uint(ispindelID)}},
			"name":        name,
			"description": description,
			"isActive":    isActive,
			"isEdit":      true,
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/ispindels/"+strconv.FormatUint(uint64(ispindel.ID), 10))
}

// Regeneracja klucza API
func (h *IspindelHandler) RegenerateAPIKey(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Wymagane logowanie"})
		return
	}

	userModel := user.(*models.User)
	ispindelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy identyfikator urządzenia"})
		return
	}

	apiKey, err := h.ispindelService.RegenerateAPIKey(uint(ispindelID), userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się wygenerować nowego klucza API: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
}

// Usunięcie urządzenia
func (h *IspindelHandler) DeleteIspindel(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Wymagane logowanie"})
		return
	}

	userModel := user.(*models.User)
	ispindelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy identyfikator urządzenia"})
		return
	}

	err = h.ispindelService.DeleteIspindel(uint(ispindelID), userModel.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się usunąć urządzenia: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// API endpoint do odbierania danych z urządzeń iSpindel
func (h *IspindelHandler) ReceiveData(c *gin.Context) {
	apiKey := c.Param("api_key")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Brak klucza API"})
		return
	}

	ispindel, err := h.ispindelService.FindIspindelByAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Nieprawidłowy klucz API"})
		return
	}

	// Odczytaj dane JSON z ciała żądania
	var data map[string]interface{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nie udało się odczytać danych"})
		return
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		// Próbuj odczytać jako tablicę obiektów
		var dataArray []map[string]interface{}
		err = json.Unmarshal(body, &dataArray)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format danych"})
			return
		}

		// Przetwórz każdy element tablicy
		for _, item := range dataArray {
			_, err := h.ispindelService.SaveMeasurement(ispindel.ID, item)
			if err != nil {
				// Loguj błąd, ale kontynuuj przetwarzanie
				// W przyszłości można dodać logowanie do pliku
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Pomyślnie zapisano dane (tablicę)",
			"count":   len(dataArray),
			"time":    time.Now(),
		})
		return
	}

	// Zapisz pojedynczy pomiar
	measurement, err := h.ispindelService.SaveMeasurement(ispindel.ID, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zapisać danych: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Pomyślnie zapisano dane",
		"measurement": measurement,
		"time":        time.Now(),
	})
} 