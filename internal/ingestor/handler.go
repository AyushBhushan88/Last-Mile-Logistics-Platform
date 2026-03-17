package ingestor

import (
	"context"
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

func (h *LocationHandler) GoOnline(ctx context.Context, req *api.DriverStatusRequest) (*api.LocationResponse, error) {
	err := h.redisClient.SAdd(ctx, "online_drivers", req.DriverId).Err()
	if err != nil {
		return nil, err
	}
	log.Printf("Driver %s is now online", req.DriverId)
	return &api.LocationResponse{Success: true, Message: "Online"}, nil
}

func (h *LocationHandler) GoOffline(ctx context.Context, req *api.DriverStatusRequest) (*api.LocationResponse, error) {
	err := h.redisClient.SRem(ctx, "online_drivers", req.DriverId).Err()
	if err != nil {
		return nil, err
	}
	// Also remove from online_drivers_geo to be safe
	h.redisClient.ZRem(ctx, "online_drivers_geo", req.DriverId)

	log.Printf("Driver %s is now offline", req.DriverId)
	return &api.LocationResponse{Success: true, Message: "Offline"}, nil
}

func (h *LocationHandler) GetDriversInRadius(ctx context.Context, req *api.RadiusQuery) (*api.DriverList, error) {
	// Search Redis for drivers within the specified radius in online_drivers_geo
	results, err := h.redisClient.GeoRadius(ctx, "online_drivers_geo", req.Longitude, req.Latitude, &redis.GeoRadiusQuery{
		Radius:      req.RadiusKm,
		Unit:        "km",
		WithDist:    true,
		WithCoord:   true,
		Sort:        "ASC",
		Count:       50,
	}).Result()

	if err != nil {
		log.Printf("Failed to query Redis for drivers in radius: %v", err)
		return nil, err
	}

	drivers := make([]*api.Driver, 0, len(results))
	for _, result := range results {
		drivers = append(drivers, &api.Driver{
			DriverId:    result.Name,
			Latitude:    result.Latitude,
			Longitude:   result.Longitude,
			DistanceKm: result.Dist,
		})
	}

	return &api.DriverList{Drivers: drivers}, nil
}
