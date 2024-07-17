package main

import (
	"database/sql"
	"go-backend/domain"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Can not open database: %v", err)
	}
	return db
}
func main() {
	db := InitDB("./smokelogger.db")
	defer db.Close()

	smokeLogger := domain.NewSmokeLogger(db)

	router := gin.Default()
	// Allow all origins
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.GET("/api/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello World!",
		})
	})
	router.DELETE("/api/clear_entries", func(c *gin.Context) {
		smokeLogger.ClearDB()
		c.JSON(200, gin.H{"message": "Entries cleared!"})
	})
	router.GET("/api/get_entries", func(c *gin.Context) {
		entries, err := smokeLogger.LoadEntriesByDay(smokeLogger.Days[smokeLogger.CurrentDay])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, entries)
	})

	router.GET("/api/get_entries_by_day/:dayID", func(c *gin.Context) {
		dayID := c.Param("dayID")
		id, err := strconv.Atoi(dayID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day ID"})
			return
		}
		day, exists := smokeLogger.Days[id]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Day not found"})
			return
		}

		entries, err := smokeLogger.LoadEntriesByDay(day)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, entries)
	})

	router.GET("/api/get_days", func(c *gin.Context) {
		days := make([]domain.DayEntry, 0, len(smokeLogger.Days))
		for _, day := range smokeLogger.Days {
			days = append(days, day)
		}
		c.JSON(200, days)
	})

	router.POST("/api/add_entry", func(c *gin.Context) {
		entry := smokeLogger.AddEntry()
		c.JSON(200, entry)
	})

	router.POST("/api/new_day", func(c *gin.Context) {
		smokeLogger.NewDay()
		c.JSON(200, smokeLogger.Days[smokeLogger.CurrentDay])
	})

	router.POST("/api/prev_day", func(c *gin.Context) {
		smokeLogger.PrevDay()
		c.JSON(200, smokeLogger.Days[smokeLogger.CurrentDay])
	})

	router.POST("/api/next_day", func(c *gin.Context) {
		smokeLogger.NextDay()
		c.JSON(200, smokeLogger.Days[smokeLogger.CurrentDay])
	})

	router.DELETE("/api/delete_entry/:id", func(c *gin.Context) {
		id := c.Param("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
		}
		smokeLogger.DeleteEntry(idInt)
		c.JSON(200, gin.H{"message": "Entry deleted"})
	})

	router.GET("/api/get_counter", func(c *gin.Context) {
		c.JSON(200, gin.H{"counter": smokeLogger.Counter})
	})

	router.GET("/api/get_current_day", func(c *gin.Context) {
		c.JSON(200, smokeLogger.Days[smokeLogger.CurrentDay])
	})

	router.Run(":8080")
}
