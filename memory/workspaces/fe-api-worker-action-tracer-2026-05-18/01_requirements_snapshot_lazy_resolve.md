# Requirements — Snapshot Now lazy resolve từ connection_registry

## Origin
User feedback (2026-05-18 17:50): "config tĩnh cho mongodb url ko đúng. vì source db đang đc connect động vào. ko thể bỏ vào env."

Bối cảnh:
- CMS UI cho phép user khai báo nhiều Mongo source khác nhau (mỗi source 1 URI riêng) qua `connection_registry`.
- `cfg.MongoDB.URL` (worker `config-local.yml`) là static — ép 1 cluster cho cả worker.
- Khi user click "Snapshot Now" từ FE → đi vào `cdc.cmd.debezium-snapshot` → worker handler `HandleDebeziumSignal` cần insert vào collection `debezium_signal` của **source mongo tương ứng với target_table**, KHÔNG phải mongo trong worker config.

## Vấn đề hiện tại

- `worker_server.go:164` gate `if cfg.MongoDB.URL != ""` → init `mongoClientShared` cho recon module.
- `worker_server.go:442` else branch (reconCore=nil) → register stub subscriber log "MongoDB not configured" cho tất cả 7 recon subject.
- → User click Snapshot Now mà worker không có config tĩnh → stub log error, không thực hiện gì.
- Refactor `reconCore` lazy init từ `connection_registry` cho cả 7 subject là quá lớn (ReconCore phụ thuộc vào ReconSourceAgent, ReconDestAgent, FullCountAggregator, BackfillSourceTsService, TimestampDetector — đều bind vào 1 mongoClient ở boot).

## Functional Requirements

- FR-1: `cdc.cmd.debezium-snapshot` + `cdc.cmd.debezium-signal` phải hoạt động NGAY CẢ KHI `cfg.MongoDB.URL` rỗng.
- FR-2: Worker phải resolve URI từ `connection_registry` per-request, không dùng config tĩnh.
- FR-3: Mỗi request mở connection mới, đóng sau khi xong (không cache client để tránh stale URI khi user update connection qua CMS).
- FR-4: Path resolve: `target_table → ResolvedSourceRoute → SourceObject.SourceConnectionID → connection_registry.host (URI)`.
- FR-5: Log dispatch path để debug: `signal_client` / `mongo_shared_client` / `mongo_lazy_resolve`.

## Out of Scope

- Refactor ReconCore/ReconHealer/FullCountAggregator/BackfillSourceTsService/TimestampDetector → lazy init từ connection_registry. (Lớn, future task.)
- Sync Fields to Shadow — user xác nhận "lấy từ bảng field đã approve" là behavior đúng.
- Đổi config schema.

## Definition of Done

- [ ] `worker_server.go`: bỏ `cdc.cmd.debezium-signal` + `cdc.cmd.debezium-snapshot` khỏi stub fallback; tạo `signalOnlyHandler` không cần reconCore khi gate đóng.
- [ ] `recon_handler.go` `HandleDebeziumSignal`: 3 nhánh (signal_client > shared mongoClient > lazy resolve). Helper `insertDebeziumSignal` + `resolveSourceMongoDSN`.
- [ ] Revert `config-local.yml` — bỏ `mongodb:` block.
- [ ] `go build ./...` PASS.
- [ ] `go vet ./...` PASS.
- [ ] `go test ./internal/handler/... ./internal/server/...` PASS.
- [ ] User test: click Snapshot Now sau restart worker → log `debezium signal: lazy resolve from connection_registry` + `debezium signal dispatched`.
