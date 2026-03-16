package ingestor

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ayush/logistics-platform/pkg/api"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

type mockUpdateLocationServer struct {
	grpc.ServerStream
	ctx      context.Context
	updates  []*api.LocationUpdate
	index    int
	response *api.LocationResponse
	closed   bool
}

func (m *mockUpdateLocationServer) Context() context.Context {
	return m.ctx
}

func (m *mockUpdateLocationServer) Recv() (*api.LocationUpdate, error) {
	if m.index >= len(m.updates) {
		return nil, io.EOF
	}
	update := m.updates[m.index]
	m.index++
	return update, nil
}

func (m *mockUpdateLocationServer) SendAndClose(resp *api.LocationResponse) error {
	m.response = resp
	m.closed = true
	return nil
}

func TestLocationHandler_UpdateLocation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	streamName := "test_stream"
	handler := NewLocationHandler(redisClient, streamName)

	updates := []*api.LocationUpdate{
		{
			DriverId:  "driver_1",
			Latitude:  37.7749,
			Longitude: -122.4194,
			Timestamp: time.Now().Unix(),
		},
		{
			DriverId:  "driver_1",
			Latitude:  37.7750,
			Longitude: -122.4195,
			Timestamp: time.Now().Unix(),
		},
	}

	mockStream := &mockUpdateLocationServer{
		ctx:     context.Background(),
		updates: updates,
	}

	err = handler.UpdateLocation(mockStream)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !mockStream.closed {
		t.Error("expected stream to be closed")
	}

	if !mockStream.response.Success {
		t.Error("expected success response")
	}

	// Verify data in miniredis stream
	messages, err := redisClient.XRange(context.Background(), streamName, "-", "+").Result()
	if err != nil {
		t.Fatalf("failed to read stream from miniredis: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages in stream, got %d", len(messages))
	}

	// Verify first message content
	if messages[0].Values["driver_id"] != "driver_1" {
		t.Errorf("expected driver_id driver_1, got %v", messages[0].Values["driver_id"])
	}
}
