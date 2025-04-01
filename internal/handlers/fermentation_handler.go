package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"text/template"
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
		return nil, fmt.Errorf("nie można znaleźć pliku ze stylami piwa. Spróbuj wykonać: " +
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
				// Usuń kategorię z nazwy jeśli występuje na początku
				cleanName := name
				if category != "" {
					prefix := category + " "
					if len(name) > len(prefix) && name[:len(prefix)] == prefix {
						cleanName = name[len(prefix):]
					}
				}

				// Tworzymy nazwę w prostym formacie [Kategoria] Nazwa [style_id]
				formattedName := fmt.Sprintf("[%s] %s [%s]", category, cleanName, styleID)

				styles = append(styles, map[string]string{
					"name":     formattedName,
					"category": category,
					"style_id": styleID,
				})
			}
		}
	}

	return styles, nil
}

// FermentationWithDuration reprezentuje fermentację wraz z czasem trwania
type FermentationWithDuration struct {
	*models.Fermentation
	Duration        string
	LastMeasurement *models.Measurement
	Ispindel        models.Ispindel
}

// FermentationsList wyświetla listę fermentacji
func (h *FermentationHandler) FermentationsList(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{
			"error": "Nie jesteś zalogowany",
		})
		return
	}
	userModel := user.(*models.User)

	fermentations, err := h.FermentationService.GetFermentationsByUserID(userModel.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania listy fermentacji",
			"user":  userModel,
		})
		return
	}

	// Przygotuj dane z czasem trwania
	var fermentationsWithDuration []FermentationWithDuration
	for _, f := range fermentations {
		fermentation := f // Utwórz kopię, aby uniknąć problemów z wskaźnikami w pętli

		// Pobierz ostatni pomiar dla tej fermentacji
		measurements, err := h.FermentationService.GetAllMeasurements(fermentation.ID)
		var lastMeasurement *models.Measurement
		if err == nil && len(measurements) > 0 {
			// GetAllMeasurements zwraca pomiary posortowane malejąco po timestamp (od najnowszych)
			// więc pierwszy element (index 0) to najnowszy pomiar
			lastMeasurement = &measurements[0]
		}

		// Pobierz urządzenie powiązane z fermentacją
		var ispindel *models.Ispindel
		var ispindelData models.Ispindel
		if fermentation.IspindelID != nil {
			ispindel, err = h.IspindelService.GetIspindelByID(fermentation.IspindelID, userModel.ID)
			if err == nil {
				ispindelData = *ispindel
			}
		}

		fermentationsWithDuration = append(fermentationsWithDuration, FermentationWithDuration{
			Fermentation:    &fermentation,
			Duration:        h.FermentationService.GetFermentationDurationString(&fermentation),
			LastMeasurement: lastMeasurement,
			Ispindel:        ispindelData,
		})
	}

	c.HTML(http.StatusOK, "fermentations.html", gin.H{
		"user":          userModel,
		"fermentations": fermentationsWithDuration,
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
		"user":            userModel,
		"activeIspindels": activeIspindels,
		"usedIspindels":   usedIspindels,
		"beerStyles":      beerStyles,
		"formTitle":       "Nowa fermentacja",
		"submitButton":    "Rozpocznij fermentację",
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
			"user":            userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":      beerStyles,
			"formTitle":       "Nowa fermentacja",
			"submitButton":    "Rozpocznij fermentację",
			"error":           "Nazwa warki jest wymagana",
			"name":            name,
			"style_id":        styleID,
			"description":     description,
			"ispindel_id":     ispindelIDStr,
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
			"user":            userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":      beerStyles,
			"formTitle":       "Nowa fermentacja",
			"submitButton":    "Rozpocznij fermentację",
			"error":           "Wybór urządzenia iSpindel jest wymagany",
			"name":            name,
			"style_id":        styleID,
			"description":     description,
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
			"user":            userModel,
			"activeIspindels": activeIspindels,
			"beerStyles":      beerStyles,
			"formTitle":       "Nowa fermentacja",
			"submitButton":    "Rozpocznij fermentację",
			"error":           "Błąd podczas tworzenia fermentacji: " + err.Error(),
			"name":            name,
			"style_id":        styleID,
			"description":     description,
			"ispindel_id":     ispindelIDStr,
		})
		return
	}

	// Przekieruj na stronę szczegółów nowej fermentacji
	c.Redirect(http.StatusSeeOther, "/fermentations/"+strconv.Itoa(int(fermentation.ID)))
}

// FermentationDetails wyświetla szczegóły fermentacji
func (h *FermentationHandler) FermentationDetails(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{
			"error": "Nie jesteś zalogowany",
		})
		return
	}
	userModel := user.(*models.User)

	// Pobierz ID fermentacji z URL
	fermentationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator fermentacji",
			"user":  userModel,
		})
		return
	}

	// Pobierz fermentację
	fermentation, err := h.FermentationService.GetFermentation(uint(fermentationID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Nie znaleziono fermentacji",
			"user":  userModel,
		})
		return
	}

	// Pobierz informacje o urządzeniu, jeśli fermentacja ma przypisane urządzenie
	var ispindel *models.Ispindel
	if fermentation.IspindelID != nil {
		ispindel, err = h.IspindelService.GetIspindelByID(fermentation.IspindelID, userModel.ID)
		if err != nil {
			// Jeśli nie znaleziono urządzenia, nie zwracamy błędu - po prostu nie wyświetlimy informacji o urządzeniu
			log.Printf("Nie znaleziono urządzenia o ID %d: %v", *fermentation.IspindelID, err)
		}
	}

	// Pobierz pomiary z ostatnich 12 godzin dla wykresów (co godzinę)
	measurementsLast12h, err := h.FermentationService.GetHourlyMeasurementsLast12Hours(uint(fermentationID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania pomiarów",
			"user":  userModel,
		})
		return
	}

	// Sortuj pomiary chronologicznie (od najstarszych do najnowszych) dla wykresów
	services.SortMeasurementsChronologically(measurementsLast12h)

	// Pobierz wszystkie pomiary dla tabeli (tylko ostatnie 15)
	allMeasurements, err := h.FermentationService.GetAllMeasurements(uint(fermentationID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania pomiarów",
			"user":  userModel,
		})
		return
	}

	// Pobierz startowe wartości (średnia z pierwszych 3 pomiarów)
	initialGravity, initialTemperature, err := h.FermentationService.GetInitialMeasurements(uint(fermentationID))
	var initialValues gin.H
	if err == nil {
		initialValues = gin.H{
			"gravity":     initialGravity,
			"temperature": initialTemperature,
		}
	} else {
		// Jeśli nie udało się pobrać wartości początkowych, nie dodawaj ich do danych szablonu
		log.Printf("Nie udało się pobrać wartości początkowych dla fermentacji %d: %v", fermentationID, err)
	}

	// Pobierz aktualne wartości (z ostatniego pomiaru)
	var currentValues gin.H
	if len(allMeasurements) > 0 {
		lastMeasurement := allMeasurements[0] // allMeasurements jest już posortowane malejąco
		currentValues = gin.H{
			"gravity":     lastMeasurement.Gravity,
			"temperature": lastMeasurement.Temperature,
		}
	} else {
		// Jeśli nie ma pomiarów, nie dodawaj wartości aktualnych do danych szablonu
		log.Printf("Brak pomiarów dla fermentacji %d", fermentationID)
	}

	// Przygotuj dane do wykresów (ostatnie 12h, co godzinę)
	var timestamps []string
	var temperatures []float64
	var gravities []float64
	var batteries []float64
	var angles []float64
	var rssi []int

	for _, m := range measurementsLast12h {
		timestamps = append(timestamps, m.Timestamp.Format("15:04")) // Format godzinowy dla małych wykresów
		temperatures = append(temperatures, m.Temperature)
		gravities = append(gravities, m.Gravity)
		batteries = append(batteries, m.Battery)
		angles = append(angles, m.Angle)
		rssi = append(rssi, m.RSSI)
	}

	// Dodaj funkcje pomocnicze do obliczeń
	funcMap := template.FuncMap{
		"add": func(a, b float64) float64 {
			return a + b
		},
		"subtract": func(a, b float64) float64 {
			return a - b
		},
		"multiply": func(a, b float64) float64 {
			return a * b
		},
		"divide": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}

	// Przygotuj dane do szablonu
	c.HTML(http.StatusOK, "fermentation_details.html", gin.H{
		"user":          userModel,
		"fermentation":  fermentation,
		"ispindel":      ispindel,
		"duration":      h.FermentationService.GetFermentationDurationString(fermentation),
		"hasData":       len(measurementsLast12h) > 0,
		"measurements":  allMeasurements[:min(len(allMeasurements), 15)], // Pokaż tylko ostatnie 15 pomiarów w tabeli
		"timestamps":    timestamps,
		"temperatures":  temperatures,
		"gravities":     gravities,
		"batteries":     batteries,
		"angles":        angles,
		"rssi":          rssi,
		"initialValues": initialValues,
		"currentValues": currentValues,
		"canDelete":     !fermentation.IsActive || len(allMeasurements) == 0, // Można usunąć jeśli zakończona lub bez pomiarów
		"funcMap":       funcMap,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	c.Redirect(http.StatusSeeOther, "/fermentations/"+fermentationIDStr)
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

// ShowCharts wyświetla szczegółowe wykresy dla fermentacji
func (h *FermentationHandler) ShowCharts(c *gin.Context) {
	// Pobierz ID fermentacji z parametrów URL
	fermentationID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Nieprawidłowy identyfikator fermentacji",
		})
		return
	}

	// Pobierz parametr okresu, domyślnie "all"
	period := c.DefaultQuery("period", "all")

	// Pobierz użytkownika z kontekstu
	user, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}
	userModel := user.(*models.User)

	// Pobierz fermentację
	fermentation, err := h.FermentationService.GetFermentation(uint(fermentationID), userModel.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Nie znaleziono fermentacji",
			"user":  userModel,
		})
		return
	}

	// Pobierz wszystkie pomiary
	measurements, err := h.FermentationService.GetAllMeasurements(uint(fermentationID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Błąd podczas pobierania pomiarów",
			"user":  userModel,
		})
		return
	}

	// Sortuj pomiary chronologicznie (od najstarszych do najnowszych)
	services.SortMeasurementsChronologically(measurements)

	// Filtruj pomiary w zależności od wybranego okresu
	var filteredMeasurements []models.Measurement
	if period != "all" && len(measurements) > 0 {
		// Oblicz datę początkową na podstawie wybranego okresu
		now := time.Now()
		var startDate time.Time

		switch period {
		case "1d":
			startDate = now.AddDate(0, 0, -1)
		case "3d":
			startDate = now.AddDate(0, 0, -3)
		case "7d":
			startDate = now.AddDate(0, 0, -7)
		default:
			startDate = time.Time{} // Jeśli nieznany okres, domyślnie wszystkie dane
		}

		// Filtruj pomiary nowsze niż startDate
		for _, m := range measurements {
			if m.Timestamp.After(startDate) || m.Timestamp.Equal(startDate) {
				filteredMeasurements = append(filteredMeasurements, m)
			}
		}
	} else {
		filteredMeasurements = measurements
	}

	// Przygotuj dane do wykresów
	var timestamps []string
	var temperatures []float64
	var gravities []float64
	var batteries []float64
	var angles []float64
	var rssi []int

	for _, m := range filteredMeasurements {
		timestamps = append(timestamps, m.Timestamp.Format("02.01.2006 15:04"))
		temperatures = append(temperatures, m.Temperature)
		gravities = append(gravities, m.Gravity)
		batteries = append(batteries, m.Battery)
		angles = append(angles, m.Angle)
		rssi = append(rssi, m.RSSI)
	}

	// Renderuj szablon z danymi
	c.HTML(http.StatusOK, "fermentation_charts.html", gin.H{
		"user":         userModel,
		"fermentation": fermentation,
		"hasData":      len(filteredMeasurements) > 0,
		"timestamps":   timestamps,
		"temperatures": temperatures,
		"gravities":    gravities,
		"batteries":    batteries,
		"angles":       angles,
		"rssi":         rssi,
		"period":       period, // Dodajemy okres do kontekstu szablonu
	})
}
