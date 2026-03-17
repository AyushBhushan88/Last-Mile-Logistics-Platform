package spatial

import (
	"math"
	"testing"
)

func TestHaversineRouting_CalculateRoute(t *testing.T) {
	routing := NewHaversineRouting(30.0) // 30 km/h

	// San Francisco to San Mateo (~30km)
	lat1, lng1 := 37.7749, -122.4194
	lat2, lng2 := 37.5630, -122.3255

	dist, duration, err := routing.CalculateRoute(lat1, lng1, lat2, lng2)
	if err != nil {
		t.Fatalf("CalculateRoute failed: %v", err)
	}

	// Distance should be roughly 24-25 km (straight line)
	expectedDist := 24.9
	if math.Abs(dist-expectedDist) > 1.0 {
		t.Errorf("expected distance roughly %f, got %f", expectedDist, dist)
	}

	// Duration should be distance / 30 * 3600
	expectedDuration := (dist / 30.0) * 3600.0
	if math.Abs(duration-expectedDuration) > 1.0 {
		t.Errorf("expected duration %f, got %f", expectedDuration, duration)
	}
}
