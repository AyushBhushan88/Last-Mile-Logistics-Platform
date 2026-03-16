package spatial

import (
	"context"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/uber/h3-go/v4"
)

type SpatialIndexer struct {
	redisClient *redis.Client
	streamName  string
	groupName   string
	consumerID  string
}

func NewSpatialIndexer(redisClient *redis.Client, streamName, groupName, consumerID string) *SpatialIndexer {
	return &SpatialIndexer{
		redisClient: redisClient,
		streamName:  streamName,
		groupName:   groupName,
		consumerID:  consumerID,
	}
}

func (si *SpatialIndexer) Start(ctx context.Context) error {
	// Create consumer group if not exists
	si.redisClient.XGroupCreateMkStream(ctx, si.streamName, si.groupName, "0").Err()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			streams, err := si.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    si.groupName,
				Consumer: si.consumerID,
				Streams:  []string{si.streamName, ">"},
				Count:    10,
				Block:    0,
			}).Result()

			if err != nil {
				log.Printf("Error reading from Redis Stream: %v", err)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					si.processMessage(ctx, message)
					si.redisClient.XAck(ctx, si.streamName, si.groupName, message.ID)
				}
			}
		}
	}
}

func (si *SpatialIndexer) processMessage(ctx context.Context, message redis.XMessage) {
	driverID := message.Values["driver_id"].(string)
	latStr := message.Values["latitude"].(string)
	lngStr := message.Values["longitude"].(string)

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)

	// Convert to H3 Index
	latLng := h3.NewLatLng(lat, lng)
	cell := h3.LatLngToCell(latLng, 9) // Resolution 9

	log.Printf("Processing driver %s: (%f, %f) -> H3 Cell: %v", driverID, lat, lng, cell)

	// Update Redis GEO or H3-based set
	// For now, we'll use GEOADD for fast radius queries in Phase 2
	err := si.redisClient.GeoAdd(ctx, "drivers_geo", &redis.GeoLocation{
		Name:      driverID,
		Latitude:  lat,
		Longitude: lng,
	}).Err()

	if err != nil {
		log.Printf("Failed to update driver location in Redis: %v", err)
	}

	// Also store H3 cell mapping for quick neighbor expansion
	err = si.redisClient.HSet(ctx, "driver_cells", driverID, cell.String()).Err()
	if err != nil {
		log.Printf("Failed to update driver H3 cell in Redis: %v", err)
	}
}
