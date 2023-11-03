package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)


type SqliteStore struct {
	db *sql.DB
}

func NewSqliteStore() (*SqliteStore, error) {
	db, err := sql.Open("sqlite3", "laser.db?cache=shared&mode=rwc&_journal=WAL&_timeout=2000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SqliteStore{
		db: db,
	}, nil
}

func (s *SqliteStore) Init() error {
	err := s.CreateLaserTable()
	if err != nil {
		return err
	}
	return s.CreateHistoryTable()
}

func (s *SqliteStore) CreateLaserTable() error {
	query := `CREATE TABLE IF NOT EXISTS laser (
		id INTEGER PRIMARY KEY,
		results_bib TEXT NOT NULL UNIQUE,
		results_first_name TEXT NOT NULL,
		results_last_name TEXT NOT NULL,
		results_time TEXT,
		results_gun_time TEXT
	);`

	queryClear := `DELETE FROM laser`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(queryClear)
	return err
}

func (s *SqliteStore) CreateHistoryTable() error {
	query := `CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bib TEXT NOT NULL UNIQUE REFERENCES laser(results_bib),
		created_at TIMESTAMP NOT NULL
	);`
	queryClear := `DELETE FROM history`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(queryClear)
	return err
}

func (s *SqliteStore) CreateRecord(a *Athlete) error {

	randID := rand.Intn(math.MaxInt64)
	query := `
		INSERT INTO laser (id, results_bib, results_first_name, results_last_name, results_time, results_gun_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		;`
	_, err := s.db.Exec(query, randID, a.ResultsBib, a.ResultsFirstName, a.ResultsLastName, a.ResultsTime, a.ResultsGunTime)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) CreateBulkRecords(a *[]Athlete) error {
	// valueStrings := make([]string, 0, len(*a))
	var valueStrings string
	valueArgs := make([]interface{}, 0, len(*a) * 6)

	valueStrings = strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?),", len(*a)), ",")
	for _, athlete := range *a {
		// valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, rand.Intn(math.MaxInt64))
		valueArgs = append(valueArgs, athlete.ResultsBib)
		valueArgs = append(valueArgs, athlete.ResultsFirstName)
		valueArgs = append(valueArgs, athlete.ResultsLastName)
		valueArgs = append(valueArgs, athlete.ResultsTime)
		valueArgs = append(valueArgs, athlete.ResultsGunTime)
	}
	// this query for append option to create valueStrings
	// query := fmt.Sprintf("INSERT INTO laser (id, results_bib, results_first_name, results_last_name, results_time, results_gun_time) VALUES %s;", strings.Join(valueStrings, ","))

	query := fmt.Sprintf("INSERT INTO laser (id, results_bib, results_first_name, results_last_name, results_time, results_gun_time) VALUES %s;", valueStrings)

	_, err := s.db.Exec(query, valueArgs...)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqliteStore) Checkpoint() {
	_, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
if err != nil {
 panic(err)
}
}

func (s *SqliteStore) CreateHistoryRecord(a *Athlete) error {
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

func (s *SqliteStore) GetHistoryRecords() ([]*Athlete, error) {
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

func (s *SqliteStore) GetLatestHistoryRecord() (*Athlete, error) {
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

func (s *SqliteStore) GetRecordByBib(bib string) (*Athlete, error) {
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

func (s *SqliteStore) GetRecords() ([]*Athlete, error) {
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

func (s *SqliteStore) GetRecordsCount() int {
	var count int
	query := `SELECT COUNT(*) FROM laser`
	resp := s.db.QueryRow(query)
	err := resp.Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count
}

func (s *SqliteStore) ClearHistory() {
	query := `DELETE FROM history;`
	_, err := s.db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

}
