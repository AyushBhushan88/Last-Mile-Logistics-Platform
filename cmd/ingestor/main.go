package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ayush/logistics-platform/internal/ingestor"
	"github.com/ayush/logistics-platform/pkg/api"
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

	// Initialize gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	handler := ingestor.NewLocationHandler(redisClient, cfg.Redis.StreamName)
	api.RegisterLocationServiceServer(s, handler)

	fmt.Printf("Starting gRPC Ingestor Service on :%d...\n", cfg.Server.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
