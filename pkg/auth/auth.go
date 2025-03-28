package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	jwtSecret         = []byte(getEnvOrDefault("JWT_SECRET", "twoj-tajny-klucz-jwt"))
	GoogleOAuthConfig *oauth2.Config
	PiwoOAuthConfig   *oauth2.Config
)

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type PiwoUserInfo struct {
	ID      int    `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"photo_url"`
}

// InitGoogleOAuth inicjalizuje konfigurację OAuth2 dla Google
func InitGoogleOAuth() {
	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("APP_URL") + "/auth/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// InitPiwoOAuth inicjalizuje konfigurację OAuth2 dla piwo.org
func InitPiwoOAuth() {
	PiwoOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("PIWO_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("PIWO_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("PIWO_OAUTH_CALLBACK_URL"),
		Scopes: []string{
			"profile", // Zmiana z "basic" na "profile" zgodnie z IPS
			"email",   // Dodanie "email" dla dostępu do emaila użytkownika
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv("PIWO_OAUTH_AUTH_URL"),
			TokenURL: os.Getenv("PIWO_OAUTH_TOKEN_URL"),
		},
	}

	// Logowanie konfiguracji
	fmt.Printf("Inicjalizacja piwo.org OAuth:\n")
	fmt.Printf("  ClientID: %s\n", os.Getenv("PIWO_OAUTH_CLIENT_ID"))
	fmt.Printf("  RedirectURL: %s\n", os.Getenv("PIWO_OAUTH_CALLBACK_URL"))
	fmt.Printf("  AuthURL: %s\n", os.Getenv("PIWO_OAUTH_AUTH_URL"))
	fmt.Printf("  TokenURL: %s\n", os.Getenv("PIWO_OAUTH_TOKEN_URL"))
	fmt.Printf("  Scopes: %v\n", PiwoOAuthConfig.Scopes)
}

// GetGoogleUserInfo pobiera informacje o użytkowniku z Google API
func GetGoogleUserInfo(token *oauth2.Token) (*GoogleUserInfo, error) {
	client := GoogleOAuthConfig.Client(oauth2.NoContext, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("nie udało się pobrać informacji o użytkowniku: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nie udało się odczytać odpowiedzi: %v", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("nie udało się przetworzyć danych użytkownika: %v", err)
	}

	return &userInfo, nil
}

// GetPiwoUserInfo pobiera informacje o użytkowniku z piwo.org API
func GetPiwoUserInfo(token *oauth2.Token) (*PiwoUserInfo, error) {
	client := PiwoOAuthConfig.Client(oauth2.NoContext, token)

	// Logowanie dodatkowych informacji
	fmt.Printf("Próba połączenia z API piwo.org. Token: %v\n", token)

	// Zmiana URL na API core/me
	userInfoURL := "https://piwo.org/api/core/me"
	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("nie udało się pobrać informacji o użytkowniku: %v", err)
	}
	defer resp.Body.Close()

	// Logowanie odpowiedzi
	fmt.Printf("Odpowiedź z API piwo.org. Status: %d\n", resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nie udało się odczytać odpowiedzi: %v", err)
	}

	// Logowanie odpowiedzi JSON
	fmt.Printf("Odpowiedź JSON: %s\n", string(data))

	// Najpierw spróbuj rozparsować częściowo, żeby sprawdzić strukturę
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		fmt.Printf("Błąd parsowania odpowiedzi JSON jako mapy: %v\n", err)
	} else {
		fmt.Printf("Struktura odpowiedzi JSON: %+v\n", rawData)

		// Sprawdź typ pola id
		if id, ok := rawData["id"]; ok {
			fmt.Printf("ID typu: %T, wartość: %v\n", id, id)
		}
	}

	var userInfo PiwoUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("nie udało się przetworzyć danych użytkownika: %v", err)
	}

	fmt.Printf("Sparsowane dane użytkownika: %+v\n", userInfo)
	return &userInfo, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token ważny przez 24 godziny
	})

	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("nieoczekiwana metoda podpisu")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := uint(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, errors.New("nieprawidłowy token")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
