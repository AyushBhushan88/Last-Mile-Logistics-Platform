# Project State: Last-Mile Logistics Platform

## Current Status
- **Phase:** 2 - Dispatching & Matching Logic (In Progress)
- **Active Task:** Integrating Matching Engine with Order Creation
- **Last Milestone:** Initial matching engine implemented using Redis Radius Search.

## Completed Milestones
- [x] Project scaffolding and Go module initialization.
- [x] gRPC API definitions for location and order management.
- [x] Ingestor service with Redis Streams integration.
- [x] Spatial Indexer with H3 (hexagonal) indexing.
- [x] Implement GetDriversInRadius and Online/Offline status tracking.
- [x] Basic Matching Engine that assigns nearest online driver to a pending order.

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
