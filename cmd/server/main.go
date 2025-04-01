package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"ispindel.piwo.org/internal/handlers"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
	"ispindel.piwo.org/pkg/auth"
	"ispindel.piwo.org/pkg/database"
	"ispindel.piwo.org/pkg/mailer"
)

func main() {
	// Inicjalizacja bazy danych
	database.InitDB()

	// Inicjalizacja mailera
	mailer.InitMailer()

	// Inicjalizacja Google OAuth
	auth.InitGoogleOAuth()

	// Inicjalizacja piwo.org OAuth
	auth.InitPiwoOAuth()

	// Inicjalizacja routera Gin
	r := gin.Default()

	// Dodanie funkcji do konwersji danych na JSON
	r.SetFuncMap(template.FuncMap{
		"jsonify": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
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
	})

	// Konfiguracja szablonów HTML - musi być po SetFuncMap
	r.LoadHTMLGlob("web/templates/*")

	// Konfiguracja statycznych plików
	r.Static("/static", "./web/static")

	// Inicjalizacja serwisów
	userService := services.NewUserService()
	ispindelService := services.NewIspindelService()
	fermentationService := services.NewFermentationService()

	// Middleware do sprawdzania autentykacji
	authMiddleware := func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			c.Next()
			return
		}

		userID, err := auth.ValidateToken(token)
		if err != nil {
			c.SetCookie("token", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		user, err := userService.GetUserByID(userID)
		if err != nil {
			c.SetCookie("token", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		c.Set("user", user)

		// Sprawdź, czy użytkownik jest administratorem
		isAdmin := user.Email == "elroyski@gmail.com"
		c.Set("isAdmin", isAdmin)

		c.Next()
	}

	// Inicjalizacja handlerów
	authHandler := handlers.NewAuthHandler()
	ispindelHandler := handlers.NewIspindelHandler()
	fermentationHandler := handlers.NewFermentationHandler()
	settingsHandler := handlers.NewSettingsHandler()
	adminHandler := handlers.NewAdminHandler(userService, ispindelService, fermentationService)

	// Użyj middleware'a dla wszystkich routów
	r.Use(authMiddleware)

	// Grupa routów dla autentykacji
	auth := r.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)
		auth.POST("/login", authHandler.Login)
		auth.GET("/register", authHandler.Register)
		auth.POST("/register", authHandler.Register)
		auth.GET("/logout", authHandler.Logout)
		auth.GET("/activate", authHandler.Activate)
		auth.GET("/resend-activation", func(c *gin.Context) {
			c.HTML(http.StatusOK, "resend_activation.html", gin.H{})
		})
		auth.POST("/resend-activation", authHandler.ResendActivation)
		auth.GET("/google/login", authHandler.GoogleLogin)
		auth.GET("/google/callback", authHandler.GoogleCallback)
		auth.GET("/piwo/login", authHandler.PiwoLogin)
		auth.GET("/piwo/callback", authHandler.PiwoCallback)
	}

	// Grupa routów dla zarządzania urządzeniami iSpindel
	ispindels := r.Group("/ispindels")
	{
		ispindels.GET("/", ispindelHandler.ListIspindels)
		ispindels.GET("/new", ispindelHandler.NewIspindelForm)
		ispindels.POST("/new", ispindelHandler.CreateIspindel)
		ispindels.GET("/:id", ispindelHandler.IspindelDetails)
		ispindels.GET("/:id/edit", ispindelHandler.EditIspindelForm)
		ispindels.POST("/:id/edit", ispindelHandler.UpdateIspindel)
		ispindels.POST("/:id/regenerate-key", ispindelHandler.RegenerateAPIKey)
		ispindels.POST("/:id/delete", ispindelHandler.DeleteIspindel)
	}

	// Grupa dla zarządzania fermentacjami (wymaga autoryzacji)
	fermentationGroup := r.Group("/fermentations")
	fermentationGroup.Use(func(c *gin.Context) {
		_, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusSeeOther, "/auth/login")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		fermentationGroup.GET("", fermentationHandler.FermentationsList)
		fermentationGroup.GET("/new", fermentationHandler.NewFermentationForm)
		fermentationGroup.POST("/new", fermentationHandler.CreateFermentation)
		fermentationGroup.GET("/:id", fermentationHandler.FermentationDetails)
		fermentationGroup.GET("/:id/charts", fermentationHandler.ShowCharts)
		fermentationGroup.POST("/:id/end", fermentationHandler.EndFermentation)
		fermentationGroup.POST("/:id/delete", fermentationHandler.DeleteFermentation)
	}

	// API endpoint dostępny bez autentykacji
	r.POST("/api/ispindel/:api_key", ispindelHandler.ReceiveData)
	// Alternatywny endpoint (bez prefiksu /api/) dla kompatybilności z różnymi wersjami firmware
	r.POST("/ispindel/:api_key", ispindelHandler.ReceiveData)
	// Endpoint dla iSpindel, który wysyła dane bez klucza API w ścieżce
	r.POST("/api/ispindel", ispindelHandler.ReceiveDataNoAPIKey)

	// Strona główna
	r.GET("/", func(c *gin.Context) {
		_, exists := c.Get("user")
		if exists {
			// Przekieruj zalogowanego użytkownika do dashboard
			c.Redirect(http.StatusSeeOther, "/dashboard")
			return
		}
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Panel główny (chroniony)
	r.GET("/dashboard", func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusSeeOther, "/auth/login")
			return
		}

		// Sprawdzenie czy użytkownik jest administratorem
		userModel := user.(*models.User)
		isAdmin := userModel.Email == adminHandler.AdminEmail

		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"user":    user,
			"isAdmin": isAdmin,
		})
	})

	// Ustawienia konta i systemu (chronione)
	settingsGroup := r.Group("/settings")
	settingsGroup.Use(func(c *gin.Context) {
		_, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusSeeOther, "/auth/login")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		settingsGroup.GET("", settingsHandler.Settings)
		settingsGroup.POST("/change-password", settingsHandler.ChangePassword)
		settingsGroup.POST("/delete-account", settingsHandler.DeleteAccount)
	}

	// Panel administratora (tylko dla administratora)
	adminGroup := r.Group("/admin")
	adminGroup.Use(adminHandler.AdminRequired())
	{
		adminGroup.GET("", adminHandler.Dashboard)
		adminGroup.GET("/users", adminHandler.ListUsers)
		adminGroup.GET("/users/:id", adminHandler.UserDetails)
		adminGroup.GET("/ispindels", adminHandler.ListIspindels)
		adminGroup.GET("/fermentations", adminHandler.ListFermentations)
		adminGroup.POST("/ispindels/:id/delete", adminHandler.AdminDeleteIspindel)
	}

	// Pobierz port z zmiennej środowiskowej lub ustaw domyślną wartość
	port := os.Getenv("PORT")
	if port == "" {
		port = "49330" // Domyślny port
		log.Println("Używam domyślnego portu:", port)
	} else {
		log.Println("Używam portu z zmiennej środowiskowej:", port)
	}

	// Ustaw zmienną środowiskową APP_URL jeśli nie jest ustawiona
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://ispindel.piwo.org"
		os.Setenv("APP_URL", appURL)
		log.Println("Ustawiam domyślny URL aplikacji:", appURL)
	} else {
		log.Println("Używam URL aplikacji z zmiennej środowiskowej:", appURL)
	}

	// Dodatkowe sprawdzenie zmiennej APP_URL
	log.Printf("Ostateczna wartość APP_URL: %s", os.Getenv("APP_URL"))

	// Uruchomienie serwera
	log.Printf("Uruchamianie serwera na porcie :%s...", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		panic(err)
	}
}
