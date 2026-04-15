package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Driver struct {
	ID         string    `json:"id"`
	IsOnline   bool      `json:"is_online"`
	CurrentLat float64   `json:"current_lat,omitempty"`
	CurrentLng float64   `json:"current_lng,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`

	Name  string `json:"name"`
	Phone string `json:"phone"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=passw0rd dbname=aimgo sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Database not reachable: %v\n", err)
	}

	log.Println("Connected to database successfully")

	r := gin.Default()

	r.PUT("/driver/:id/online", setOnline)
	r.PUT("/driver/:id/offline", setOffline)
	r.GET("/drivers/online", getOnlineDrivers)
	r.GET("/driver/:id", getDriver)

	log.Println("driver-service running on :8003...")
	r.Run(":8003")
}

func setOnline(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		INSERT INTO drivers (id, is_online, current_lat, current_lng, updated_at)
		VALUES ($1::uuid, true, $2, $3, NOW())
		ON CONFLICT (id) DO UPDATE
		SET is_online = true, current_lat = $2, current_lng = $3, updated_at = NOW()
	`, id, body.Lat, body.Lng)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "online"})
}

func setOffline(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec(`UPDATE drivers SET is_online = false, updated_at = NOW() WHERE id = $1::uuid
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "offline"})
}

func getOnlineDrivers(c *gin.Context) {
	rows, err := db.Query(`
		SELECT d.id, d.is_online, d.current_lat, d.current_lng, d.updated_at, u.name, u.phone FROM drivers d JOIN users u ON d.id = u.id WHERE d.is_online = true
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer rows.Close()

	var drivers []Driver
	for rows.Next() {
		var d Driver
		err := rows.Scan(
			&d.ID, &d.IsOnline, &d.CurrentLat, &d.CurrentLng, &d.UpdatedAt, &d.Name, &d.Phone,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		drivers = append(drivers, d)
	}
	c.JSON(http.StatusOK, drivers)
}

func getDriver(c *gin.Context) {
	id := c.Param("id")

	var d Driver

	err := db.QueryRow(`
		SELECT d.id, d.is_online, d.current_lat, d.current_lng, d.updated_at, u.name, u.phone FROM drivers d JOIN users u ON d.id = u.id WHERE d.id = $1::uuid
	`, id).Scan(
		&d.ID, &d.IsOnline, &d.CurrentLat, &d.CurrentLng, &d.UpdatedAt, &d.Name, &d.Phone,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}
