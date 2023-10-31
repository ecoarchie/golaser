package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type Scraper struct {
	s *PostgresStore
}

func NewScraper(store *PostgresStore) *Scraper {
	return &Scraper{
		s: store,
	}
}


type ChronoTrackURLConfig struct {
	source string
	clientID string
	eventID string
	size int
	page int
	columns string

}

func (c *ChronoTrackURLConfig) Default() *ChronoTrackURLConfig {
	source := os.Getenv("SOURCE")
	clientID := os.Getenv("CLIENT_ID")
	eventID := os.Getenv("EVENT_ID")
	size := 1000
	page := 1
	columns := fmt.Sprintf("%s,%s,%s,%s,%s,%s",
			"results_bib",
			"results_first_name",
			"results_last_name",
			"results_time",
			"results_gun_time",
			"results_race_name",
		)
		return &ChronoTrackURLConfig{
			source : source,
			clientID: clientID,
			eventID: eventID,
			size: size,
			page: page,
			columns: columns,
		}
}

func InitURL (opts *ChronoTrackURLConfig) string {
	return fmt.Sprintf("%s/%s/results?client_id=%s&size=%d&page=%d&columns=%s",
		opts.source,
		opts.eventID,
		opts.clientID,
		opts.size,
		opts.page,
		opts.columns)
}

func getTotalPagesForRequest(pageSize int) int {
	basicHash := os.Getenv("HASH")
	authHeader := fmt.Sprintf("Basic %s", basicHash)
	

	config := new(ChronoTrackURLConfig).Default()
	config.size = 1
	config.page = 1
	url := InitURL(config)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Fail to make request", err)
	}
	req.Header.Add("Authorization", authHeader)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Fail to make request", err)
	}
	defer resp.Body.Close()

	rowQty := resp.Header.Get("x-ctlive-row-count")
	rows, err := strconv.Atoi(rowQty)
	if err != nil {
		log.Fatal(err)
	}
	return rows / pageSize + 1

}

func (scraper *Scraper) StartScraping() {
	config := new(ChronoTrackURLConfig).Default()
	pageQty := getTotalPagesForRequest(config.size)
	// pageQty := 10
	fmt.Printf("pageQty = %d\n", pageQty)

	wg := &sync.WaitGroup{}
	for i := 1; i <= pageQty; i++ {
		config.page = i
		url := InitURL(config)
		wg.Add(1)
		go scrape(url, scraper.s, wg)
	}
	wg.Wait()
}

func (scraper *Scraper) StartPartialScraping() {
	config := new(ChronoTrackURLConfig).Default()
	pageQty := getTotalPagesForRequest(config.size)
	fmt.Printf("pageQty = %d\n", pageQty)

	fromPage := scraper.s.GetRecordsCount() / config.size + 1
	wg := &sync.WaitGroup{}
	for i := fromPage; i <= pageQty; i++ {
		config.page = i
		url := InitURL(config)
		wg.Add(1)
		go scrape(url, scraper.s, wg)
	}
	wg.Wait()
}

func scrape(url string, s *PostgresStore, wg *sync.WaitGroup) {
	defer wg.Done()

	basicHash := os.Getenv("HASH")
	authHeader := fmt.Sprintf("Basic %s", basicHash)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Fail to make request", err)
	}

	req.Header.Add("Authorization", authHeader)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Fail to make request", err)
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Fail to read body", err)
	}

	res := Response{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		log.Fatal("cannot parse json", err)
	}

	pageNum := resp.Header.Get("X-Ctlive-Current-Page")
	fmt.Printf("Qty of records = %d on page %s\n", len(res.EventResults), pageNum)

	for i := 0; i < len(res.EventResults); i++ {
		s.CreateRecord(&res.EventResults[i])
	}
}
