package main

import (
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
	// startPartialScraping(store)

	newScraper := NewScraper(store)
	
	server := NewAPIServer(":3000", store, *newScraper)
	server.Run()

	// a, err := store.GetRecordByBib("99999")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("athlete = %v\n", a)

}