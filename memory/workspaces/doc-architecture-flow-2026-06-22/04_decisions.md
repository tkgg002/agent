# Architectural Decisions

## ADR 1: Standardization of Centralized Data Service Architecture Documentation

### Status
Accepted

### Context
The lack of visual mapping for the 9 high-level layers and three primary ingestion flows in `centralized-data-service` made it difficult for developers to reason about data mutation paths and NATS command flow boundaries.

### Decision
* Maintain a single source of architectural truth in `centralized_data_service_architecture_flow.md`.
* Document system topology using Mermaid diagrams (TD topology and E2E sequence flows).
* Place the documentation within `agent/memory/workspaces/doc-architecture-flow-2026-06-22/`.

### Consequences
* Developers can clearly visualize the path of a CDC event from Kafka into shadow table, then via transmutation to master table.
