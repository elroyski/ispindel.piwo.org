package main

import (
	"fmt"
	"net/http"
	"os"
	"log"

	"github.com/gin-gonic/gin"
	"ispindel.piwo.org/internal/handlers"
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

	// Inicjalizacja routera Gin
	r := gin.Default()

	// Konfiguracja szablonów HTML
	r.LoadHTMLGlob("web/templates/*")

	// Konfiguracja statycznych plików
	r.Static("/static", "./web/static")

	// Inicjalizacja serwisów
	userService := services.NewUserService()

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
		c.Next()
	}

	// Inicjalizacja handlerów
	authHandler := handlers.NewAuthHandler()
	ispindelHandler := handlers.NewIspindelHandler()

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

	// API endpoint dostępny bez autentykacji
	r.POST("/api/ispindel/:api_key", ispindelHandler.ReceiveData)
	// Alternatywny endpoint (bez prefiksu /api/) dla kompatybilności z różnymi wersjami firmware
	r.POST("/ispindel/:api_key", ispindelHandler.ReceiveData)

	// Strona główna
	r.GET("/", func(c *gin.Context) {
		user, _ := c.Get("user")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"user": user,
		})
	})

	// Panel główny (chroniony)
	r.GET("/dashboard", func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusSeeOther, "/auth/login")
			return
		}
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"user": user,
		})
	})

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