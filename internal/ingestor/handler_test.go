package ingestor

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type mockUpdateLocationServer struct {
	grpc.ServerStream
	ctx      context.Context
	updates  []*api.LocationUpdate
	index    int
	response *api.LocationResponse
	closed   bool
}

func (m *mockUpdateLocationServer) Context() context.Context {
	return m.ctx
}

func (m *mockUpdateLocationServer) Recv() (*api.LocationUpdate, error) {
	if m.index >= len(m.updates) {
		return nil, io.EOF
	}
	update := m.updates[m.index]
	m.index++
	return update, nil
}

func (m *mockUpdateLocationServer) SendAndClose(resp *api.LocationResponse) error {
	m.response = resp
	m.closed = true
	return nil
}

func TestLocationHandler_UpdateLocation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	streamName := "test_stream"
	handler := NewLocationHandler(redisClient, streamName)

	updates := []*api.LocationUpdate{
		{
			DriverId:  "driver_1",
			Latitude:  37.7749,
			Longitude: -122.4194,
			Timestamp: time.Now().Unix(),
		},
		{
			DriverId:  "driver_1",
			Latitude:  37.7750,
			Longitude: -122.4195,
			Timestamp: time.Now().Unix(),
		},
	}

	mockStream := &mockUpdateLocationServer{
		ctx:     context.Background(),
		updates: updates,
	}

	err = handler.UpdateLocation(mockStream)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !mockStream.closed {
		t.Error("expected stream to be closed")
	}

	if !mockStream.response.Success {
		t.Error("expected success response")
	}

	// Verify data in miniredis stream
	messages, err := redisClient.XRange(context.Background(), streamName, "-", "+").Result()
	if err != nil {
		t.Fatalf("failed to read stream from miniredis: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages in stream, got %d", len(messages))
	}

	// Verify first message content
	if messages[0].Values["driver_id"] != "driver_1" {
		t.Errorf("expected driver_id driver_1, got %v", messages[0].Values["driver_id"])
	}
}

func TestLocationHandler_GoOnline_GoOffline(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	handler := NewLocationHandler(redisClient, "test_stream")
	ctx := context.Background()
	driverID := "driver_abc"

	// Test GoOnline
	resp, err := handler.GoOnline(ctx, &api.DriverStatusRequest{DriverId: driverID})
	if err != nil {
		t.Fatalf("GoOnline failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success response, got: %v", resp.Message)
	}

	isMember, _ := redisClient.SIsMember(ctx, "online_drivers", driverID).Result()
	if !isMember {
		t.Error("expected driver to be in online_drivers set")
	}

	// Test GoOffline
	resp, err = handler.GoOffline(ctx, &api.DriverStatusRequest{DriverId: driverID})
	if err != nil {
		t.Fatalf("GoOffline failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success response, got: %v", resp.Message)
	}

	isMember, _ = redisClient.SIsMember(ctx, "online_drivers", driverID).Result()
	if isMember {
		t.Error("expected driver to be removed from online_drivers set")
	}
}

func TestLocationHandler_GetDriversInRadius(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	handler := NewLocationHandler(redisClient, "test_stream")
	ctx := context.Background()

	// Seed some online drivers in GEO set
	driverID := "driver_123"
	lat, lng := 37.7749, -122.4194
	err = redisClient.GeoAdd(ctx, "online_drivers_geo", &redis.GeoLocation{
		Name:      driverID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()
	if err != nil {
		t.Fatalf("failed to seed GEO data: %v", err)
	}

	// Test GetDriversInRadius
	req := &api.RadiusQuery{
		Latitude:  lat,
		Longitude: lng,
		RadiusKm:  10.0,
	}
	res, err := handler.GetDriversInRadius(ctx, req)
	if err != nil {
		t.Fatalf("GetDriversInRadius failed: %v", err)
	}

	if len(res.Drivers) != 1 {
		t.Errorf("expected 1 driver, got %d", len(res.Drivers))
	} else if res.Drivers[0].DriverId != driverID {
		t.Errorf("expected driver_id %s, got %s", driverID, res.Drivers[0].DriverId)
	}
}
