package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"ispindel.piwo.org/internal/models"
)

var DB *gorm.DB

func InitDB() {
	host := getEnvOrDefault("DB_HOST", "pgsql18.mydevil.net")
	user := getEnvOrDefault("DB_USER", "p1270_ispindle")
	password := getEnvOrDefault("DB_PASSWORD", "Kochanapysia1")
	dbname := getEnvOrDefault("DB_NAME", "p1270_ispindle")
	port := getEnvOrDefault("DB_PORT", "5432")

	log.Printf("Próba połączenia z bazą danych:")
	log.Printf("Host: %s", host)
	log.Printf("User: %s", user)
	log.Printf("Database: %s", dbname)
	log.Printf("Port: %s", port)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable connect_timeout=10 application_name=ispindel",
		host, user, password, dbname, port)

	log.Printf("DSN: %s", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Nie udało się połączyć z bazą danych:", err)
	}

	// Automatyczna migracja schematu
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("Nie udało się zmigrować schematu bazy danych:", err)
	}

	DB = db
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		log.Printf("Używam zmiennej środowiskowej %s: %s", key, value)
		return value
	}
	log.Printf("Używam wartości domyślnej dla %s: %s", key, defaultValue)
	return defaultValue
} 