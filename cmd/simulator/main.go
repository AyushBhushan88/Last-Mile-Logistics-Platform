package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ayush/logistics-platform/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	driverCount := flag.Int("drivers", 100, "Number of concurrent drivers to simulate")
	ingestorAddr := flag.String("addr", "localhost:50051", "Ingestor gRPC address")
	updateInterval := flag.Duration("interval", 3*time.Second, "Location update interval")
	flag.Parse()

	log.Printf("Starting simulation for %d drivers to %s", *driverCount, *ingestorAddr)

	conn, err := grpc.Dial(*ingestorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to ingestor: %v", err)
	}
	defer conn.Close()

	client := api.NewLocationServiceClient(conn)
	
	var wg sync.WaitGroup
	for i := 0; i < *driverCount; i++ {
		wg.Add(1)
		driverID := fmt.Sprintf("sim_driver_%d", i)
		go simulateDriver(context.Background(), client, driverID, *updateInterval, &wg)
		
		// Stagger startup to avoid initial burst
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
}

func simulateDriver(ctx context.Context, client api.LocationServiceClient, driverID string, interval time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()

	// 1. Go Online
	_, err := client.GoOnline(ctx, &api.DriverStatusRequest{DriverId: driverID})
	if err != nil {
		log.Printf("Driver %s failed to go online: %v", driverID, err)
		return
	}

	// Base coordinates (San Francisco)
	lat := 37.7749 + (rand.Float64()-0.5)*0.1
	lng := -122.4194 + (rand.Float64()-0.5)*0.1

	stream, err := client.UpdateLocation(ctx)
	if err != nil {
		log.Printf("Driver %s failed to open stream: %v", driverID, err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update location slightly
			lat += (rand.Float64() - 0.5) * 0.001
			lng += (rand.Float64() - 0.5) * 0.001

			err := stream.Send(&api.LocationUpdate{
				DriverId:  driverID,
				Latitude:  lat,
				Longitude: lng,
				Timestamp: time.Now().Unix(),
			})
			if err != nil {
				log.Printf("Driver %s failed to send update: %v", driverID, err)
				return
			}
		}
	}
}
