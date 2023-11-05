package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type Storage interface {
	// CreateLaserTable() error
	// CreateRecord(*Athlete) error
	// GetRecords() ([]*Athlete, error)
	GetHistoryRecords() ([]*Athlete, error)
	CreateBulkRecords(a *[]Athlete) error
	GetRecordByBib(bib string) (*Athlete, error)
	GetLatestHistoryRecord() (*Athlete, error)
	GetRecordsCount() int
	ClearHistory()
	Checkpoint()
}
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=postgres password=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	err := s.CreateLaserTable()
	if err != nil {
		return err
	}
	return s.CreateHistoryTable()
}

func (s *PostgresStore) CreateLaserTable() error {
	query := `CREATE TABLE IF NOT EXISTS laser (
		id SERIAL PRIMARY KEY,
		results_bib TEXT NOT NULL UNIQUE,
		results_first_name TEXT,
		results_last_name TEXT,
		results_time TEXT,
		results_gun_time TEXT
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateHistoryTable() error {
	query := `CREATE TABLE IF NOT EXISTS history (
		id SERIAL PRIMARY KEY,
		bib TEXT NOT NULL UNIQUE REFERENCES laser(results_bib),
		created_at TIMESTAMP NOT NULL
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateRecord(a *Athlete) error {
	query := `
		INSERT INTO laser (results_bib, results_first_name, results_last_name, results_time, results_gun_time)
		VALUES ($1, $2, $3, $4, $5)
		;`
	_, err := s.db.Exec(query, a.ResultsBib, a.ResultsFirstName, a.ResultsLastName, a.ResultsTime, a.ResultsGunTime)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) CreateHistoryRecord(a *Athlete) error {
	query := `
		INSERT INTO history (bib, created_at)
		VALUES ($1, $2)
		;`
	_, err := s.db.Exec(query, a.ResultsBib, time.Now().UTC())
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) GetHistoryRecords() ([]*Athlete, error) {
	query := `
		SELECT history.bib,
		laser.results_first_name, 
		laser.results_last_name,
		laser.results_time,
		laser.results_gun_time
		FROM history JOIN laser ON history.bib = laser.results_bib ORDER BY history.created_at DESC;
	`
	resp, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	athletes := []*Athlete{}
	for resp.Next() {
		a := new(Athlete)
		if err := resp.Scan(
			&a.ResultsBib,
			&a.ResultsFirstName,
			&a.ResultsLastName,
			&a.ResultsTime,
			&a.ResultsGunTime,
		); err != nil {
			return nil, err
		}
		athletes = append(athletes, a)
	}

	for _, a := range athletes {
		err := processTimeForRecord(a)
		if err != nil {
			return nil, err
		}
	}
	return athletes, nil
}

func (s *PostgresStore) GetLatestHistoryRecord() (*Athlete, error) {
	query := `
		SELECT history.bib,
		laser.results_first_name, 
		laser.results_last_name,
		laser.results_time,
		laser.results_gun_time
		FROM history JOIN laser ON history.bib = laser.results_bib ORDER BY history.created_at DESC
		LIMIT 1
	`
	row := s.db.QueryRow(query)
	a := new(Athlete)
	err := row.Scan(
		&a.ResultsBib,
		&a.ResultsFirstName,
		&a.ResultsLastName,
		&a.ResultsTime,
		&a.ResultsGunTime,
	)
	if err != nil {
		return nil, err
	}
	err = processTimeForRecord(a)
	if err != nil {
		return nil, err
	}
	return a, nil

}

func processTimeForRecord(a *Athlete) error {
	gunTime, err := processTimeStr(a.ResultsGunTime)
	if err != nil {
		return err
	}
	netTime, err := processTimeStr(a.ResultsTime)
	if err != nil {
		return err
	}
	a.ResultsGunTime = gunTime
	a.ResultsTime = netTime
	return nil
}

func (s *PostgresStore) GetRecordByBib(bib string) (*Athlete, error) {
	query := `
		SELECT results_bib, results_first_name, results_last_name, results_time, results_gun_time
		FROM laser WHERE results_bib = $1;
	`
	res := s.db.QueryRow(query, bib)
	a := new(Athlete)
	err := res.Scan(
		&a.ResultsBib,
		&a.ResultsFirstName,
		&a.ResultsLastName,
		&a.ResultsTime,
		&a.ResultsGunTime,
	)
	if err != nil {
		return nil, err
	}
	err = processTimeForRecord(a)
	if err != nil {
		return nil, err
	}
	s.CreateHistoryRecord(a)
	return a, nil
}

func processTimeStr(timeToParse string) (string, error) {
	t, err := time.Parse(time.TimeOnly, timeToParse)
	if err != nil {
		return "", err
	}

	ms := t.Nanosecond()
	truncTime := t.Truncate(time.Second)
	if ms > 0 {
		truncTime = truncTime.Add(time.Second)
	}
	return truncTime.Format(time.TimeOnly), nil

}

func (s *PostgresStore) GetRecords() ([]*Athlete, error) {
	query := `
		SELECT * FROM laser;
	`
	resp, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	athletes := []*Athlete{}
	for resp.Next() {
		a := new(Athlete)
		if err := resp.Scan(
			&a.ResultsBib,
			&a.ResultsFirstName,
			&a.ResultsLastName,
			&a.ResultsTime,
			&a.ResultsGunTime,
		); err != nil {
			return nil, err
		}
		athletes = append(athletes, a)
	}

	return athletes, nil
}

func (s *PostgresStore) GetRecordsCount() int {
	var count int
	query := `SELECT COUNT(*) FROM laser`
	resp := s.db.QueryRow(query)
	err := resp.Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count
}

func (s *PostgresStore) ClearHistory() {
	query := `TRUNCATE TABLE history;`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

}
