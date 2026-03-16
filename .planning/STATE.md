# Project State: Last-Mile Logistics Platform

## Current Status
- **Phase:** 1 - Foundation & Real-Time Ingestion (Completed)
- **Active Task:** Starting Phase 2 - Dispatching & Matching Logic
- **Last Milestone:** High-throughput location ingestion pipeline implemented.

## Completed Milestones
- [x] Project scaffolding and Go module initialization.
- [x] gRPC API definitions for location updates.
- [x] Ingestor service with Redis Streams integration.
- [x] Spatial Indexer with H3 (hexagonal) indexing.
- [x] Docker Compose and Kubernetes manifests (scaffolding).

## Technical Decisions
1. **Spatial Index:** H3 (Hexagonal) selected over Geohash for consistent neighbor distance.
2. **Real-time:** Redis Streams for ingestion; WebSockets for delivery to clients.
3. **Workflow:** Temporal.io selected for delivery lifecycle management to ensure reliability.
4. **Database:** PostGIS for persistence and complex spatial analysis.
5. **Ingestion Pipeline:** gRPC -> Redis Streams -> Spatial Indexer (Consumer Group) -> Redis GEO / H3 Mapping.

## Key Metrics (Planned)
- Matching Latency Target: < 500ms
- Driver Ingestion Throughput: 5k/sec

## Outstanding Questions/Risks
- Need to decide between `uber/h3-go` (CGO) or a pure Go implementation for easier K8s builds.
- Integration with external maps provider (Google Maps vs. Mapbox vs. OpenStreetMap).
