package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <password>")
	}

	password := os.Args[1]

	// Generar hash con costo 10 (mismo que usa el sistema)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", string(hash))

	// Verificar que el hash funciona
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		log.Fatal("Hash verification failed:", err)
	}

	fmt.Println("Hash verification: OK")
}
