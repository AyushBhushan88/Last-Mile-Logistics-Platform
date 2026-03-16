package spatial

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSpatialIndexer_ProcessMessage(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	indexer := NewSpatialIndexer(redisClient, "stream", "group", "id")

	driverID := "driver_abc"
	lat := "37.7749"
	lng := "-122.4194"

	message := redis.XMessage{
		ID: "123-0",
		Values: map[string]interface{}{
			"driver_id": driverID,
			"latitude":  lat,
			"longitude": lng,
		},
	}

	indexer.processMessage(context.Background(), message)

	// Verify GEO entry
	pos, err := redisClient.GeoPos(context.Background(), "drivers_geo", driverID).Result()
	if err != nil {
		t.Fatalf("failed to get geo pos: %v", err)
	}
	if len(pos) == 0 || pos[0] == nil {
		t.Error("expected driver position in Redis GEO")
	}

	// Verify H3 cell mapping
	cell, err := redisClient.HGet(context.Background(), "driver_cells", driverID).Result()
	if err != nil {
		t.Fatalf("failed to get H3 cell: %v", err)
	}
	if cell != "89283082803ffff" {
		t.Errorf("expected H3 cell 89283082803ffff, got %s", cell)
	}
}
