package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Parametry połączenia
	host := "mysql18.mydevil.net"
	user := "m1270_ispindel"
	password := "Kochanapysia1"
	dbname := "m1270_ispindel"
	port := "3306"

	// Format DSN dla MySQL: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)
	fmt.Println("Próba połączenia MySQL:", dsn)
	
	// Otwieramy połączenie
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia:", err)
		return
	}
	defer db.Close()
	
	// Sprawdzamy, czy połączenie działa
	err = db.Ping()
	if err != nil {
		fmt.Println("Błąd ping:", err)
		return
	}
	
	fmt.Println("Połączenie z MySQL udane!")
} 