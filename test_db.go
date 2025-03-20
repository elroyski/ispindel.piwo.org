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
	
	// Format 5 - sslmode=require
	dsn5 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require",
		host, user, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 5):", dsn5)
	db5, err := sql.Open("postgres", dsn5)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 5:", err)
	} else {
		err = db5.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 5:", err)
		} else {
			fmt.Println("Połączenie 5 udane!")
		}
		db5.Close()
	}
	
	// Format 6 - search_path=public
	dsn6 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require search_path=public",
		host, user, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 6):", dsn6)
	db6, err := sql.Open("postgres", dsn6)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 6:", err)
	} else {
		err = db6.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 6:", err)
		} else {
			fmt.Println("Połączenie 6 udane!")
		}
		db6.Close()
	}
	
	// Format 7 - inna nazwa użytkownika
	altUser := "piwo_p1270_ispindle"
	dsn7 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require",
		host, altUser, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 7 - alternatywna nazwa użytkownika):", dsn7)
	db7, err := sql.Open("postgres", dsn7)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 7:", err)
	} else {
		err = db7.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 7:", err)
		} else {
			fmt.Println("Połączenie 7 udane!")
		}
		db7.Close()
	}
	
	// Format 8 - inna nazwa użytkownika, bez SSL
	dsn8 := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, altUser, password, dbname, port)
	fmt.Println("\nPróba połączenia (format 8 - alternatywna nazwa użytkownika, bez SSL):", dsn8)
	db8, err := sql.Open("postgres", dsn8)
	if err != nil {
		fmt.Println("Błąd otwarcia połączenia 8:", err)
	} else {
		err = db8.Ping()
		if err != nil {
			fmt.Println("Błąd ping połączenia 8:", err)
		} else {
			fmt.Println("Połączenie 8 udane!")
		}
		db8.Close()
	}
} 