# Last-Mile Logistics & Geo-Routing Platform

## Project Overview
A real-time, distributed backend system designed to dynamically match delivery orders to the nearest available drivers using geo-spatial data. The system provides live location streaming to customers and handles high-load last-mile logistics operations.

## Core Vision
To provide a highly scalable, low-latency logistics engine that optimizes delivery routes and driver allocation in real-time.

## Tech Stack
- **Language:** Go (Golang)
- **Primary Database:** PostgreSQL with PostGIS extension for geo-spatial queries.
- **Real-time / Caching:** Redis (Geo-indexing, Pub/Sub, and caching).
- **Orchestration:** Kubernetes (K8s).
- **Communication:** gRPC for internal services, WebSockets for live streaming.

## Domain
Logistics, Geo-spatial Computing, Distributed Systems.
