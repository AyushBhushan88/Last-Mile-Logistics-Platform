package order

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

func TestOrderHandler_CreateOrder(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	handler := NewOrderHandler(redisClient)
	ctx := context.Background()

	req := &api.CreateOrderRequest{
		CustomerId:      "cust_1",
		PickupLatitude:  37.7749,
		PickupLongitude: -122.4194,
	}

	res, err := handler.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if res.Status != api.OrderStatus_ORDER_STATUS_PENDING {
		t.Errorf("expected status PENDING, got %v", res.Status)
	}

	// Verify in redis
	data, _ := redisClient.HGet(ctx, "orders", res.OrderId).Result()
	var order api.Order
	json.Unmarshal([]byte(data), &order)
	if order.CustomerId != "cust_1" {
		t.Errorf("expected customer cust_1, got %s", order.CustomerId)
	}
}

func TestOrderHandler_AcceptReject(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	handler := NewOrderHandler(redisClient)
	ctx := context.Background()

	orderID := "order_1"
	driverID := "driver_1"
	
	// Pre-seed a matched order
	order := &api.Order{
		OrderId:  orderID,
		DriverId: driverID,
		Status:   api.OrderStatus_ORDER_STATUS_MATCHED,
	}
	data, _ := json.Marshal(order)
	redisClient.HSet(ctx, "orders", orderID, data)

	// Test Accept
	req := &api.DriverOrderActionRequest{OrderId: orderID, DriverId: driverID}
	res, err := handler.AcceptOrder(ctx, req)
	if err != nil {
		t.Fatalf("AcceptOrder failed: %v", err)
	}
	if res.Status != api.OrderStatus_ORDER_STATUS_PICKING_UP {
		t.Errorf("expected status PICKING_UP, got %v", res.Status)
	}

	// Test Reject (should move back to PENDING)
	res, err = handler.RejectOrder(ctx, req)
	if err != nil {
		t.Fatalf("RejectOrder failed: %v", err)
	}
	if res.Status != api.OrderStatus_ORDER_STATUS_PENDING {
		t.Errorf("expected status PENDING, got %v", res.Status)
	}
	if res.DriverId != "" {
		t.Errorf("expected empty driver_id, got %s", res.DriverId)
	}
}
