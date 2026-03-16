package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

type OrderHandler struct {
	api.UnimplementedOrderServiceServer
	redisClient *redis.Client
}

func NewOrderHandler(redisClient *redis.Client) *OrderHandler {
	return &OrderHandler{
		redisClient: redisClient,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *api.CreateOrderRequest) (*api.Order, error) {
	orderID := fmt.Sprintf("order_%d", time.Now().UnixNano())
	
	order := &api.Order{
		OrderId:          orderID,
		CustomerId:       req.CustomerId,
		PickupLatitude:   req.PickupLatitude,
		PickupLongitude:  req.PickupLongitude,
		DeliveryLatitude: req.DeliveryLatitude,
		DeliveryLongitude: req.DeliveryLongitude,
		Status:           api.OrderStatus_ORDER_STATUS_PENDING,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	data, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}

	err = h.redisClient.HSet(ctx, "orders", orderID, data).Err()
	if err != nil {
		return nil, err
	}

	log.Printf("Created order: %s for customer: %s", orderID, req.CustomerId)
	
	// TODO: Trigger Matching Engine
	
	return order, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *api.GetOrderRequest) (*api.Order, error) {
	data, err := h.redisClient.HGet(ctx, "orders", req.OrderId).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, err
	}

	var order api.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *api.UpdateOrderStatusRequest) (*api.Order, error) {
	data, err := h.redisClient.HGet(ctx, "orders", req.OrderId).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, err
	}

	var order api.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		return nil, err
	}

	order.Status = req.Status
	order.UpdatedAt = time.Now().Unix()
	if req.DriverId != "" {
		order.DriverId = req.DriverId
	}

	newData, err := json.Marshal(&order)
	if err != nil {
		return nil, err
	}

	err = h.redisClient.HSet(ctx, "orders", req.OrderId, newData).Err()
	if err != nil {
		return nil, err
	}

	log.Printf("Updated order %s status to %s", req.OrderId, req.Status)
	
	return &order, nil
}

func (h *OrderHandler) StreamOrderUpdates(req *api.StreamOrderUpdatesRequest, stream api.OrderService_StreamOrderUpdatesServer) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			order, err := h.GetOrder(stream.Context(), &api.GetOrderRequest{OrderId: req.OrderId})
			if err != nil {
				return err
			}
			if err := stream.Send(order); err != nil {
				return err
			}
			if order.Status == api.OrderStatus_ORDER_STATUS_DELIVERED || order.Status == api.OrderStatus_ORDER_STATUS_CANCELLED {
				return nil
			}
		}
	}
}
