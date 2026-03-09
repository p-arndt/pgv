package main

import (
	"log"
	"os"

	"pgv/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
