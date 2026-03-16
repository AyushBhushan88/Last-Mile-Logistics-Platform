package ingestor

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
)

type LocationHandler struct {
	api.UnimplementedLocationServiceServer
	redisClient *redis.Client
	streamName  string
}

func NewLocationHandler(redisClient *redis.Client, streamName string) *LocationHandler {
	return &LocationHandler{
		redisClient: redisClient,
		streamName:  streamName,
	}
}

func (h *LocationHandler) UpdateLocation(stream api.LocationService_UpdateLocationServer) error {
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&api.LocationResponse{
				Success: true,
				Message: "Stream closed",
			})
		}
		if err != nil {
			return err
		}

		log.Printf("Received update from driver %s: (%f, %f)", update.DriverId, update.Latitude, update.Longitude)

		// Push to Redis Stream
		err = h.redisClient.XAdd(context.Background(), &redis.XAddArgs{
			Stream: h.streamName,
			Values: map[string]interface{}{
				"driver_id": update.DriverId,
				"latitude":  update.Latitude,
				"longitude": update.Longitude,
				"timestamp": update.Timestamp,
			},
		}).Err()

		if err != nil {
			log.Printf("Failed to push to Redis: %v", err)
		}
	}
}

func (h *LocationHandler) GetDriversInRadius(ctx context.Context, req *api.RadiusQuery) (*api.DriverList, error) {
	// To be implemented in Phase 2
	return nil, fmt.Errorf("method not implemented")
}
