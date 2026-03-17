package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ayush/logistics-platform/internal/tracking"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

type OrderHandler struct {
	api.UnimplementedOrderServiceServer
	redisClient    *redis.Client
	matchingEngine *MatchingEngine
	aggregator     *tracking.Aggregator
}

func NewOrderHandler(redisClient *redis.Client) *OrderHandler {
	return &OrderHandler{
		redisClient: redisClient,
		aggregator:  tracking.NewAggregator(redisClient),
	}
}

func (h *OrderHandler) SetMatchingEngine(me *MatchingEngine) {
	h.matchingEngine = me
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
	
	// Start Assignment Flow
	if h.matchingEngine != nil {
		go h.assignmentWorkflow(order)
	}
	
	return order, nil
}

func (h *OrderHandler) assignmentWorkflow(order *api.Order) {
	ctx := context.Background()
	maxAttempts := 5
	attempt := 0

	for attempt < maxAttempts {
		attempt++
		log.Printf("Assignment attempt %d for order %s", attempt, order.OrderId)

		driverID, err := h.matchingEngine.FindDriver(ctx, order)
		if err != nil {
			log.Printf("Matching failed for order %s: %v", order.OrderId, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if driverID == "" {
			log.Printf("No drivers found for order %s, retrying...", order.OrderId)
			time.Sleep(5 * time.Second)
			continue
		}

		// Found a driver, offer them the order
		log.Printf("Offering order %s to driver %s", order.OrderId, driverID)
		
		h.UpdateOrderStatus(ctx, &api.UpdateOrderStatusRequest{
			OrderId:  order.OrderId,
			Status:   api.OrderStatus_ORDER_STATUS_MATCHED,
			DriverId: driverID,
		})

		// Wait for acceptance with timeout (30s)
		accepted := h.waitForAcceptance(ctx, order.OrderId, driverID, 30*time.Second)
		if accepted {
			log.Printf("Driver %s accepted order %s", driverID, order.OrderId)
			return
		}

		log.Printf("Driver %s did not accept order %s (timeout/rejection)", driverID, order.OrderId)
		// Reset driver_id and keep status PENDING for next search
		h.UpdateOrderStatus(ctx, &api.UpdateOrderStatusRequest{
			OrderId:  order.OrderId,
			Status:   api.OrderStatus_ORDER_STATUS_PENDING,
			DriverId: "",
		})
	}

	log.Printf("Failed to assign order %s after %d attempts", order.OrderId, maxAttempts)
	h.UpdateOrderStatus(ctx, &api.UpdateOrderStatusRequest{
		OrderId: order.OrderId,
		Status:  api.OrderStatus_ORDER_STATUS_CANCELLED,
	})
}

func (h *OrderHandler) waitForAcceptance(ctx context.Context, orderID, driverID string, timeout time.Duration) bool {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	start := time.Now()
	for time.Since(start) < timeout {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			order, _ := h.GetOrder(ctx, &api.GetOrderRequest{OrderId: orderID})
			if order.Status == api.OrderStatus_ORDER_STATUS_PICKING_UP && order.DriverId == driverID {
				return true
			}
			if order.Status == api.OrderStatus_ORDER_STATUS_CANCELLED {
				return false
			}
			// If driver rejected, we might see it as PENDING (reset by RejectOrder)
			if order.Status == api.OrderStatus_ORDER_STATUS_PENDING && order.DriverId == "" {
				return false
			}
		}
	}
	return false
}

func (h *OrderHandler) AcceptOrder(ctx context.Context, req *api.DriverOrderActionRequest) (*api.Order, error) {
	order, err := h.GetOrder(ctx, &api.GetOrderRequest{OrderId: req.OrderId})
	if err != nil {
		return nil, err
	}

	if order.Status != api.OrderStatus_ORDER_STATUS_MATCHED || order.DriverId != req.DriverId {
		return nil, fmt.Errorf("order not available for acceptance")
	}

	// Update status to PICKING_UP
	res, err := h.UpdateOrderStatus(ctx, &api.UpdateOrderStatusRequest{
		OrderId:  req.OrderId,
		Status:   api.OrderStatus_ORDER_STATUS_PICKING_UP,
		DriverId: req.DriverId,
	})
	if err != nil {
		return nil, err
	}

	// Mark driver as busy
	h.redisClient.SAdd(ctx, "busy_drivers", req.DriverId)

	return res, nil
}

func (h *OrderHandler) RejectOrder(ctx context.Context, req *api.DriverOrderActionRequest) (*api.Order, error) {
	order, err := h.GetOrder(ctx, &api.GetOrderRequest{OrderId: req.OrderId})
	if err != nil {
		return nil, err
	}

	if order.DriverId != req.DriverId {
		return nil, fmt.Errorf("not assigned to this driver")
	}

	// Reset status to PENDING
	return h.UpdateOrderStatus(ctx, &api.UpdateOrderStatusRequest{
		OrderId:  req.OrderId,
		Status:   api.OrderStatus_ORDER_STATUS_PENDING,
		DriverId: "",
	})
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
	order.DriverId = req.DriverId // Overwrite or clear

	newData, err := json.Marshal(&order)
	if err != nil {
		return nil, err
	}

	err = h.redisClient.HSet(ctx, "orders", req.OrderId, newData).Err()
	if err != nil {
		return nil, err
	}

	log.Printf("Updated order %s status to %s (Driver: %s)", req.OrderId, req.Status, req.DriverId)
	
	return &order, nil
}

func (h *OrderHandler) StreamOrderUpdates(req *api.StreamOrderUpdatesRequest, stream api.OrderService_StreamOrderUpdatesServer) error {
	ctx := stream.Context()

	// Initial Polling for driver assignment
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			order, err := h.GetOrder(ctx, &api.GetOrderRequest{OrderId: req.OrderId})
			if err != nil {
				return err
			}
			
			if err := stream.Send(order); err != nil {
				return err
			}

			if order.DriverId != "" && order.Status != api.OrderStatus_ORDER_STATUS_PENDING {
				// Driver assigned! Use Pub/Sub for movement
				return h.streamWithPubSub(ctx, order.OrderId, order.DriverId, stream)
			}
			
			if order.Status == api.OrderStatus_ORDER_STATUS_CANCELLED {
				return nil
			}
		}
	}
}

func (h *OrderHandler) streamWithPubSub(ctx context.Context, orderID, driverID string, stream api.OrderService_StreamOrderUpdatesServer) error {
	updates, err := h.aggregator.SubscribeToDriver(ctx, driverID)
	if err != nil {
		return err
	}

	for update := range updates {
		order, err := h.GetOrder(ctx, &api.GetOrderRequest{OrderId: orderID})
		if err != nil {
			return err
		}

		// Inject current location into the order object for the stream
		order.CurrentLatitude = update.Latitude
		order.CurrentLongitude = update.Longitude
		
		if err := stream.Send(order); err != nil {
			return err
		}

		if order.Status == api.OrderStatus_ORDER_STATUS_DELIVERED || order.Status == api.OrderStatus_ORDER_STATUS_CANCELLED {
			return nil
		}
	}
	return nil
}
