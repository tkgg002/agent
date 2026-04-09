# 00_context.md - Dependency Analysis for GooPay

## Project Context
- **Project**: GooPay (60+ microservices)
- **Goal**: Analyze if the system suffers from "Spaghetti Dependency".
- **Workspace**: `agent/memory/workspaces/task-goopay-dependency-analysis`

## Scope
1.  Identify service boundaries.
2.  Analyze inter-service dependencies (via NATS/Màu, Database sharing).
3.  Analyze intra-service dependencies (Circular imports, modularity).
4.  Report findings and potential refactorings.

## Definition of Done (DoD)
- Clear report on current dependency state.
- Evidence of "spaghetti" (if any).
- Recommendations for decoupling.
