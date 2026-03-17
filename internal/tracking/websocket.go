package tracking

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for prototype
	},
}

type WebSocketHandler struct {
	aggregator *Aggregator
}

func NewWebSocketHandler(aggregator *Aggregator) *WebSocketHandler {
	return &WebSocketHandler{
		aggregator: aggregator,
	}
}

func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		http.Error(w, "order_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 1. Get driver for order
	driverID, err := h.aggregator.GetDriverForOrder(ctx, orderID)
	if err != nil {
		log.Printf("Failed to get driver for order %s: %v", orderID, err)
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	// 2. Subscribe to driver updates
	updates, err := h.aggregator.SubscribeToDriver(ctx, driverID)
	if err != nil {
		log.Printf("Failed to subscribe to driver %s: %v", driverID, err)
		return
	}

	log.Printf("Customer tracking order %s (driver %s) via WebSocket", orderID, driverID)

	// 3. Forward updates to WebSocket
	for update := range updates {
		if err := conn.WriteJSON(update); err != nil {
			log.Printf("Failed to send location update to WebSocket: %v", err)
			break
		}
	}
}
