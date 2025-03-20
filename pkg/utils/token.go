package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

// GenerateToken tworzy losowy token o określonej długości
func GenerateToken(length int) string {
	// Oblicz ile bajtów potrzebujemy dla tokenów o podanej długości
	// Każdy bajt to 2 znaki hex, więc potrzebujemy length/2 bajtów
	bytes := make([]byte, length/2)
	
	// Wypełnij bajtami losowymi
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Błąd generowania tokenu: %v", err)
		return ""
	}
	
	// Konwersja na string hex
	return hex.EncodeToString(bytes)
}

// GenerateActivationToken tworzy standardowy token aktywacyjny
func GenerateActivationToken() string {
	return GenerateToken(64) // 64 znaki (32 bajty)
}

// GeneratePasswordResetToken tworzy token do resetowania hasła
func GeneratePasswordResetToken() string {
	return GenerateToken(64) // 64 znaki (32 bajty)
} 