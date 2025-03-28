package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	GoogleOAuthConfig *oauth2.Config
)

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
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
