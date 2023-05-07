package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Spot struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Rating    float64 `json:"rating"`
}

func main() {
	// Connect to the database
	db, err := sql.Open("postgres", "postgres://user:password@localhost/dbname?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a new router
	r := mux.NewRouter()

	// Define the endpoint
	r.HandleFunc("/spots", func(w http.ResponseWriter, r *http.Request) {
		// Parse the request parameters
		latStr := r.URL.Query().Get("latitude")
		lonStr := r.URL.Query().Get("longitude")
		radStr := r.URL.Query().Get("radius")
		typeStr := r.URL.Query().Get("type")

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			http.Error(w, "Invalid latitude parameter", http.StatusBadRequest)
			return
		}
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			http.Error(w, "Invalid longitude parameter", http.StatusBadRequest)
			return
		}
		rad, err := strconv.ParseFloat(radStr, 64)
		if err != nil {
			http.Error(w, "Invalid radius parameter", http.StatusBadRequest)
			return
		}
		if typeStr != "circle" && typeStr != "square" {
			http.Error(w, "Invalid type parameter", http.StatusBadRequest)
			return
		}

		// Define the query
		query := `
			SELECT id, name, latitude, longitude, rating,
				   (6371 * acos(cos(radians($1)) * cos(radians(latitude)) *
				    cos(radians(longitude) - radians($2)) + sin(radians($1)) *
				    sin(radians(latitude)))) AS distance
			FROM spots
			WHERE %s
			ORDER BY distance ASC, rating DESC
		`

		// Determine the condition based on the type parameter
		var condition string
		if typeStr == "circle" {
			condition = fmt.Sprintf("(6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) <= $3")
		} else {
			latDiff := rad / 111111.0
			lonDiff := rad / (111111.0 * math.Cos(lat*math.Pi/180.0))
			condition = fmt.Sprintf("latitude BETWEEN $1 - %f AND $1 + %f AND longitude BETWEEN $2 - %f AND $2 + %f", latDiff, lonDiff, lonDiff, latDiff)
		}

		// Execute the query
		rows, err := db.Query(fmt.Sprintf(query, condition), lat, lon, rad)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		defer rows.Close()

		
