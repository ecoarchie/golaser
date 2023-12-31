package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type APIServer struct {
	listenAddr string
	store      Storage
	scraper    Scraper
}

func NewAPIServer(listenAddr string, store Storage, scraper Scraper) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
		scraper:    scraper,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/", s.handleIndexPage)
	router.HandleFunc("/search", s.handleSearchBib)
	router.HandleFunc("/archive", s.HandleArchiveRecord)
	router.HandleFunc("/pupdate", s.HandlePartialDBUpdate)
	router.HandleFunc("/auto-update-start", s.HandleStartAutoDBUpdate)
	router.HandleFunc("/auto-update-stop", s.HandleStopAutoDBUpdate)
	router.HandleFunc("/history", s.HandleDeleteHistory)
	router.HandleFunc("/config", s.HandleCreateConfig)

	log.Println("JSON API server running on port: ", s.listenAddr)
	log.Printf("http://localhost%s\n", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

var (
	//go:embed static
	res embed.FS

	pages = map[string]string{
		"/": "static/index.html",
	}
)

func (s *APIServer) handleIndexPage(w http.ResponseWriter, r *http.Request) {
	page, ok := pages[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	records, err := s.store.GetHistoryRecords()

	if err != nil {
		fmt.Println("error", err)
	}
	templ := template.Must(template.ParseFS(res, page))

	data := map[string][]*Athlete{
		"Records": {},
	}
	data["Records"] = records
	templ.Execute(w, data)
}

func (s *APIServer) handleSearchBib(w http.ResponseWriter, r *http.Request) {
	bib := r.PostFormValue("bib")
	a, err := s.store.GetRecordByBib(bib)
	if err != nil {
		fmt.Println("error", err)
	}

	var htmlStr string
	if a == nil {
		htmlStr = fmt.Sprintf(`
			<button type='button' class='list-group-item list-group-item-action list-group-item-danger' id='copy-data'>Участник %s не найден</button>
			`, bib)
	} else {
		htmlStr = fmt.Sprintf(`
			<button type='button' class='list-group-item list-group-item-action list-group-item-success' id='copy-data' onclick='copyToClipboard()'>%s %s %s</button>
			`,
			a.ResultsFirstName, a.ResultsLastName, a.ResultsTime)

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

func alertDangerResponse(w http.ResponseWriter, header string, errText string) {
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-danger" role="alert">
		<h4 class="alert-heading">%s</h4>
		<p>%s</p>
		</div>
	`, header, errText)
	templ, _ := template.New("alert").Parse(htmlStr)
	templ.Execute(w, nil)

}

func (s *APIServer) HandlePartialDBUpdate(w http.ResponseWriter, r *http.Request) {
	if s.scraper.config.clientID == "" || s.scraper.config.eventID == "" || s.scraper.config.source == "" {
		alertDangerResponse(w, "Соревнование не настроено", "Заполните поля в разделе 'Настройка соревнования'")
		return
	}
	recordsAdded := s.scraper.StartPartialScraping()
	updTime := time.Now().Format(time.TimeOnly)
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-info" role="alert">
		<h4 class="alert-heading">База данных обновлена!</h4>
		<p>Добавлено %d записей в %s.</p>
		</div>
	`, recordsAdded, updTime)
	templ, _ := template.New("count").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandleStartAutoDBUpdate(w http.ResponseWriter, r *http.Request) {
	if s.scraper.config.clientID == "" || s.scraper.config.eventID == "" || s.scraper.config.source == "" {
		alertDangerResponse(w, "Соревнование не настроено", "Заполните поля в разделе 'Настройка соревнования'")
		return
	}
	// recordsAdded := s.scraper.StartPartialScraping()
	nextUpdTime := time.Now().Add(time.Minute * 5).Format(time.TimeOnly)
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-info" role="alert">
		<h4 class="alert-heading">Автообновление запущено</h4>
		<p>База данных будет обновляться каждые 5 минут</p>
		<p>Следующее обновление в <strong>%s</strong></p>
		</div>

		<button type="button"
		hx-post="/pupdate" 
		hx-target="#notification" 
		hx-swap="innerHTML"
		hx-swap-oob="true"
		hx-indicator="#spinner"
		hx-trigger="every 5m, click"
		id="btn-manual-update"
		class="btn btn-primary">
			Обновить базу
			<div class="htmx-indicator spinner-border spinner-border-sm" role="status" id="spinner"></div>
		</button>

		<button type="button" 
		hx-post="/auto-update-stop" 
		hx-target="#notification" 
		hx-swap="innerHTML" 
		hx-swap-oob="true"
		hx-indicator="#spinner-auto" 
		id="btn-auto-update" 
		class="btn btn-warning">
			Остановить
		</button>
	`, nextUpdTime)
	templ, _ := template.New("count").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandleStopAutoDBUpdate(w http.ResponseWriter, r *http.Request) {
	if s.scraper.config.clientID == "" || s.scraper.config.eventID == "" || s.scraper.config.source == "" {
		alertDangerResponse(w, "Соревнование не настроено", "Заполните поля в разделе 'Настройка соревнования'")
		return
	}
	// recordsAdded := s.scraper.StartPartialScraping()
	updTime := time.Now().Format(time.TimeOnly)
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-info" role="alert">
		<h4 class="alert-heading">Автообновление остановлено</h4>
		<p>%s</p>
		</div>

		<button type="button"
		hx-post="/pupdate" 
		hx-target="#notification" 
		hx-swap="innerHTML"
		hx-swap-oob="true"
		hx-indicator="#spinner"
		hx-trigger="click"
		id="btn-manual-update"
		class="btn btn-primary">
			Обновить базу
			<div class="htmx-indicator spinner-border spinner-border-sm" role="status" id="spinner"></div>
		</button>

		<button type="button" 
		hx-post="/auto-update-start" 
		hx-target="#notification" 
		hx-swap="innerHTML" 
		hx-swap-oob="true"
		hx-indicator="#spinner-auto" 
		id="btn-auto-update" 
		class="btn btn-secondary">
			Автообновление
		</button>
	`, updTime)
	templ, _ := template.New("count").Parse(htmlStr)
	templ.Execute(w, nil)
}

func (s *APIServer) HandleDeleteHistory(w http.ResponseWriter, r *http.Request) {
	s.store.ClearHistory()
	// http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func (s *APIServer) HandleCreateConfig(w http.ResponseWriter, r *http.Request) {
	login := r.PostFormValue("login")
	password := r.PostFormValue("password")
	clienID := r.PostFormValue("clientID")
	eventID := r.PostFormValue("eventID")

	s.scraper.config = *s.scraper.config.Default(login, password, clienID, eventID)

	event, err := s.scraper.CheckEventURL()
	if err != nil {
		alertDangerResponse(w, "Конфигурация Chronotrack API НЕ НАСТРОЕНА!", fmt.Sprintf("Ошибка %s", err))
		return
	}
	timeParsed, err := strconv.Atoi(event.StartTime)
	if err != nil {
		log.Fatal(err)
	}
	startTime := time.Unix(int64(timeParsed), 0).Format(time.TimeOnly)
	htmlStr := fmt.Sprintf(`
		<div class="alert alert-info" role="alert">
		<h4 class="alert-heading">Конфигурация Chronotrack API настроена!</h4>
		<p>Соревнование <strong>%s</strong></p>
		<p>Начало в %s</p>
		</div>
	`, event.EventName, startTime)
	templ, _ := template.New("config").Parse(htmlStr)
	templ.Execute(w, nil)
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
