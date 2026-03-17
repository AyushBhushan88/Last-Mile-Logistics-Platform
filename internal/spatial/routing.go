package spatial

import (
	"math"
)

// RoutingEngine defines how to calculate distances and ETAs
type RoutingEngine interface {
	CalculateRoute(lat1, lng1, lat2, lng2 float64) (distanceKM float64, durationSeconds float64, err error)
}

// HaversineRouting implements RoutingEngine using Euclidean distance and a constant speed
type HaversineRouting struct {
	AverageSpeedKMH float64
}

func NewHaversineRouting(avgSpeed float64) *HaversineRouting {
	if avgSpeed <= 0 {
		avgSpeed = 30.0 // Default 30 km/h for city
	}
	return &HaversineRouting{AverageSpeedKMH: avgSpeed}
}

func (r *HaversineRouting) CalculateRoute(lat1, lng1, lat2, lng2 float64) (float64, float64, error) {
	const earthRadius = 6371.0 // KM
	
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLng := (lng2 - lng1) * (math.Pi / 180.0)
	
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadius * c
	
	// ETA = Distance / Speed
	duration := (distance / r.AverageSpeedKMH) * 3600.0 // In seconds
	
	return distance, duration, nil
}
