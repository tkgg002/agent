# 00_context.md - Context & Scope

## Project Context
- **Project**: GooPay Core Refactor 2026
- **Owner**: Brain (Antigravity) — Chairman
- **Status**: 🟡 Active — Phase 0 đang triển khai
- **Goal**: Modernize 60+ services (Node.js/Go) for Resilience, Idempotency, and Async Architecture.

## Global Links (Phải đọc cả 2 trước khi làm)
### Agent Framework
- [Conventions](../../global/conventions.md)

### GooPay-specific Memory
- [GooPay Project Context](../../global-goopay/project_context.md)
- [GooPay Tech Stack](../../global-goopay/tech_stack.md)
- [GooPay ADRs](../../global-goopay/architectural_decisions.md)

## Source Material
- Based on `work-desc/refactor2026/flow/refactor-planing-final.md` (Master Plan).

## Scope
- **Phase 0**: Audit & Prep (DB, Redis, Libs). 🟡 In Progress
- **Phase 1**: Infrastructure (Graceful shutdown, K8s). ⬜ Pending
- **Phase 2**: Resilience (Retry, Sweepers). ⬜ Pending
- **Phase 3**: Core Re-arch (NATS JetStream, Saga). ⬜ Pending
- **Phase 4**: Observability (Tracing, Metrics). ⬜ Pending
