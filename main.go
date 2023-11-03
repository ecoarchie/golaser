package main

import (
	"log"
	// "github.com/joho/godotenv"
)

func main() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file", err)
	// }

	// store, err := NewPostgresStore()
	store, err := NewSqliteStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	// startScraping(store)
	// startPartialScraping(store)

	newScraper := NewScraper(store)
	// newScraper := new(Scraper)
	// newScraper := NewScraperPsql(store)
	
	server := NewAPIServer(":3000", store, *newScraper)
	server.Run()
}