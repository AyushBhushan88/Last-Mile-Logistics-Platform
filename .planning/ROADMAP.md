# Project Roadmap: Last-Mile Logistics Platform

## Phase 1: Foundation & Real-Time Ingestion
**Goal:** Build the core pipeline for location updates and spatial indexing.
- [ ] Setup Go project structure and Docker/K8s manifests.
- [ ] Implement Location Ingestor (gRPC/WebSockets).
- [ ] Setup PostgreSQL + PostGIS and Redis.
- [ ] Implement H3-based spatial indexing service.
- [ ] **Milestone:** Driver can send GPS updates and see them reflected in the spatial index.

## Phase 2: Dispatching & Matching Logic
**Goal:** Match orders to drivers efficiently.
- [x] Implement Order Creation API.
- [x] Implement Driver GoOnline/GoOffline and Radius Search.
- [x] Build the Matching Engine (Redis Radius search).
- [ ] Integrate A* or OSRM for ETA calculations.
- [x] Implement Driver Assignment State Machine (with timeouts and retries).
- [x] **Milestone:** Order is automatically assigned to the nearest driver.

## Phase 3: Live Tracking & Customer Experience
**Goal:** Stream location data to the end-user.
- [x] Implement Customer WebSocket service for live updates.
- [x] Build the "Tracking Stream" aggregator (Redis Pub/Sub).
- [ ] Dynamic ETA updates based on live traffic/location.
- [ ] **Milestone:** Customer sees driver moving on a map in real-time.

## Phase 4: Production Readiness & Optimization
**Goal:** Scalability, Resilience, and Observability.
- [ ] Horizontal scaling with Kubernetes (HPA).
- [ ] Implement Prometheus/Grafana dashboards for logistics metrics (Matching time, Driver density).
- [ ] High-availability PostgreSQL/Redis clusters.
- [ ] **Milestone:** System handles simulated 10k driver load.
