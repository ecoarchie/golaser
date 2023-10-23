package main

type Response struct {
	EventResults []Athlete `json:"event_results"`
}

type Athlete struct {
		ID string
		ResultsBib string `json:"results_bib"`
		ResultsFirstName string `json:"results_first_name"`
		ResultsLastsName string `json:"results_last_name"`
		ResultsTime string `json:"results_time"`
		ResultsGunTime string `json:"results_gun_time"`
}
