# 09_tasks_solution — Reconcile MongoDB-not-configured

## Solution summary
Bug có 2 lớp chồng nhau:

1. **L2 — Architectural debt**: V2 `MetadataRegistryService.ReloadAll` build
   `TableRegistry` synthetic nhưng **bỏ trống** `SourceURL` field, mặc dù
   `connection_registry` đã đủ thông tin để resolve URI per-source.
2. **L1 — Legacy guard**: `worker_server.go:176 if cfg.MongoDB.URL != ""`
   gate cả init ReconCore. Khi YAML không có block `mongodb:` (V2 deployment
   không cần legacy field này), reconCore=nil → scheduler tick ghi "skipped"
   mỗi 60s + feature reconcile **dead**.

## Fix layered (2 layers)

### Layer 2 — Populate `SourceURL` từ V2 connection_registry
- `MetadataRegistryService.ReloadAll` build `connectionURIByCode` map từ
  `connections` slice đã fetch (qua `resolveSourceURIFromConn(conn)`).
- Pass URI vào `synthesizeLegacyTableRegistry(src, binding, sourceURI)`.
- Mỗi `entry.SourceURL` populate đúng cho V2 sources.

### Layer 1 — Bỏ guard `cfg.MongoDB.URL` quanh ReconCore init
- `mongoClientShared` vẫn gate bởi cfg.MongoDB.URL (legacy default client).
- ReconCore init **luôn** (defaultClient có thể nil).
- ReconSourceAgent: `sa.clients[sourceURL]` map đã sẵn sàng multi-source
  — chỉ cần entry.SourceURL non-empty để lazy-create client per-source.
- Healer / Backfill / TimestampDetector / FullCountAgg vẫn gate (chúng chưa
  refactor sang per-source URIs) — handler tự return structured error khi
  service nil (nil-check đã có sẵn ở recon_handler).

### Defense in depth — Hard-assert trong ReconSourceAgent.getClient
- `sourceURL=="" && defaultClient==nil` → return error rõ, không panic.
- Operator-facing message chỉ rõ "Verify connection_registry OR set
  cfg.MongoDB.URL" → debuggable.

## Why this layering
- 2 lớp fix tương ứng 2 failure mode độc lập:
  - Layer 2 fix gốc rễ kiến trúc (V2 chưa populate SourceURL).
  - Layer 1 fix legacy gate (cfg.MongoDB.URL không phải feature flag).
- Defense in depth hard-assert đảm bảo nếu cả 2 fix bị regress, không
  silent panic mà có error message rõ ràng.
- Tuân thủ §6 "Simplicity First & Demand Elegance":
  - KHÔNG refactor toàn bộ ReconHealer / Backfill / TsDetector / FullCountAgg
    sang per-source URIs (scope quá lớn cho bug fix này).
  - KHÔNG sửa DB / migration / YAML config.
  - Helper `resolveSourceURIFromConn` minimal refactor — chỉ tách
    method, giữ logic resolution chain nguyên vẹn.

## File touched
- `centralized-data-service/internal/service/metadata_registry_service.go`
- `centralized-data-service/internal/service/metadata_registry_service_test.go`
- `centralized-data-service/internal/server/worker_server.go`
- `centralized-data-service/internal/service/recon_source_agent.go`

## Verification
- `go build ./...` PASS.
- `go vet ./internal/...` PASS.
- `go test ./internal/service/ -count=1` PASS (0.525s).
- `go test ./internal/handler/ -count=1` PASS (3.771s).
- **Limitation**: không có DB / Mongo / NATS live trong môi trường này để
  test runtime scheduler tick. Manual smoke step (cần operator):
  1. Start worker với config-local.yml (KHÔNG có `mongodb:` block).
  2. Quan sát log `Reconciliation Core initialized default_mongo_client=false
     source_uri_resolution=per-source via connection_registry (V2)` —
     CONFIRM reconCore init.
  3. Quan sát log `V2 metadata registry reloaded connection_uris_resolved=N`
     — CONFIRM URI resolution.
  4. Đợi 60s + interval reconcile (mặc định 30 phút, có thể set thấp hơn để
     test) → activity_log KHÔNG còn ghi "skipped (MongoDB not configured)".
     Thay vào đó dispatch CheckAll → success / drift / error tuỳ V2 mongo
     connection có reach được không.

## Open follow-up (defer)
- Refactor ReconHealer / BackfillSourceTsService / TimestampDetector /
  FullCountAggregator sang per-source URIs (giống ReconSourceAgent đã hỗ trợ).
  Hiện tại chúng dùng `mongoClientShared` single default. → scope ngoài
  bug fix này.
- Cân nhắc xóa hẳn `cfg.MongoDB` config block sau khi 4 service trên migrate
  sang V2 — deprecate legacy field.
- Prometheus metric `recon_default_client_missing_total` để alert khi
  legacy fallback path bị trigger.
