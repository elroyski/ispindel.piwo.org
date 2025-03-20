package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"ispindel.piwo.org/internal/models"
)

var DB *gorm.DB

func InitDB() {
	host := getEnvOrDefault("DB_HOST", "mysql18.mydevil.net")
	user := getEnvOrDefault("DB_USER", "m1270_ispindel")
	password := getEnvOrDefault("DB_PASSWORD", "Kochanapysia1")
	dbname := getEnvOrDefault("DB_NAME", "m1270_ispindel")
	port := getEnvOrDefault("DB_PORT", "3306")

	log.Printf("Próba połączenia z bazą danych MySQL:")
	log.Printf("Host: %s", host)
	log.Printf("User: %s", user)
	log.Printf("Database: %s", dbname)
	log.Printf("Port: %s", port)

	// Format DSN dla MySQL: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)
	log.Printf("Próba połączenia MySQL: %s", dsn)
	
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Nie udało się połączyć z bazą danych MySQL:", err)
	}

	log.Println("Połączenie z bazą danych MySQL udane!")

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