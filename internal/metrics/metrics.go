package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LocationUpdates records the number of GPS updates processed
	LocationUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "logistics_location_updates_total",
		Help: "The total number of processed driver location updates",
	})

	// MatchingDuration records how long it takes to find a driver for an order
	MatchingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "logistics_matching_duration_seconds",
		Help:    "Histogram of driver matching latency in seconds",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	})

	// OrdersCreated records the total number of orders created
	OrdersCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "logistics_orders_created_total",
		Help: "The total number of orders created",
	})

	// ActiveDrivers records the number of online drivers (gauge)
	ActiveDrivers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "logistics_active_drivers_count",
		Help: "The current number of online drivers",
	})

	// MatchingRetries records the number of attempts to match an order
	MatchingRetries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "logistics_matching_retries_total",
		Help: "The total number of matching retries per order",
	}, []string{"order_id"})
)
