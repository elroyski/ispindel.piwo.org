package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ispindel.piwo.org/internal/models"
	"ispindel.piwo.org/internal/services"
	"ispindel.piwo.org/pkg/auth"
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
	passwordConfirm := c.PostForm("confirm_password")

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

	// Przekierowanie do strony logowania z parametrem informującym o pomyślnej aktywacji
	c.Redirect(http.StatusSeeOther, "/auth/login?activation=true")
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

// GoogleLogin rozpoczyna proces logowania przez Google
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	url := auth.GoogleOAuthConfig.AuthCodeURL("state")
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback obsługuje odpowiedź od Google po udanym logowaniu
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	token, err := auth.GoogleOAuthConfig.Exchange(c, code)
	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Nie udało się zalogować przez Google",
		})
		return
	}

	userInfo, err := auth.GetGoogleUserInfo(token)
	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Nie udało się pobrać informacji o użytkowniku",
		})
		return
	}

	// Sprawdź czy użytkownik już istnieje
	user, err := h.userService.GetUserByGoogleID(userInfo.ID)
	if err != nil {
		// Użytkownik nie istnieje - sprawdź czy istnieje konto z tym samym emailem
		user, err = h.userService.GetUserByEmail(userInfo.Email)
		if err != nil {
			// Utwórz nowego użytkownika
			user = &models.User{
				Name:     userInfo.Name,
				Email:    userInfo.Email,
				GoogleID: userInfo.ID,
				Picture:  userInfo.Picture,
				IsActive: true,
			}
			if err := h.userService.CreateUser(user); err != nil {
				c.HTML(http.StatusInternalServerError, "login.html", gin.H{
					"error": "Nie udało się utworzyć konta",
				})
				return
			}
		} else {
			// Połącz istniejące konto z Google
			user.GoogleID = userInfo.ID
			user.Picture = userInfo.Picture
			if err := h.userService.UpdateUser(user); err != nil {
				c.HTML(http.StatusInternalServerError, "login.html", gin.H{
					"error": "Nie udało się połączyć konta z Google",
				})
				return
			}
		}
	}

	// Wygeneruj token JWT
	jwtToken, err := auth.GenerateToken(user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error": "Nie udało się wygenerować tokenu",
		})
		return
	}

	// Ustaw cookie z tokenem
	c.SetCookie("token", jwtToken, 3600*24, "/", "", false, true)

	// Przekieruj na dashboard
	c.Redirect(http.StatusSeeOther, "/dashboard")
}

// PiwoLogin rozpoczyna proces logowania przez piwo.org
func (h *AuthHandler) PiwoLogin(c *gin.Context) {
	url := auth.PiwoOAuthConfig.AuthCodeURL("state")
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// PiwoCallback obsługuje odpowiedź od piwo.org po udanym logowaniu
func (h *AuthHandler) PiwoCallback(c *gin.Context) {
	// Sprawdź, czy wystąpił błąd w parametrach URL
	errorParam := c.Query("error")
	if errorParam != "" {
		errorDesc := c.Query("error_description")
		fmt.Printf("Otrzymano błąd OAuth: %s, opis: %s\n", errorParam, errorDesc)
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": fmt.Sprintf("Błąd autoryzacji piwo.org: %s (%s)", errorParam, errorDesc),
		})
		return
	}

	code := c.Query("code")
	fmt.Printf("Otrzymany kod autoryzacyjny: %s\n", code)

	// Sprawdź czy kod jest pusty
	if code == "" {
		fmt.Printf("Pusty kod autoryzacyjny\n")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Nie otrzymano kodu autoryzacyjnego od piwo.org",
		})
		return
	}

	token, err := auth.PiwoOAuthConfig.Exchange(c, code)
	if err != nil {
		fmt.Printf("Błąd wymiany kodu na token: %v\n", err)

		// Zwróć HTML z błędem
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Nie udało się zalogować przez piwo.org: " + err.Error(),
		})
		return
	}

	fmt.Printf("Otrzymany token: %v\n", token)

	userInfo, err := auth.GetPiwoUserInfo(token)
	if err != nil {
		fmt.Printf("Błąd pobierania informacji o użytkowniku: %v\n", err)

		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error": "Nie udało się pobrać informacji o użytkowniku: " + err.Error(),
		})
		return
	}

	// Sprawdź czy użytkownik już istnieje przez PiwoID
	user, err := h.userService.GetUserByPiwoID(userInfo.ID)
	if err != nil {
		// Użytkownik nie istnieje - sprawdź czy istnieje konto z tym samym emailem
		user, err = h.userService.GetUserByEmail(userInfo.Email)
		if err != nil {
			// Utwórz nowego użytkownika
			user = &models.User{
				Name:     userInfo.Name,
				Email:    userInfo.Email,
				PiwoID:   userInfo.ID,
				Picture:  userInfo.Picture,
				IsActive: true,
			}
			if err := h.userService.CreateUser(user); err != nil {
				c.HTML(http.StatusInternalServerError, "login.html", gin.H{
					"error": "Nie udało się utworzyć konta",
				})
				return
			}
		} else {
			// Połącz istniejące konto z piwo.org
			user.PiwoID = userInfo.ID
			user.Picture = userInfo.Picture
			if err := h.userService.UpdateUser(user); err != nil {
				c.HTML(http.StatusInternalServerError, "login.html", gin.H{
					"error": "Nie udało się połączyć konta z piwo.org",
				})
				return
			}
		}
	}

	// Wygeneruj token JWT
	jwtToken, err := auth.GenerateToken(user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error": "Nie udało się wygenerować tokenu",
		})
		return
	}

	// Aktualizuj datę ostatniego logowania
	if err := h.userService.UpdateLastLogin(user.ID); err != nil {
		// Logujemy błąd, ale kontynuujemy proces logowania
		fmt.Printf("Błąd aktualizacji daty logowania: %v\n", err)
	}

	// Ustaw cookie z tokenem
	c.SetCookie("token", jwtToken, 3600*24, "/", "", false, true)

	// Przekieruj na dashboard
	c.Redirect(http.StatusSeeOther, "/dashboard")
}
