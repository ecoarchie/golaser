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

func startScraping(s *PostgresStore) {

	source := os.Getenv("SOURCE")
	clientID := os.Getenv("CLIENT_ID")
	eventID := os.Getenv("EVENT_ID")
	basicHash := os.Getenv("HASH")
	authHeader := fmt.Sprintf("Basic %s", basicHash)

	size := 1000
	page := 1
	columns := fmt.Sprintf("%s,%s,%s,%s,%s",
			"results_bib",
			"results_first_name",
			"results_last_name",
			"results_time",
			"results_gun_time",
		)
	url := fmt.Sprintf("%s/%s/results?client_id=%s&size=%d&page=%d&columns=%s", source, eventID, clientID, 1, page, columns)
	// fmt.Printf("%s",url)

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
	pageQty := rows / size + 1
	fmt.Printf("pageQty = %d\n", pageQty)

	wg := &sync.WaitGroup{}
	for i := 1; i <= pageQty; i++ {
		url := fmt.Sprintf("%s/%s/results?client_id=%s&size=%d&page=%d&columns=%s", source, eventID, clientID, size, i, columns)
		wg.Add(1)
		go scrape(url, s, wg)
	}
	wg.Wait()
	// data, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatal("Fail to read body", err)
	// }

	// res := Response{}
	// err = json.Unmarshal(data, &res)
	// if err != nil {
	// 	log.Fatal("cannot parse json", err)
	// }
	// fmt.Printf("Qty of records = %d\n", len(res.EventResults))
	// fmt.Printf("Records = %+v\n", res.EventResults)

}

func scrape(url string, s *PostgresStore, wg *sync.WaitGroup) {
	defer wg.Done()

	// fmt.Printf("%s\n", url)
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
	// fmt.Printf("Records = %+v\n", res.EventResults)
	for i := 0; i < len(res.EventResults); i++ {
		s.CreateRecord(&res.EventResults[i])
	}
}