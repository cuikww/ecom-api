package main

import (
	"log"

	"ecom-api/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found")
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
