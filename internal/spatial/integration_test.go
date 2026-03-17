//go:build integration
// +build integration

package spatial

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ayush/logistics-platform/internal/ingestor"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestPhase1_EndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	redisAddr := "localhost:6379"
	if os.Getenv("REDIS_ADDR") != "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer redisClient.Close()

	// Cleanup Redis
	redisClient.FlushAll(ctx)

	streamName := "test_driver_updates"
	groupName := "test_spatial_indexer_group"
	consumerID := "test_consumer_1"

	// 1. Start gRPC Ingestor
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	handler := ingestor.NewLocationHandler(redisClient, streamName)
	api.RegisterLocationServiceServer(s, handler)

	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()
	defer s.Stop()

	// 2. Start Spatial Indexer
	indexer := NewSpatialIndexer(redisClient, streamName, groupName, consumerID)
	go func() {
		if err := indexer.Start(ctx); err != nil {
			fmt.Printf("Spatial Indexer stopped: %v\n", err)
		}
	}()

	// 3. Connect as a Driver Client
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := api.NewLocationServiceClient(conn)
	stream, err := client.UpdateLocation(ctx)
	if err != nil {
		t.Fatalf("failed to open stream: %v", err)
	}

	driverID := "driver_123"
	lat, lng := 37.7749, -122.4194 // San Francisco

	err = stream.Send(&api.LocationUpdate{
		DriverId:  driverID,
		Latitude:  lat,
		Longitude: lng,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("failed to send update: %v", err)
	}

	// Close stream and wait for response
	res, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("failed to close and recv: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success response, got: %v", res.Message)
	}

	// 4. Verify in Redis
	// Wait a bit for the indexer to process
	time.Sleep(2 * time.Second)

	// Check GEO index
	pos, err := redisClient.GeoPos(ctx, "drivers_geo", driverID).Result()
	if err != nil {
		t.Fatalf("failed to get driver pos from Redis: %v", err)
	}
	if len(pos) == 0 || pos[0] == nil {
		t.Fatalf("driver not found in drivers_geo index")
	}

	fmt.Printf("Verified driver %s in Redis GEO at (%f, %f)\n", driverID, pos[0].Latitude, pos[0].Longitude)

	// 5. Test GoOnline and Radius Search
	_, err = handler.GoOnline(ctx, &api.DriverStatusRequest{DriverId: driverID})
	if err != nil {
		t.Fatalf("failed to go online: %v", err)
	}

	// Send another update while online to trigger indexer to update online_drivers_geo
	stream, err = client.UpdateLocation(ctx)
	if err != nil {
		t.Fatalf("failed to open second stream: %v", err)
	}
	err = stream.Send(&api.LocationUpdate{
		DriverId:  driverID,
		Latitude:  lat,
		Longitude: lng,
		Timestamp: time.Now().Unix(),
	})
	stream.CloseAndRecv()

	time.Sleep(2 * time.Second)

	// Search for drivers in radius
	drivers, err := handler.GetDriversInRadius(ctx, &api.RadiusQuery{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  10,
	})
	if err != nil {
		t.Fatalf("failed to search drivers in radius: %v", err)
	}

	if len(drivers.Drivers) == 0 {
		t.Errorf("expected to find driver %s in radius search", driverID)
	} else {
		fmt.Printf("Found driver %s in radius search\n", drivers.Drivers[0].DriverId)
	}

	// 6. Test GoOffline
	_, err = handler.GoOffline(ctx, &api.DriverStatusRequest{DriverId: driverID})
	if err != nil {
		t.Fatalf("failed to go offline: %v", err)
	}

	drivers, _ = handler.GetDriversInRadius(ctx, &api.RadiusQuery{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  10,
	})
	if len(drivers.Drivers) > 0 {
		t.Errorf("expected driver %s to be removed from online search after going offline", driverID)
	}
}
