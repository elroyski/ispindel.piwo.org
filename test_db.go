package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func main() {
	// Test połączenia 1 - standardowy format DSN
	host := "pgsql18.mydevil.net"
	user1 := "p1270_ispindle"
	user2 := "p1270"    // Alternatywna nazwa użytkownika
	password := "Kochanapysia1"
	dbname := "p1270_ispindle"
	port := "5432"
	
	// Format 1
	dsn1 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user1, password, dbname, port)
	fmt.Println("Próba połączenia (format 1):", dsn1)
	db1, err := sql.Open("postgres", dsn1)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 1:", err)
	} else {
		err = db1.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 1:", err)
		} else {
			fmt.Println("Połączenie 1 udane!")
		}
		db1.Close()
	}
	
	// Format 2
	dsn2 := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		user1, password, host, port, dbname)
	fmt.Println("\nPróba połączenia (format 2):", dsn2)
	db2, err := sql.Open("postgres", dsn2)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 2:", err)
	} else {
		err = db2.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 2:", err)
		} else {
			fmt.Println("Połączenie 2 udane!")
		}
		db2.Close()
	}
	
	// Format 9 - użytkownik p1270
	dsn9 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user2, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 9 - użytkownik p1270):", dsn9)
	db9, err := sql.Open("postgres", dsn9)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 9:", err)
	} else {
		err = db9.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 9:", err)
		} else {
			fmt.Println("Połączenie 9 udane!")
		}
		db9.Close()
	}
	
	// Format 10 - użytkownik p1270, URL style
	dsn10 := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		user2, password, host, port, dbname)
	fmt.Println("\nPróba połączenia (format 10 - URL z użytkownikiem p1270):", dsn10)
	db10, err := sql.Open("postgres", dsn10)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 10:", err)
	} else {
		err = db10.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 10:", err)
		} else {
			fmt.Println("Połączenie 10 udane!")
		}
		db10.Close()
	}
} 