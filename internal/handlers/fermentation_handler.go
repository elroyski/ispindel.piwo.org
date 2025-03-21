package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
)

type FermentationHandler struct {
	FermentationService *services.FermentationService
	IspindelService     *services.IspindelService
}

func NewFermentationHandler() *FermentationHandler {
	return &FermentationHandler{
		FermentationService: services.NewFermentationService(),
		IspindelService:     services.NewIspindelService(),
	}
}

// GetBeerStyles pobiera style piwa z pliku JSON
func (h *FermentationHandler) GetBeerStyles() ([]map[string]string, error) {
	// Możliwe lokalizacje pliku stylów piwa
	possiblePaths := []string{
		filepath.Join("static", "data", "beer_styles.json"),
		filepath.Join("beer_styles.json"),
	}

	var fileContent []byte
	var err error
	var loadedPath string

	// Próbuj odczytać plik z różnych lokalizacji
	for _, path := range possiblePaths {
		fileContent, err = ioutil.ReadFile(path)
		if err == nil {
			loadedPath = path
			break
		}
	}

	// Jeśli plik nie został znaleziony, zwróć przyjazny komunikat błędu
	if err != nil {
		return nil, fmt.Errorf("nie można znaleźć pliku ze stylami piwa. Spróbuj wykonać: "+
			"wget -O static/data/beer_styles.json https://raw.githubusercontent.com/beerjson/bjcp-json/main/styles/bjcp_styleguide-2021.json")
	}

	// Parsowanie JSON
	var data map[string]interface{}
	if err := json.Unmarshal(fileContent, &data); err != nil {
		return nil, fmt.Errorf("błąd parsowania pliku JSON ze stylami piwa (%s): %v", loadedPath, err)
	}

	// Pobieranie stylów piwa
	var styles []map[string]string
	
	// Dodaj pierwszy element jako "styl własny"
	styles = append(styles, map[string]string{
		"name":     "Styl własny",
		"category": "",
		"style_id": "OWN",
	})

	// Przetwarzanie danych z pliku JSON - sprawdzamy, czy dane są w oczekiwanej strukturze
	beerJson, ok := data["beerjson"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("nieprawidłowa struktura pliku JSON - brak klucza beerjson")
	}
	
	stylesArray, ok := beerJson["styles"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("nieprawidłowa struktura pliku JSON - brak stylesArray w beerjson")
	}
	
	for _, style := range stylesArray {
		if styleMap, ok := style.(map[string]interface{}); ok {
			// Pobierz tylko potrzebne pola
			name, nameOk := styleMap["name"].(string)
			category, categoryOk := styleMap["category"].(string)
			styleID, styleIDOk := styleMap["style_id"].(string)

			if nameOk && categoryOk && styleIDOk {
				styles = append(styles, map[string]string{
					"name":     name,
					"category": category,
					"style_id": styleID,
				})
			}
		}
	}

	return styles, nil
}

// FermentationsList wyświetla listę fermentacji użytkownika
func (h *FermentationHandler) FermentationsList(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz fermentacje użytkownika
	fermentations, err := h.FermentationService.GetFermentationsByUserID(userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas pobierania danych: " + err.Error()})
		return
	}

	// Renderuj szablon z listą fermentacji
	c.HTML(http.StatusOK, "fermentations.html", gin.H{
		"user":         userModel,
		"fermentations": fermentations,
	})
}

// NewFermentationForm wyświetla formularz do tworzenia nowej fermentacji
func (h *FermentationHandler) NewFermentationForm(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz aktywne urządzenia iSpindel użytkownika (niewykorzystane w aktywnych fermentacjach)
	activeIspindels, err := h.FermentationService.GetActiveIspindelsForUser(userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas pobierania urządzeń: " + err.Error()})
		return
	}

	// Pobierz wszystkie urządzenia iSpindel użytkownika
	allIspindels, err := h.IspindelService.GetIspindelsByUserID(userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas pobierania urządzeń: " + err.Error()})
		return
	}

	// Znajdź urządzenia, które są aktywne, ale już wykorzystywane w fermentacjach
	var usedIspindels []models.Ispindel
	for _, isp := range allIspindels {
		if !isp.IsActive {
			continue
		}

		isUsed := true
		for _, availableIsp := range activeIspindels {
			if availableIsp.ID == isp.ID {
				isUsed = false
				break
			}
		}

		if isUsed {
			usedIspindels = append(usedIspindels, isp)
		}
	}

	// Pobierz style piwa
	beerStyles, err := h.GetBeerStyles()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas pobierania stylów piwa: " + err.Error()})
		return
	}

	// Renderuj formularz nowej fermentacji
	c.HTML(http.StatusOK, "fermentation_form.html", gin.H{
		"user":           userModel,
		"activeIspindels": activeIspindels,
		"usedIspindels":  usedIspindels,
		"beerStyles":     beerStyles,
		"formTitle":      "Nowa fermentacja",
		"submitButton":   "Rozpocznij fermentację",
	})
}

// CreateFermentation tworzy nową fermentację
func (h *FermentationHandler) CreateFermentation(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz dane z formularza
	name := c.PostForm("name")
	styleID := c.PostForm("style_id")
	description := c.PostForm("description")
	ispindelIDStr := c.PostForm("ispindel_id")

	// Walidacja danych
	if name == "" {
		// Jeśli nazwa jest pusta, zwróć formularz z błędem
		activeIspindels, _ := h.FermentationService.GetActiveIspindelsForUser(userModel.ID)
		beerStyles, _ := h.GetBeerStyles()

		c.HTML(http.StatusBadRequest, "fermentation_form.html", gin.H{
			"user":          userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":    beerStyles,
			"formTitle":     "Nowa fermentacja",
			"submitButton":  "Rozpocznij fermentację",
			"error":         "Nazwa warki jest wymagana",
			"name":          name,
			"style_id":      styleID,
			"description":   description,
			"ispindel_id":   ispindelIDStr,
		})
		return
	}

	// Konwersja ID iSpindel na uint
	ispindelID, err := strconv.ParseUint(ispindelIDStr, 10, 64)
	if err != nil || ispindelID == 0 {
		// Jeśli nie wybrano urządzenia, zwróć formularz z błędem
		activeIspindels, _ := h.FermentationService.GetActiveIspindelsForUser(userModel.ID)
		beerStyles, _ := h.GetBeerStyles()

		c.HTML(http.StatusBadRequest, "fermentation_form.html", gin.H{
			"user":          userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":    beerStyles,
			"formTitle":     "Nowa fermentacja",
			"submitButton":  "Rozpocznij fermentację",
			"error":         "Wybór urządzenia iSpindel jest wymagany",
			"name":          name,
			"style_id":      styleID,
			"description":   description,
		})
		return
	}

	// Znajdź styl piwa na podstawie styleID
	var style, styleCategory string
	beerStyles, _ := h.GetBeerStyles()
	for _, s := range beerStyles {
		if s["style_id"] == styleID {
			style = s["name"]
			styleCategory = s["category"]
			break
		}
	}

	// Utwórz nową fermentację
	fermentation, err := h.FermentationService.CreateFermentation(
		userModel.ID,
		name,
		style,
		styleID,
		styleCategory,
		description,
		uint(ispindelID),
	)

	if err != nil {
		// W przypadku błędu, zwróć formularz z błędem
		activeIspindels, _ := h.FermentationService.GetActiveIspindelsForUser(userModel.ID)

		c.HTML(http.StatusInternalServerError, "fermentation_form.html", gin.H{
			"user":          userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":    beerStyles,
			"formTitle":     "Nowa fermentacja",
			"submitButton":  "Rozpocznij fermentację",
			"error":         "Błąd podczas tworzenia fermentacji: " + err.Error(),
			"name":          name,
			"style_id":      styleID,
			"description":   description,
			"ispindel_id":   ispindelIDStr,
		})
		return
	}

	// Przekieruj na stronę szczegółów nowej fermentacji
	c.Redirect(http.StatusSeeOther, "/fermentations/" + strconv.Itoa(int(fermentation.ID)))
}

// FermentationDetails wyświetla szczegóły fermentacji
func (h *FermentationHandler) FermentationDetails(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz ID fermentacji z parametrów URL
	fermentationIDStr := c.Param("id")
	fermentationID, err := strconv.ParseUint(fermentationIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Nieprawidłowe ID fermentacji"})
		return
	}

	// Pobierz szczegóły fermentacji
	fermentation, err := h.FermentationService.GetFermentationByID(uint(fermentationID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Nie znaleziono fermentacji: " + err.Error()})
		return
	}

	// Pobierz dane pomiarowe dla iSpindel przypisanego do tej fermentacji
	var measurements []models.Measurement
	var hasData bool
	var timestamps, temperatures, gravities, angles, batteries, rssiValues []interface{}
	
	if fermentation.IspindelID > 0 {
		// Pobierz dane pomiarowe dla tego urządzenia
		// Jeśli fermentacja jest aktywna, pobierz dane od momentu rozpoczęcia
		// Jeśli fermentacja jest zakończona, pobierz dane z całego okresu fermentacji
		var startTime, endTime time.Time
		startTime = fermentation.StartedAt
		
		if fermentation.IsActive {
			endTime = time.Now()
		} else if fermentation.EndedAt != nil {
			endTime = *fermentation.EndedAt
		} else {
			endTime = time.Now()
		}
		
		// Pobierz dane pomiarowe z okresu fermentacji
		measurements, err = h.IspindelService.GetMeasurementsForIspindelInRange(fermentation.IspindelID, startTime, endTime, 10)
		if err == nil && len(measurements) > 0 {
			hasData = true
			
			// Przygotuj dane do wykresów
			for _, m := range measurements {
				timestamps = append(timestamps, m.Timestamp.Format("02.01 15:04"))
				temperatures = append(temperatures, m.Temperature)
				gravities = append(gravities, m.Gravity)
				angles = append(angles, m.Angle)
				batteries = append(batteries, m.Battery)
				rssiValues = append(rssiValues, m.RSSI)
			}
		}
	}

	// Renderuj szablon szczegółów fermentacji
	c.HTML(http.StatusOK, "fermentation_details.html", gin.H{
		"user":         userModel,
		"fermentation": fermentation,
		"hasData":      hasData,
		"measurements": measurements,
		"timestamps":   timestamps,
		"temperatures": temperatures,
		"gravities":    gravities,
		"angles":       angles,
		"batteries":    batteries,
		"rssi":         rssiValues,
	})
}

// EndFermentation kończy aktywną fermentację
func (h *FermentationHandler) EndFermentation(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz ID fermentacji z parametrów URL
	fermentationIDStr := c.Param("id")
	fermentationID, err := strconv.ParseUint(fermentationIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Nieprawidłowe ID fermentacji"})
		return
	}

	// Zakończ fermentację
	err = h.FermentationService.EndFermentation(uint(fermentationID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas kończenia fermentacji: " + err.Error()})
		return
	}

	// Przekieruj z powrotem do szczegółów fermentacji
	c.Redirect(http.StatusSeeOther, "/fermentations/" + fermentationIDStr)
}

// DeleteFermentation usuwa zakończoną fermentację
func (h *FermentationHandler) DeleteFermentation(c *gin.Context) {
	// Pobierz zalogowanego użytkownika
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Wymagane zalogowanie"})
		return
	}

	userModel := user.(*models.User)

	// Pobierz ID fermentacji z parametrów URL
	fermentationIDStr := c.Param("id")
	fermentationID, err := strconv.ParseUint(fermentationIDStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Nieprawidłowe ID fermentacji"})
		return
	}

	// Usuń fermentację
	err = h.FermentationService.DeleteFermentation(uint(fermentationID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Błąd podczas usuwania fermentacji: " + err.Error()})
		return
	}

	// Przekieruj do listy fermentacji
	c.Redirect(http.StatusSeeOther, "/fermentations")
} 