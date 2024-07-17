package domain

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB() (*sql.DB, *SmokeLogger, error) {
	// Создаем in-memory базу данных SQLite
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, nil, err
	}

	// Инициализация базы данных
	smokeLogger := NewSmokeLogger(db)
	// smokeLogger.InitDB()

	return db, smokeLogger, nil
}

func TestAddEntry(t *testing.T) {
	db, smokeLogger, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer db.Close()

	smokeLogger.AddEntry()

	if len(smokeLogger.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(smokeLogger.Entries))
	}

	entry := smokeLogger.Entries[1]
	if entry.Value != 1 {
		t.Fatalf("Expected entry value to be 1, got %d", entry.Value)
	}
}

func TestNewDay(t *testing.T) {
	db, smokeLogger, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer db.Close()

	smokeLogger.NewDay()
	t.Log(smokeLogger.Days)
	if len(smokeLogger.Days) != 2 {
		t.Fatalf("Expected 1 day, got %d", len(smokeLogger.Days))
	}

	day := smokeLogger.Days[1]
	if day.EndTS != nil {
		t.Fatalf("Expected end timestamp to be nil, got %v", *day.EndTS)
	}
}

func TestLoadEntriesByDay(t *testing.T) {
	db, smokeLogger, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test DB: %v", err)
	}
	defer db.Close()

	smokeLogger.NewDay()
	time.Sleep(2 * time.Second)
	smokeLogger.AddEntry()

	trueEntries := smokeLogger.LoadEntries() // Загружаем все записи

	t.Logf("True Entries: %+v", trueEntries)

	t.Logf("Days: %+v, CurrentDay: %+v", smokeLogger.Days, smokeLogger.CurrentDay)
	t.Logf("Current Day: %+v", smokeLogger.Days[smokeLogger.CurrentDay])
	day := smokeLogger.Days[smokeLogger.CurrentDay]
	entries, err := smokeLogger.LoadEntriesByDay(day)
	if err != nil {
		t.Fatalf("Failed to load entries by day: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Value != 1 {
		t.Fatalf("Expected entry value to be 1, got %d", entry.Value)
	}
}
