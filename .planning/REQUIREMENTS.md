# Project Requirements: Last-Mile Logistics Platform

## 1. Functional Requirements (FR)

### Driver Management & Real-Time Tracking
- **FR-1.1:** Drivers must be able to "Go Online/Offline".
- **FR-1.2:** System must ingest GPS coordinates from drivers every 3-5 seconds.
- **FR-1.3:** Drivers must be indexed spatially using H3 for fast proximity lookups.

### Order Matching (Dispatching)
- **FR-2.1:** When an order is placed, the system must find the top 5 nearest available drivers.
- **FR-2.2:** System must calculate ETA using A* or a routing engine (OSRM).
- **FR-2.3:** System must handle "Driver Acceptance" workflows with timeouts.

### Live Tracking (Customer Experience)
- **FR-3.1:** Customers must receive a live stream of the driver's location via WebSockets.
- **FR-3.2:** System must calculate and update the "Estimated Time of Arrival" (ETA) dynamically.

### Delivery Lifecycle
- **FR-4.1:** Manage delivery states: `PENDING`, `MATCHED`, `PICKED_UP`, `IN_TRANSIT`, `DELIVERED`, `CANCELLED`.
- **FR-4.2:** Persistent history of all driver movements for a specific delivery.

## 2. Non-Functional Requirements (NFR)

### Performance & Scalability
- **NFR-1.1:** Sub-500ms latency for driver matching queries.
- **NFR-1.2:** Support 10,000+ concurrent active drivers.
- **NFR-1.3:** Geographic sharding support (Kubernetes-ready).

### Reliability
- **NFR-2.1:** Delivery state must be resilient to service restarts (using Temporal or persistent DB).
- **NFR-2.2:** Real-time location stream should recover gracefully from temporary disconnects.

### Security
- **NFR-3.1:** Secure WebSocket (WSS) and gRPC (mTLS) communication.
- **NFR-3.2:** JWT-based authentication for drivers and customers.

## 3. Technical Constraints
- **Language:** Go 1.22+
- **Database:** PostgreSQL 16 + PostGIS
- **Cache:** Redis 7.0+ (Streams + Geo)
- **Orchestration:** Kubernetes (Helm charts)
