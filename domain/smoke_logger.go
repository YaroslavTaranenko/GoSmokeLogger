package domain

import (
	"database/sql"
	"log"
	"time"
)

type SmokeEntry struct {
	ID    int
	TS    time.Time
	Value int
}

type DayEntry struct {
	ID      int
	StartTS time.Time
	EndTS   *time.Time
}

type SmokeLogger struct {
	db         *sql.DB
	Entries    map[int]SmokeEntry
	Days       map[int]DayEntry
	Counter    int
	CurrentDay int
}

func NewSmokeLogger(db *sql.DB) *SmokeLogger {
	sl := &SmokeLogger{
		db:         db,
		Entries:    make(map[int]SmokeEntry),
		Days:       make(map[int]DayEntry),
		Counter:    0,
		CurrentDay: 0,
	}

	sl.InitDB()
	// sl.loadData()

	return sl
}

func (sl *SmokeLogger) InitDB() {
	createSmokeEntriesTableSQL := `CREATE TABLE IF NOT EXISTS smoke_entries (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"ts" DATETIME DEFAULT CURRENT_TIMESTAMP,
		"value" INTEGER
	);`

	createDayEntriesTableSQL := `CREATE TABLE IF NOT EXISTS days (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"start_ts" DATETIME DEFAULT CURRENT_TIMESTAMP,
		"end_ts" DATETIME
	);`

	countDaysSql := `SELECT COUNT(id) AS count FROM days;`
	inserDefaultDayEntrySQL := `INSERT INTO days (start_ts, end_ts) VALUES (?, ?);`

	_, err := sl.db.Exec(createSmokeEntriesTableSQL)
	if err != nil {
		log.Fatalf("Cannot create smoke_entries table: %v", err)
	}
	_, err = sl.db.Exec(createDayEntriesTableSQL)
	if err != nil {
		log.Fatalf("Cannot create days table: %v", err)
	}

	rows, err := sl.db.Query(countDaysSql)
	if err != nil {
		log.Fatalf("Cannot query days table: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err != nil {
			log.Fatalf("Cannot scan day entry: %v", err)
		}

		log.Printf("Found %d days", count)
		if count == 0 {
			rows.Close()
			_, err = sl.db.Exec(inserDefaultDayEntrySQL, time.Now(), nil)
			if err != nil {
				log.Printf("Cannot insert default day entry: %v", err)
			}
		}
	}

	sl.loadDays()
	sl.LoadEntriesByDay(sl.Days[sl.CurrentDay])
}

func (sl *SmokeLogger) ClearDB() {
	truncEntries := "DELETE FROM smoke_entries;"
	truncDays := "DELETE FROM days;"

	_, err := sl.db.Exec(truncEntries)
	if err != nil {
		log.Fatalf("Cannot truncate smoke_entries table: %v", err)
	}
	_, err = sl.db.Exec(truncDays)
	if err != nil {
		log.Fatalf("Cannot truncate days table: %v", err)
	}
}

// func (sl *SmokeLogger) loadData() {
// 	sl.loadDays()
// 	sl.LoadEntriesByDay(sl.Days[sl.CurrentDay])
// }

func (sl *SmokeLogger) LoadEntries() map[int]SmokeEntry {
	rows, err := sl.db.Query("SELECT id, ts, value FROM smoke_entries")
	if err != nil {
		log.Fatalf("Cannot query smoke_entries table: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry SmokeEntry
		if err := rows.Scan(&entry.ID, &entry.TS, &entry.Value); err != nil {
			log.Fatalf("Cannot scan smoke entry: %v", err)
		}
		sl.Entries[entry.ID] = entry

		sl.Counter = entry.Value
	}
	return sl.Entries
}

func (sl *SmokeLogger) loadDays() {
	rows, err := sl.db.Query("SELECT id, start_ts, end_ts FROM days ORDER BY start_ts")
	if err != nil {
		log.Fatalf("Cannot query days table: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var day DayEntry
		var endTS sql.NullTime
		if err := rows.Scan(&day.ID, &day.StartTS, &endTS); err != nil {
			log.Fatalf("Cannot scan day entry: %v", err)
		}
		if endTS.Valid {
			day.EndTS = &endTS.Time
		} else {
			day.EndTS = nil
		}
		sl.Days[day.ID] = day
		sl.CurrentDay = day.ID
	}
}

func (sl *SmokeLogger) getEntryById(id int) (*SmokeEntry, error) {
	query := "SELECT id, ts, value FROM smoke_entries WHERE id = ?"
	row := sl.db.QueryRow(query, id)
	var entry SmokeEntry
	if err := row.Scan(&entry.ID, &entry.TS, &entry.Value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (sl *SmokeLogger) AddEntry() SmokeEntry {
	currentTime := time.Now()
	sl.Counter++
	result, err := sl.db.Exec("INSERT INTO smoke_entries (ts,value) VALUES (?, ?)", currentTime, sl.Counter)
	if err != nil {
		log.Fatalf("Cannot insert smoke entry: %v", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		log.Fatalf("Cannot get last insert ID: %v", err)
	}

	entry, err := sl.getEntryById(int(lastInsertID))
	if err != nil {
		log.Fatalf("Cannot get smoke entry by ID: %v", err)
	}
	if entry != nil {
		sl.Entries[entry.ID] = *entry
	}

	return *entry
}

func (sl *SmokeLogger) DeleteEntry(id int) {
	_, err := sl.db.Exec("DELETE FROM smoke_entries WHERE id = ?", id)
	if err != nil {
		log.Fatalf("Cannot delete smoke entry: %v", err)
	}
	delete(sl.Entries, id)

	// Обновление sl.Counter до последнего значения
	sl.Counter = 0
	for _, entry := range sl.Entries {
		if entry.Value > sl.Counter {
			sl.Counter = entry.Value
		}
	}
}

func (sl *SmokeLogger) NewDay() {
	currentTime := time.Now()
	_, err := sl.db.Exec("UPDATE days SET end_ts = ? WHERE end_ts IS NULL", currentTime)
	if err != nil {
		log.Fatalf("Cannot update day entry: %v", err)
	}

	result, err := sl.db.Exec("INSERT INTO days (start_ts, end_ts) VALUES (?, NULL)", currentTime)
	if err != nil {
		log.Fatalf("Cannot insert day entry: %v", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		log.Fatalf("Cannot get last insert ID: %v", err)
	}

	sl.CurrentDay = int(lastInsertID)
	sl.Days[sl.CurrentDay] = DayEntry{
		ID:      sl.CurrentDay,
		StartTS: time.Now(),
		EndTS:   nil,
	}
}

func (sl *SmokeLogger) NextDay() {
	sl.CurrentDay++
	day, exists := sl.Days[sl.CurrentDay]
	if !exists {
		log.Fatalf("Cannot get current day: day %d does not exist", sl.CurrentDay)
	}
	entries, err := sl.LoadEntriesByDay(day)
	if err != nil {
		log.Fatalf("Cannot load entries for the day: %v", err)
	}
	sl.Entries = make(map[int]SmokeEntry) // Очистите текущие записи перед загрузкой новых
	for _, entry := range entries {
		sl.Entries[entry.ID] = entry
	}
}

func (sl *SmokeLogger) PrevDay() {
	sl.CurrentDay--
	day, exists := sl.Days[sl.CurrentDay]
	if !exists {
		log.Fatalf("Cannot get current day: day %d does not exist", sl.CurrentDay)
	}
	entries, err := sl.LoadEntriesByDay(day)
	if err != nil {
		log.Fatalf("Cannot load entries for the day: %v", err)
	}
	sl.Entries = make(map[int]SmokeEntry) // Очистите текущие записи перед загрузкой новых
	for _, entry := range entries {
		sl.Entries[entry.ID] = entry
	}
}

func (sl *SmokeLogger) LoadEntriesByDay(day DayEntry) ([]SmokeEntry, error) {
	var endTS time.Time
	if day.EndTS != nil {
		endTS = *day.EndTS
	} else {
		endTS = time.Now()
	}

	query := "SELECT id, ts, value FROM smoke_entries WHERE ts BETWEEN ? AND ? ORDER BY ts"
	rows, err := sl.db.Query(query, day.StartTS, endTS)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SmokeEntry
	for rows.Next() {
		var entry SmokeEntry
		if err := rows.Scan(&entry.ID, &entry.TS, &entry.Value); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
