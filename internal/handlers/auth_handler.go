package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ispindel.piwo.org/internal/services"
)

type AuthHandler struct {
	userService *services.UserService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		userService: services.NewUserService(),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "register.html", nil)
		return
	}

	// Pobierz dane z formularza
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")

	// Walidacja danych
	if name == "" || email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Wszystkie pola są wymagane",
		})
		return
	}

	if password != passwordConfirm {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Hasła nie są identyczne",
		})
		return
	}

	if len(password) < 8 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": "Hasło musi mieć co najmniej 8 znaków",
		})
		return
	}

	// Pobierz IP użytkownika
	ip := c.ClientIP()
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		ip = strings.Split(forwardedFor, ",")[0]
	}

	// Zarejestruj użytkownika
	err := h.userService.Register(name, email, password, ip)
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	// Przekieruj do strony logowania z informacją o konieczności aktywacji
	c.Redirect(http.StatusSeeOther, "/auth/login?registered=true")
}

func (h *AuthHandler) Login(c *gin.Context) {
	if c.Request.Method == "GET" {
		registered := c.Query("registered") == "true"
		activation := c.Query("activation") == "true"
		c.HTML(http.StatusOK, "login.html", gin.H{
			"registered": registered,
			"activation": activation,
		})
		return
	}

	// Pobierz dane z formularza
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Walidacja danych
	if email == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Wszystkie pola są wymagane",
		})
		return
	}

	// Pobierz IP użytkownika
	ip := c.ClientIP()
	if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		ip = strings.Split(forwardedFor, ",")[0]
	}

	// Zaloguj użytkownika
	token, err := h.userService.Login(email, password, ip)
	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	// Ustaw token w ciasteczku
	c.SetCookie("token", token, 3600*24, "/", "", false, true)

	// Przekieruj do strony głównej
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}

func (h *AuthHandler) Activate(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "activation.html", gin.H{
			"error": "Brak tokenu aktywacyjnego",
		})
		return
	}

	err := h.userService.ActivateAccount(token)
	if err != nil {
		c.HTML(http.StatusBadRequest, "activation.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "activation.html", gin.H{
		"success": "Konto zostało pomyślnie aktywowane! Możesz się teraz zalogować.",
	})
}

func (h *AuthHandler) ResendActivation(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "resend_activation.html", nil)
		return
	}

	email := c.PostForm("email")
	if email == "" {
		c.HTML(http.StatusBadRequest, "resend_activation.html", gin.H{
			"error": "Adres email jest wymagany",
		})
		return
	}

	err := h.userService.ResendActivationEmail(email)
	if err != nil {
		c.HTML(http.StatusBadRequest, "resend_activation.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "resend_activation.html", gin.H{
		"success": "Email aktywacyjny został wysłany ponownie. Sprawdź swoją skrzynkę pocztową.",
	})
} 