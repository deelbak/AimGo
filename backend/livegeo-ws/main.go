package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type GeoUpdate struct {
	DriverID string  `json:"driver_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type Hub struct {
	mu      sync.RWMutex
	drivers map[string]*websocket.Conn
	riders  map[string]*websocket.Conn
}

var hub = &Hub{
	drivers: make(map[string]*websocket.Conn),
	riders:  make(map[string]*websocket.Conn),
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	r := gin.Default()

	r.GET("/ws/driver/:driver_id", driverWS)
	r.GET("/ws/rider/:passenger_id", riderWS)
	r.GET("/location/:driver_id", getDriverLocation)

	log.Println("LiveGo WebSocket service is running on port 8004...")
	r.Run(":8004")
}

var (
	locationsMu sync.RWMutex
	locations   = make(map[string]GeoUpdate)
)

func driverWS(c *gin.Context) {
	driverID := c.Param("driver_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Регистрируем водителя
	hub.mu.Lock()
	hub.drivers[driverID] = conn
	hub.mu.Unlock()

	log.Printf("Driver %s connected", driverID)

	defer func() {
		hub.mu.Lock()
		delete(hub.drivers, driverID)
		hub.mu.Unlock()
		log.Printf("Driver %s disconnected", driverID)
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var update GeoUpdate
		if err := json.Unmarshal(msg, &update); err != nil {
			continue
		}
		update.DriverID = driverID

		// Сохраняем последнюю позицию
		locationsMu.Lock()
		locations[driverID] = update
		locationsMu.Unlock()

		// Рассылаем всем пассажирам
		hub.mu.RLock()
		for _, riderConn := range hub.riders {
			riderConn.WriteMessage(websocket.TextMessage, msg)
		}
		hub.mu.RUnlock()
	}
}

func riderWS(c *gin.Context) {
	passengerID := c.Param("passenger_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error for rider %s: %v\n", passengerID, err)
		return
	}

	defer conn.Close()

	hub.mu.Lock()
	hub.riders[passengerID] = conn
	hub.mu.Unlock()

	log.Printf("Rider %s connected", passengerID)

	defer func() {
		hub.mu.Lock()
		delete(hub.riders, passengerID)
		hub.mu.Unlock()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error for rider %s: %v\n", passengerID, err)
			break
		}
	}
}

func getDriverLocation(c *gin.Context) {
	driverID := c.Param("driver_id")

	locationsMu.RLock()
	loc, ok := locations[driverID]
	locationsMu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Driver location not found"})
		return
	}

	c.JSON(http.StatusOK, loc)
}
