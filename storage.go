package main

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateRecord(*Athlete) error
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
	return s.CreateLaserTable()
}

func (s *PostgresStore) CreateLaserTable() error {
	query := `CREATE TABLE IF NOT EXISTS laser (
		id SERIAL PRIMARY KEY,
		results_bib TEXT NOT NULL UNIQUE,
		results_first_name TEXT NOT NULL,
		results_last_name TEXT NOT NULL,
		results_time TEXT,
		results_gun_time TEXT
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateRecord(a *Athlete) error {
	query := `
		INSERT INTO laser (results_bib, results_first_name, results_last_name, results_time, results_gun_time)
		VALUES ($1, $2, $3, $4, $5)
		;`
	_, err := s.db.Exec(query, a.ResultsBib, a.ResultsFirstName, a.ResultsLastsName, a.ResultsTime, a.ResultsGunTime)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) GetRecords() ([]*Athlete, error) {
	query := `
		SELECT * FROM laser;
	`
	resp, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	athletes := []*Athlete{}
	for resp.Next() {
		a := new(Athlete)
		if err := resp.Scan(
			&a.ResultsBib,
			&a.ResultsFirstName,
			&a.ResultsLastsName,
			&a.ResultsTime,
			&a.ResultsGunTime,
		); err != nil {
			return nil, err
		}
		athletes = append(athletes, a)
	}

	return athletes, nil
}

func (s *PostgresStore) GetRecordByBib(bib string) (*Athlete, error) {
	query := `
		SELECT results_bib, results_first_name, results_last_name, results_time, results_gun_time
		FROM laser WHERE results_bib = $1;
	`
	res := s.db.QueryRow(query, bib)
	a := new(Athlete)
	if err := res.Scan(
			&a.ResultsBib,
			&a.ResultsFirstName,
			&a.ResultsLastsName,
			&a.ResultsTime,
			&a.ResultsGunTime,
		); err != nil {
		return nil, err
	}
	gunTime, err := processTimeStr(a.ResultsGunTime)
	if err != nil {
		return nil, err
	}
	netTime, err := processTimeStr(a.ResultsTime)
	if err != nil {
		return nil, err
	}
	a.ResultsGunTime = gunTime
	a.ResultsTime = netTime
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