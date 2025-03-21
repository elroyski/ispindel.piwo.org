package database

import (
	"log"

	"ispindel.piwo.org/internal/models"
)

// RunMigrations - Wykonuje wszystkie migracje dla bazy danych
func RunMigrations() {
	// Automatyczna migracja tabel
	log.Println("Uruchamianie migracji bazy danych...")
	
	// Migracja tabeli users
	err := DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Błąd podczas migracji tabeli users: %v", err)
	}
	log.Println("Migracja tabeli users zakończona powodzeniem")
	
	// Migracja tabeli ispindels
	err = DB.AutoMigrate(&models.Ispindel{})
	if err != nil {
		log.Fatalf("Błąd podczas migracji tabeli ispindels: %v", err)
	}
	log.Println("Migracja tabeli ispindels zakończona powodzeniem")
	
	// Migracja tabeli measurements
	err = DB.AutoMigrate(&models.Measurement{})
	if err != nil {
		log.Fatalf("Błąd podczas migracji tabeli measurements: %v", err)
	}
	log.Println("Migracja tabeli measurements zakończona powodzeniem")
	
	log.Println("Wszystkie migracje zakończone powodzeniem")
} 