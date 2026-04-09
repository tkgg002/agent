# 01_requirements.md - Requirements & Voice of Customer

## Trigger
> "làm lại task này xem" (Redo this task based on refactor-planing-final.md)

## Functional Requirements
1.  **Metric-Driven Reliability**:
    -   Zero 502 errors during restarts.
    -   No "Socket Hang up" logs.
    -   Automatic data consistency (Saga/Sweepers).
2.  **Architecture Upgrade**:
    -   Move from synchronous RPC to Async Event-Driven (NATS JetStream).
    -   Implement Idempotency (Unique Index) across all transactions.
3.  **Operations**:
    -   Graceful Shutdown for both Node.js (Moleculer) and Go services.
    -   Admin tools for manual intervention.

## Constraints
- **Hybrid Stack**: Must update both Node.js and Go services.
- **Scale**: ~60 services.
- **Zero Downtime**: Must maintain availability during refactor.
