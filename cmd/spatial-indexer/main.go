package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayush/logistics-platform/internal/spatial"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Redis struct {
		Addr       string `yaml:"addr"`
		StreamName string `yaml:"stream_name"`
	} `yaml:"redis"`
	SpatialIndexer struct {
		GroupName  string `yaml:"group_name"`
		ConsumerID string `yaml:"consumer_id"`
	} `yaml:"spatial_indexer"`
}

func main() {
	// Load config
	configPath := "config/config.yaml"
	if os.Getenv("CONFIG_PATH") != "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(configFile, &cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
	})

	// Initialize Spatial Indexer
	indexer := spatial.NewSpatialIndexer(
		redisClient,
		cfg.Redis.StreamName,
		cfg.SpatialIndexer.GroupName,
		cfg.SpatialIndexer.ConsumerID,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("Starting Spatial Indexer on stream %s...\n", cfg.Redis.StreamName)
	if err := indexer.Start(ctx); err != nil {
		log.Fatalf("Spatial Indexer failed: %v", err)
	}
}
