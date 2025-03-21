package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

// UpdateIspindel aktualizuje dane urządzenia
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

	ispindel, err := h.ispindelService.GetIspindelByID(uint(ispindelID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Nie znaleziono urządzenia",
			"user":  userModel,
		})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")

	if name == "" {
		c.HTML(http.StatusBadRequest, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nazwa urządzenia jest wymagana",
			"title":       "Edytuj urządzenie iSpindel",
			"ispindel":    ispindel,
			"name":        ispindel.Name,
			"description": ispindel.Description,
			"isEdit":      true,
		})
		return
	}

	ispindel.Name = name
	ispindel.Description = description

	err = h.ispindelService.UpdateIspindel(ispindel)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ispindel_form.html", gin.H{
			"user":        userModel,
			"error":       "Nie udało się zaktualizować urządzenia: " + err.Error(),
			"title":       "Edytuj urządzenie iSpindel",
			"ispindel":    ispindel,
			"name":        name,
			"description": description,
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
		log.Printf("Brak klucza API w żądaniu")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Brak klucza API"})
		return
	}

	ispindel, err := h.ispindelService.FindIspindelByAPIKey(apiKey)
	if err != nil {
		log.Printf("Nieprawidłowy klucz API: %s, błąd: %s", apiKey, err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Nieprawidłowy klucz API"})
		return
	}

	// Loguj informację o otrzymanym żądaniu
	log.Printf("Odebrano dane dla urządzenia: %s (ID: %d) od adresu IP: %s", 
		ispindel.Name, ispindel.ID, c.ClientIP())

	// Odczytaj dane JSON z ciała żądania
	var data map[string]interface{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Błąd odczytu danych z żądania: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nie udało się odczytać danych"})
		return
	}

	// Wyświetl otrzymane dane w logach
	log.Printf("Otrzymano dane: %s", string(body))

	err = json.Unmarshal(body, &data)
	if err != nil {
		// Próbuj odczytać jako tablicę obiektów
		var dataArray []map[string]interface{}
		err = json.Unmarshal(body, &dataArray)
		if err != nil {
			log.Printf("Nieprawidłowy format danych JSON: %s", err.Error())
			
			// Sprawdź, czy to nie jest prosty tekst
			if len(body) > 0 && !strings.Contains(string(body), "{") && !strings.Contains(string(body), "[") {
				log.Printf("Otrzymano dane w formacie tekstowym zamiast JSON")
				c.String(http.StatusOK, "Otrzymano dane tekstowe zamiast JSON. System wymaga danych w formacie JSON.")
				return
			}
			
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format danych JSON"})
			return
		}

		// Przetwórz każdy element tablicy
		successful := 0
		failed := 0
		
		for _, item := range dataArray {
			_, err := h.ispindelService.SaveMeasurement(ispindel.ID, item)
			if err != nil {
				failed++
				log.Printf("Błąd zapisywania pomiaru z tablicy: %s, dane: %v", err.Error(), item)
			} else {
				successful++
			}
		}

		log.Printf("Zapisano %d pomiarów z tablicy, niepowodzenie dla %d pomiarów", successful, failed)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Pomyślnie zapisano dane: %d z %d", successful, successful+failed),
			"count":   successful,
			"failed":  failed,
			"time":    time.Now(),
		})
		return
	}

	// Zapisz pojedynczy pomiar
	measurement, err := h.ispindelService.SaveMeasurement(ispindel.ID, data)
	if err != nil {
		log.Printf("Błąd zapisywania pojedynczego pomiaru: %s, dane: %v", err.Error(), data)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zapisać danych: " + err.Error()})
		return
	}

	log.Printf("Pomyślnie zapisano pomiar dla urządzenia: %s, temp: %.2f°, gęstość: %.4f", 
		ispindel.Name, measurement.Temperature, measurement.Gravity)
		
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Pomyślnie zapisano dane",
		"measurement": measurement,
		"time":        time.Now(),
	})
}

// ReceiveDataNoAPIKey obsługuje dane z urządzeń iSpindel, które nie przekazują klucza API w ścieżce URL
func (h *IspindelHandler) ReceiveDataNoAPIKey(c *gin.Context) {
	// Odczytaj dane JSON z ciała żądania
	var data map[string]interface{}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Błąd odczytu danych z żądania bez klucza API w URL: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nie udało się odczytać danych"})
		return
	}

	// Wyświetl otrzymane dane w logach
	log.Printf("Otrzymano dane bez klucza API w URL: %s", string(body))

	// Próba odczytania klucza API z nagłówka
	apiKey := c.GetHeader("X-API-KEY")
	if apiKey == "" {
		apiKey = c.GetHeader("API-KEY")
	}

	if apiKey == "" {
		// Próba odczytania klucza API z danych JSON
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Printf("Nieprawidłowy format danych JSON bez klucza API w URL: %s", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format danych JSON"})
			return
		}

		// Sprawdzanie różnych możliwych nazw pola z kluczem API
		if val, ok := data["api_key"].(string); ok && val != "" {
			apiKey = val
		} else if val, ok := data["apikey"].(string); ok && val != "" {
			apiKey = val
		} else if val, ok := data["token"].(string); ok && val != "" {
			apiKey = val
		}
	}

	// Jeśli nadal brak klucza API
	if apiKey == "" {
		log.Printf("Brak klucza API w żądaniu (ani w URL, nagłówkach ani w danych JSON)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Brak klucza API"})
		return
	}

	// Znajdź urządzenie po kluczu API
	ispindel, err := h.ispindelService.FindIspindelByAPIKey(apiKey)
	if err != nil {
		log.Printf("Nieprawidłowy klucz API: %s, błąd: %s", apiKey, err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Nieprawidłowy klucz API"})
		return
	}

	// Loguj informację o otrzymanym żądaniu
	log.Printf("Odebrano dane bez klucza API w URL dla urządzenia: %s (ID: %d) od adresu IP: %s", 
		ispindel.Name, ispindel.ID, c.ClientIP())

	// Jeśli wcześniej nie udało się odczytać danych JSON, spróbuj ponownie
	if data == nil {
		err = json.Unmarshal(body, &data)
		if err != nil {
			log.Printf("Nieprawidłowy format danych JSON bez klucza API w URL: %s", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nieprawidłowy format danych JSON"})
			return
		}
	}

	// Zapisz pomiar
	measurement, err := h.ispindelService.SaveMeasurement(ispindel.ID, data)
	if err != nil {
		log.Printf("Błąd zapisywania pomiaru bez klucza API w URL: %s, dane: %v", err.Error(), data)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nie udało się zapisać danych: " + err.Error()})
		return
	}

	log.Printf("Pomyślnie zapisano pomiar dla urządzenia bez klucza API w URL: %s, temp: %.2f°, gęstość: %.4f", 
		ispindel.Name, measurement.Temperature, measurement.Gravity)
		
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Pomyślnie zapisano dane",
		"measurement": measurement,
		"time":        time.Now(),
	})
} 