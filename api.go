package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)


type APIServer struct {
	listenAddr string
	store Storage
	scraper Scraper
}

func NewAPIServer(listenAddr string, store Storage, scraper Scraper) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store: store,
		scraper: scraper,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/index", s.handleIndexPage)
	router.HandleFunc("/search", s.handleSearchBib)
	router.HandleFunc("/archive", s.HandleArchiveRecord)
	router.HandleFunc("/pupdate", s.HandlePartialDBUpdate)
	router.HandleFunc("/history", s.HandleDeleteHistory)

	log.Println("JSON API server running on port: ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	records, err := s.store.GetHistoryRecords()

	if err != nil {
		fmt.Println("error", err)
	}
	templ := template.Must(template.ParseFiles("index.html"))

	data := map[string][]*Athlete{
		"Records": {},
	}
	data["Records"] = records
	fmt.Printf("%+v\n", data)
	templ.Execute(w, data)
}

func (s *APIServer) handleSearchBib(w http.ResponseWriter, r *http.Request) {
	bib := r.PostFormValue("bib")
	a, err := s.store.GetRecordByBib(bib)
	// fmt.Printf("record = %v\n", a)
	if err != nil {
		fmt.Println("error", err)
	}
	// fmt.Printf("%+v\n", a)
	var htmlStr string
	if a == nil {
		htmlStr = fmt.Sprintf(`
			<tr class='table-danger'>
				<th scope='row'>%s</th>
				<td>Участник не найден</td>
				<td>-</td>
			</tr>
			`, bib)

	} else {
		htmlStr = fmt.Sprintf(`
			<tr class='table-success'>
				<th scope='row'>%s</th>
				<td>%s %s</td>
				<td>%s</td>
			</tr>
			`, a.ResultsBib, a.ResultsFirstName, a.ResultsLastName, a.ResultsTime)
		w.Header().Add("HX-Trigger", "found")
	}
	templ, _ := template.New("t").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandleArchiveRecord(w http.ResponseWriter, r *http.Request) {
	a, err := s.store.GetLatestHistoryRecord()
	if err != nil {
		fmt.Println("error", err)
	}
	// fmt.Printf("%+v\n", a)
	htmlStr := fmt.Sprintf(`
          <tr class='table-secondary'>
            <th scope='row'>%s</th>
            <td>%s %s</td>
            <td>%s</td>
          </tr>
	`, a.ResultsBib, a.ResultsFirstName, a.ResultsLastName, a.ResultsTime)
	templ, _ := template.New("a").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandlePartialDBUpdate(w http.ResponseWriter, r *http.Request) {
	countBefore := s.store.GetRecordsCount()
	s.scraper.StartPartialScraping()
	countAfter := s.store.GetRecordsCount()
	totalAdded := countAfter - countBefore
	updTime := time.Now().Local().Truncate(time.Second)
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-info" role="alert">
		<h4 class="alert-heading">База данных обновлена!</h4>
		<p>Добавлено %d записей в %s.</p>
		</div>
	`, totalAdded, updTime)
	templ, _ := template.New("count").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandleDeleteHistory(w http.ResponseWriter, r *http.Request) {
	s.store.ClearHistory()
	// http.Redirect(w, r, "/index", http.StatusSeeOther)
}

// decorator to decorate all apiFuncs to HandleFuncs
// func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		if err := f(w, r); err != nil {
// 			//handle the error
// 			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
// 		}
// 	}
// }