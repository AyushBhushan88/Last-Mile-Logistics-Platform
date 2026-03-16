# Geo-Routing & Last-Mile Logistics Research

## 1. Spatial Indexing: Why H3?
For real-time driver matching, **H3 (Hexagonal Hierarchical Geospatial Indexing)** is the industry standard for ride-sharing and logistics (Uber, DoorDash).

- **Hexagons vs. Geohash:** Unlike Geohash (rectangles), hexagons have only one distance between a cell center and its neighbors. This eliminates "edge cases" where points are geographically close but have different hash prefixes.
- **Consistent Neighbors:** In a hexagonal grid, all 6 neighbors are equidistant. This simplifies "expansion" searches when looking for drivers in neighboring cells.
- **Go Implementation:** `uber/h3-go` (via CGO) or pure Go alternatives like `h3-go`.

## 2. Routing & Pathfinding
### Point-to-Point (ETA Calculation)
- **A* (A-Star):** More efficient than Dijkstra for point-to-point as it uses heuristics (Euclidean distance) to guide the search towards the goal.
- **OSRM / Valhalla:** For production, integrating with an Open Source Routing Machine (OSRM) or Valhalla (via MapLibre) is often better than building from scratch for road-network awareness.

### Multi-Stop Optimization (VRP)
- **Vehicle Routing Problem (VRP):** NP-hard problem optimizing a fleet across multiple stops with constraints (time windows, capacity).
- **Go Tooling:** `Nextmv` is the leading Go-native solver for VRP.

## 3. Real-Time Architecture in Go
### Ingestion & Streaming
- **WebSockets:** Best for bi-directional live updates between the Driver App and the Backend.
- **Redis Streams:** High-throughput, low-latency message bus for coordinate updates.

### Distributed State & Orchestration
- **Temporal.io:** Highly recommended for Go projects to manage the long-running "Delivery Lifecycle" (Order Created -> Driver Matched -> Pickup -> Transit -> Delivered). It handles retries, state persistence, and timeouts out of the box.
- **Redis GEO:** Excellent for fast "Point in Radius" queries using `GEOADD` and `GEORADIUS`.

## 4. Performance Benchmarks (Go)
- Go's concurrency model (Goroutines) allows handling tens of thousands of concurrent WebSocket connections on a single node.
- Memory-efficient H3 indexing enables sub-100ms driver matching even with large fleets.
