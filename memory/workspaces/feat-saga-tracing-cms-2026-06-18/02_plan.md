# 02 — High-Level Plan: Saga & OTel Tracing

## Roadmap

```
Phase 1: Core Infrastructure (saga + tracing helpers)
   └─ T1.1  internal/app/saga/saga.go          [NEW]
   └─ T1.2  internal/app/saga/saga_test.go     [NEW]
   └─ T1.3  pkgs/observability/otel.go         [MODIFY - EndSpan + Ctx]
   └─ T1.4  internal/middleware/otel_propagator.go [NEW]
   └─ T1.5  internal/server/server.go          [MODIFY - register middleware]
   └─ T1.6  internal/infra/messaging/nats_command_bus.go [MODIFY - span Execute+Dispatch]

Phase 2: Source Group Saga (S1, S5)
   └─ T2.1  source/register_registry.go        [MODIFY - saga.Runner]
   └─ T2.2  source/debezium_connector.go       [MODIFY - saga.Runner x2]

Phase 3: Governance + Master Saga (S2, S3, S4)
   └─ T3.1  ports/repository.go               [MODIFY - RevertSchemaTx interface]
   └─ T3.2  persistence/master/...            [MODIFY - impl RevertSchemaTx]
   └─ T3.3  governance/approve_master.go      [MODIFY - saga.Runner]
   └─ T3.4  governance/approve_schema_proposal.go [MODIFY - saga.Runner]
   └─ T3.5  master/create_master.go           [MODIFY - saga.Runner]

Phase 4: Verification & Report
   └─ T4.1  go build + vet + test toàn bộ
   └─ T4.2  report_saga_tracing_2026-06-18.md [NEW]
```

## Timeline ước tính

| Phase | Complexity | Ước tính LOC |
|-------|------------|-------------|
| 1 | Medium | ~200 |
| 2 | Medium | ~100 |
| 3 | High | ~200 |
| 4 | Low | ~80 |

## Dependencies

- Phase 2 → phụ thuộc Phase 1 (saga core)
- Phase 3 → phụ thuộc Phase 1 (saga core) + port interface mới
- Phase 4 → phụ thuộc tất cả phases trước

## DoD tổng thể

- `go build ./... EXIT=0`
- `go vet ./... EXIT=0`
- `go test ./... -count=1` → PASS
- OTel span propagation hoạt động end-to-end
- Saga compensation tự động khi bước giữa fail
