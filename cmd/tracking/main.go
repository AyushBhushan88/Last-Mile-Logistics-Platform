package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ayush/logistics-platform/internal/tracking"
	"github.com/redis/go-redis/v9"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	aggregator := tracking.NewAggregator(redisClient)
	wsHandler := tracking.NewWebSocketHandler(aggregator)

	http.Handle("/ws/track", wsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Tracking service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start tracking service: %v", err)
	}
}
