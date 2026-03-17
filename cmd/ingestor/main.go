package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/ayush/logistics-platform/internal/ingestor"
	"github.com/ayush/logistics-platform/internal/order"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		HTTPPort int `yaml:"http_port"`
		GRPCPort int `yaml:"grpc_port"`
	} `yaml:"server"`
	Redis struct {
		Addr       string `yaml:"addr"`
		StreamName string `yaml:"stream_name"`
	} `yaml:"redis"`
}

func main() {
	// ... (config loading)
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

	// 1. Start Prometheus Metrics Server (HTTP)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Printf("Starting Metrics Server on :%d/metrics...\n", cfg.Server.HTTPPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.HTTPPort), nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// 2. Initialize gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	
	// Register Location Service
	locationHandler := ingestor.NewLocationHandler(redisClient, cfg.Redis.StreamName)
	api.RegisterLocationServiceServer(s, locationHandler)

	// Register Order Service
	orderHandler := order.NewOrderHandler(redisClient)
	api.RegisterOrderServiceServer(s, orderHandler)

	fmt.Printf("Starting gRPC Services (Location + Order) on :%d...\n", cfg.Server.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
