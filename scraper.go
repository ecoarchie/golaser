package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Scraper struct {
	store Storage
	config ChronoTrackURLConfig
}

func NewScraper(store Storage) *Scraper {
	return &Scraper{
		store: store,
		config: *new(ChronoTrackURLConfig),
	}
}

// func NewScraperSqlite(store *SqliteStore) *Scraper {
// 	return &Scraper{
// 		s: store,
// 	}
// }

type ChronoTrackURLConfig struct {
	source string
	clientID string
	eventID string
	size int
	page int
	columns string
	authHeader string
}

func (c *ChronoTrackURLConfig) Default(login string, password string, clientID string, eventID string) *ChronoTrackURLConfig {
	source := "https://api.chronotrack.com/api/event.json"
	size := 1000
	page := 1
	strToHash := fmt.Sprintf("%s:%s", login, password)
	hash := base64.StdEncoding.EncodeToString([]byte(strToHash))
	authHeader := fmt.Sprintf("Basic %s", hash)
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
			authHeader: authHeader,
		}
}

func ResultsURL (opts ChronoTrackURLConfig) string {
	return fmt.Sprintf("%s/%s/results?client_id=%s&size=%d&page=%d&columns=%s",
		opts.source,
		opts.eventID,
		opts.clientID,
		opts.size,
		opts.page,
		opts.columns)
}

func EventInfoURL (opts ChronoTrackURLConfig) string {
	return fmt.Sprintf("%s/%s?client_id=%s",
		opts.source,
		opts.eventID,
		opts.clientID,
	)
}

func (scraper *Scraper) CheckEventURL() (*Event, error) {
	config := scraper.config
	fmt.Printf("config = %+v\n", config)
	url := EventInfoURL(config)
	fmt.Printf("url = %s\n", url)

	authHeader := config.authHeader

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authHeader)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("проверьте правильность clientID.\n%s", resp.Status) 
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := EventInfoResp{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, fmt.Errorf("неверно указан ID соревнования")
	}
	fmt.Printf("responsse = %+v\n", res)
	return &res.Event, nil
}

func (scraper *Scraper) getTotalPagesForRequest(pageSize int) int {
	config := scraper.config
	config.size = 1
	config.page = 1
	url := ResultsURL(config)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Fail to make request", err)
	}
	req.Header.Add("Authorization", config.authHeader)
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
	config := scraper.config
	pageQty := scraper.getTotalPagesForRequest(config.size)
	// pageQty := 10
	fmt.Printf("pageQty = %d\n", pageQty)

	wg := &sync.WaitGroup{}
	for i := 1; i <= pageQty; i++ {
		config.page = i
		url := ResultsURL(config)
		wg.Add(1)
		go scraper.scrape(url, wg)
	}
	wg.Wait()
}

func (scraper *Scraper) StartPartialScraping() int {
	config := scraper.config
	pageQty := scraper.getTotalPagesForRequest(config.size)
	fmt.Printf("pageQty = %d\n", pageQty)

	countBefore := scraper.store.GetRecordsCount()
	fromPage := countBefore / config.size + 1
	wg := &sync.WaitGroup{}
	for i := fromPage; i <= pageQty; i++ {
		config.page = i
		url := ResultsURL(config)
		wg.Add(1)
		go scraper.scrape(url, wg)
	}
	wg.Wait()
	return scraper.store.GetRecordsCount() - countBefore
}

func (scraper *Scraper) scrape(url string, wg *sync.WaitGroup) {
	// start := time.Now()
	defer wg.Done()

	authHeader := scraper.config.authHeader

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
		scraper.store.CreateRecord(&res.EventResults[i])
	}
	// s.CreateBulkRecords(&res.EventResults)
	scraper.store.Checkpoint()
	// fmt.Printf("Time past = %v\n", time.Since(start))
}
