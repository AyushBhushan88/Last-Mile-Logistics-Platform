package tracking

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

func TestAggregator_GetDriverForOrder(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	agg := NewAggregator(redisClient)
	ctx := context.Background()

	orderID := "order_123"
	driverID := "driver_456"
	order := &api.Order{
		OrderId:  orderID,
		DriverId: driverID,
	}
	data, _ := json.Marshal(order)
	redisClient.HSet(ctx, "orders", orderID, data)

	res, err := agg.GetDriverForOrder(ctx, orderID)
	if err != nil {
		t.Fatalf("GetDriverForOrder failed: %v", err)
	}
	if res != driverID {
		t.Errorf("expected driverID %s, got %s", driverID, res)
	}
}

func TestAggregator_SubscribeToDriver(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	agg := NewAggregator(redisClient)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	driverID := "driver_123"
	updates, err := agg.SubscribeToDriver(ctx, driverID)
	if err != nil {
		t.Fatalf("SubscribeToDriver failed: %v", err)
	}

	// Publish an update
	update := &api.LocationUpdate{
		DriverId:  driverID,
		Latitude:  1.0,
		Longitude: 2.0,
		Timestamp: 12345,
	}
	updateData, _ := json.Marshal(update)
	
	// Use a goroutine to publish because Subscribe might block if not handled correctly
	// But our aggregator handles it in a goroutine already.
	go func() {
		time.Sleep(100 * time.Millisecond)
		redisClient.Publish(context.Background(), "driver_location:"+driverID, updateData)
	}()

	select {
	case received := <-updates:
		if received.DriverId != driverID {
			t.Errorf("expected driver %s, got %s", driverID, received.DriverId)
		}
		if received.Latitude != 1.0 {
			t.Errorf("expected lat 1.0, got %f", received.Latitude)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for update")
	}
}
