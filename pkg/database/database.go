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

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	log.Printf("Próba połączenia 1: %s", dsn)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Połączenie 1 nieudane: %v", err)
		
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
		log.Printf("Próba połączenia 2: %s", dsn)
		
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Printf("Połączenie 2 nieudane: %v", err)
			
			dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable client_encoding=UTF8",
				host, port, user, password, dbname)
			log.Printf("Próba połączenia 3: %s", dsn)
			
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
			if err != nil {
				log.Printf("Połączenie 3 nieudane: %v", err)
				
				dsn = fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s",
					user, password, host, port, dbname)
				log.Printf("Próba połączenia 4: %s", dsn)
				
				db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
				if err != nil {
					log.Fatal("Wszystkie próby połączenia nieudane:", err)
				}
			}
		}
	}

	log.Println("Połączenie z bazą danych udane!")

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