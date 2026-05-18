# Workspace: CDC MongoDB Discovery Stabilization

## Context
Stabilizing MongoDB CDC pipeline by fixing:
1. Field scanning "record not found" errors (ID resolution).
2. "Snapshot Now" button failures (Debezium signal path).
3. Automated synchronization when data source changes (NATS trigger).

## Tech Stack
- Golang (Worker, CMS)
- MongoDB (Source)
- NATS (Command Bus)
- Kafka Connect (Debezium)
