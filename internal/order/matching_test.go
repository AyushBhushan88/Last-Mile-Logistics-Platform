package order

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type mockLocationClient struct {
	api.LocationServiceClient
	drivers []*api.Driver
}

func (m *mockLocationClient) GetDriversInRadius(ctx context.Context, in *api.RadiusQuery, opts ...grpc.CallOption) (*api.DriverList, error) {
	return &api.DriverList{Drivers: m.drivers}, nil
}

func TestMatchingEngine_FindDriver(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	driverID := "driver_123"
	mockLoc := &mockLocationClient{
		drivers: []*api.Driver{
			{DriverId: driverID, Latitude: 37.7749, Longitude: -122.4194},
		},
	}

	engine := NewMatchingEngine(redisClient, mockLoc, nil)
	ctx := context.Background()

	order := &api.Order{
		OrderId:        "order_1",
		PickupLatitude: 37.7749,
		PickupLongitude: -122.4194,
	}

	// Case 1: Driver is available
	id, eta, err := engine.FindDriver(ctx, order)
	if err != nil {
		t.Fatalf("FindDriver failed: %v", err)
	}
	if id != driverID {
		t.Errorf("expected driver %s, got %s", driverID, id)
	}
	if eta != 0 { // Same point
		t.Errorf("expected eta 0, got %f", eta)
	}

	// Case 2: Driver is busy
	redisClient.SAdd(ctx, "busy_drivers", driverID)
	id, _, _ = engine.FindDriver(ctx, order)
	if id != "" {
		t.Errorf("expected no driver (all busy), got %s", id)
	}
}
