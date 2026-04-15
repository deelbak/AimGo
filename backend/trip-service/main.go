package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Trip struct {
	ID          string    `json:"id"`
	PassengerID string    `json:"passenger_id"`
	DriverID    *string   `json:"driver_id"`
	FromLat     float64   `json:"from_lat"`
	FromLng     float64   `json:"from_lng"`
	ToLat       float64   `json:"to_lat"`
	ToLng       float64   `json:"to_lng"`
	Status      string    `json:"status"` // pending, accepted, ongoing, done
	Price       int       `json:"price"`  // всегда 500
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=passw0rd dbname=aimgo sslmode=disable")

	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}

	defer db.Close()

	r := gin.Default()

	r.POST("/trips", createTrip)
	r.GET("/trips", getAllPendingTrips)
	r.GET("/trips/:id", getTrip)
	r.PUT("/trips/:id/accept", acceptTrip)
	r.PUT("/trips/:id/start", startTrip)
	r.PUT("/trips/:id/complete", doneTrip)

	log.Println("Trip service is running on port 8005...")

	r.Run(":8005")
}

func createTrip(c *gin.Context) {
	var trip Trip

	if err := c.ShouldBindJSON(&trip); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trip.Price = 500
	trip.Status = "pending"
	trip.CreatedAt = time.Now()

	err := db.QueryRow(
		`INSERT INTO trips (passenger_id, from_lat, from_lng, to_lat, to_lng, status, price) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		trip.PassengerID, trip.FromLat, trip.FromLng, trip.ToLat, trip.ToLng,
		trip.Status, trip.Price).Scan(&trip.ID, &trip.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, trip)
}

func getTrip(c *gin.Context) {
	id := c.Param("id")
	var trip Trip

	err := db.QueryRow(`
		SELECT id, passenger_id, driver_id, from_lat, from_lng, to_lat, to_lng, status, price, created_at 
		FROM trips WHERE id = $1`, id).Scan(
		&trip.ID, &trip.PassengerID, &trip.DriverID, &trip.FromLat, &trip.FromLng,
		&trip.ToLat, &trip.ToLng, &trip.Status, &trip.Price, &trip.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, trip)
}

func getAllPendingTrips(c *gin.Context) {
	var trips []Trip
	rows, err := db.Query(`
		SELECT id, passenger_id, driver_id, from_lat, from_lng, to_lat, to_lng 
		FROM trips WHERE status = 'pending'
	`)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "No pending trips found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	defer rows.Close()

	for rows.Next() {
		var trip Trip
		if err := rows.Scan(&trip.ID, &trip.PassengerID, &trip.DriverID, &trip.FromLat, &trip.FromLng, &trip.ToLat, &trip.ToLng); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		trips = append(trips, trip)
	}

	c.JSON(http.StatusOK, trips)

}

func acceptTrip(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		DriverID string `json:"driver_id"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`UPDATE trips SET driver_id = $1, status = 'accepted' 
		WHERE id = $2 AND status = 'pending'`, body.DriverID, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Trip accepted"})
}

func startTrip(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec(`
		UPDATE trips SET status = 'done' WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "done"})
}

func doneTrip(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec(`
		UPDATE trips SET status = 'done' WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "done"})
}
