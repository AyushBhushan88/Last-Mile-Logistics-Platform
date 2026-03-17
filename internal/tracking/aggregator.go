package tracking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

type LocationUpdate struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type Aggregator struct {
	redisClient *redis.Client
}

func NewAggregator(redisClient *redis.Client) *Aggregator {
	return &Aggregator{
		redisClient: redisClient,
	}
}

// SubscribeToDriver returns a channel that receives location updates for a specific driver
func (a *Aggregator) SubscribeToDriver(ctx context.Context, driverID string) (<-chan *api.LocationUpdate, error) {
	channelName := fmt.Sprintf("driver_location:%s", driverID)
	pubsub := a.redisClient.Subscribe(ctx, channelName)

	updates := make(chan *api.LocationUpdate)

	go func() {
		defer pubsub.Close()
		defer close(updates)

		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				var update api.LocationUpdate
				if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
					log.Printf("Failed to unmarshal location update: %v", err)
					continue
				}

				updates <- &update
			}
		}
	}()

	return updates, nil
}

// GetDriverForOrder fetches the assigned driver ID for a given order from Redis
func (a *Aggregator) GetDriverForOrder(ctx context.Context, orderID string) (string, error) {
	data, err := a.redisClient.HGet(ctx, "orders", orderID).Result()
	if err != nil {
		return "", err
	}

	var order api.Order
	if err := json.Unmarshal([]byte(data), &order); err != nil {
		return "", err
	}

	if order.DriverId == "" {
		return "", fmt.Errorf("no driver assigned to order %s", orderID)
	}

	return order.DriverId, nil
}
