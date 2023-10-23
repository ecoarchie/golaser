package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	// startScraping(store)

	a, err := store.GetRecordByBib("99999")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("athlete = %v\n", a)

}