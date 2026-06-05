# 09_tasks_solution.md — Muscle handoff (REVISED)

## Task
Snapshot bypass cache, query `mapping_rule_v2` trực tiếp. Patch theo `03_implementation.md §1-3`.

## File
- `centralized-data-service/internal/handler/snapshot_runner_handler.go` (~10 LOC)
- `centralized-data-service/internal/service/dynamic_mapper.go` (~10 LOC)

## DoD
- [ ] Build pass: `go build ./...`
- [ ] Unit test pass: `go test ./internal/handler/... ./internal/service/...`
- [ ] Restart `centralized-data-service`.
- [ ] Approve 1 rule cho source 66 (không cần restart CMS).
- [ ] Trigger snapshot.v2 source 66 → log show `snapshot.mapping_rules.loaded count=N`.
- [ ] Shadow table có column rule mới.

## Lý do approach này (vs approach cũ)
- **Snapshot là one-shot, low-frequency** → query DB direct = always fresh, không phụ thuộc cache invalidation.
- **Realtime CDC vẫn dùng cache** → high-throughput không bị động.
- **Không cross-service**: CMS không động → không cần coordinate deploy 2 service.
- **NATS không động** → không cần backwards-compat.
