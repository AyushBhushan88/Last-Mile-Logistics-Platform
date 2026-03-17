package order

import (
	"context"
	"log"
	"time"

	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

type MatchingEngine struct {
	redisClient     *redis.Client
	locationService api.LocationServiceClient
	orderHandler    *OrderHandler
}

func NewMatchingEngine(redisClient *redis.Client, locationService api.LocationServiceClient, orderHandler *OrderHandler) *MatchingEngine {
	return &MatchingEngine{
		redisClient:     redisClient,
		locationService: locationService,
		orderHandler:    orderHandler,
	}
}

// FindDriver finds the best driver for an order
func (m *MatchingEngine) FindDriver(ctx context.Context, order *api.Order) (string, error) {
	// 1. Initial radius search (e.g., 5km)
	radius := 5.0
	maxRadius := 20.0
	
	for radius <= maxRadius {
		log.Printf("Searching for drivers for order %s in %f km radius", order.OrderId, radius)
		
		res, err := m.locationService.GetDriversInRadius(ctx, &api.RadiusQuery{
			Latitude:  order.PickupLatitude,
			Longitude: order.PickupLongitude,
			RadiusKm:  radius,
		})
		
		if err != nil {
			return "", err
		}

		if len(res.Drivers) > 0 {
			// In a real system, we'd rank them or ask them to accept.
			// For now, we take the nearest one.
			// We should also check if the driver is currently "Available" (not on another order).
			// For simplicity, we'll assume any online driver is available.
			
			for _, d := range res.Drivers {
				// Check if driver is already busy
				isBusy, _ := m.redisClient.SIsMember(ctx, "busy_drivers", d.DriverId).Result()
				if !isBusy {
					return d.DriverId, nil
				}
			}
		}

		// Expand search if no driver found
		radius += 5.0
		time.Sleep(500 * time.Millisecond) // Don't spam
	}

	return "", nil
}

func (m *MatchingEngine) ProcessPendingOrders(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Fetch pending orders (simplified: we'd use a queue or Redis set)
			// For now, let's look for orders with status PENDING in Redis
			// This is NOT efficient for production but works for the prototype.
			
			// In a real system, we'd use Redis Streams or a worker queue.
		}
	}
}
