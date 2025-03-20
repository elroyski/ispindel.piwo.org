package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func main() {
	// Test połączenia 1 - standardowy format DSN
	host := "pgsql18.mydevil.net"
	user := "p1270_ispindle"
	password := "Kochanapysia1"
	dbname := "p1270_ispindle"
	port := "5432"

	// Format 1
	dsn1 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)
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
		user, password, host, port, dbname)
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

	// Format 3 - sslmode=prefer
	dsn3 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=prefer",
		host, user, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 3):", dsn3)
	db3, err := sql.Open("postgres", dsn3)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 3:", err)
	} else {
		err = db3.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 3:", err)
		} else {
			fmt.Println("Połączenie 3 udane!")
		}
		db3.Close()
	}

	// Format 4 - bez żadnych parametrów SSL
	dsn4 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
		host, user, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 4):", dsn4)
	db4, err := sql.Open("postgres", dsn4)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 4:", err)
	} else {
		err = db4.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 4:", err)
		} else {
			fmt.Println("Połączenie 4 udane!")
		}
		db4.Close()
	}
} 