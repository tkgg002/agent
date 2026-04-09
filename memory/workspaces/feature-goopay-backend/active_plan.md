# Active Plans — GooPay Refactor 2026

> **Last Updated**: 2026-02-10
> **Maintained by**: /context-manager

## Current Phase: GĐ0 — Rà Soát & Chuẩn Bị (Tháng 1-2)

### Tasks
- [ ] Database Schema Audit (UNIQUE INDEX cho idempotency)
- [ ] Redis Key Management (TTL cho distributed locks)
- [ ] Environment Audit (chuẩn hóa config management)
- [ ] Shared Library Upgrade (`package` repo → tách riêng theo chức năng)
- [x] **Agent Memory Refactor** (V3 Standard) → `../task-refactor-memory/`
- [x] **Workspace Protocol Enforcement** → `../task-refactor-memory-workspaces/`
- [ ] **GooPay Refactor 2026** (Major) → `../feature-refactor-2026/`



---

## Roadmap Overview

| Phase | Timeline | Status | Deliverables |
|-------|----------|--------|-------------|
| **GĐ0** | Tháng 1-2 | 🔄 In Progress | DB audit, shared lib, env standardization |
| **GĐ1** | Tháng 3-5 | ⏳ Pending | Graceful shutdown 60 services, K8s configs |
| **GĐ2** | Tháng 6-9 | ⏳ Pending | Retry policies, transaction sweeper, admin tooling |
| **GĐ3** | Tháng 10-15 | ⏳ Pending | NATS JetStream, Saga/CQRS, async migration |
| **GĐ4** | Tháng 16-18 | ⏳ Pending | Distributed tracing, logging, metrics & alerting |

## Key Decisions Pending

- **Temporal.io vs Custom Workflow Engine?**
  - Custom: Linh hoạt nhưng tốn 18 tháng build
  - Temporal: Production-ready trong 1-2 tháng, ít control
  - → Xem `temporal-migration-plan.md` và `config-driven-transaction-plan.md`

## Next Steps (sau GĐ0)
1. Pilot graceful shutdown trên nhóm ít rủi ro (notification, promotion)
2. Expand ra Gateways → Core → Finance → Connectors

## Backlog
- Admin Portal: Module quản lý stuck transactions
- Frontend: Async response handling (202 Accepted pattern)
- Config-Driven Transaction Engine (nếu không chọn Temporal)
