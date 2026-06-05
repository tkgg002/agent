# 00_context — OTel Log Sampling Fix

## Triệu chứng
User báo: SigNoz chỉ nhận được subset log so với stdout worker. Ví dụ `introspect.mongo.collections.start` xuất hiện trên SigNoz nhưng matching `.ok` thì không. Hàng loạt log periodic (`V2 metadata registry reloaded`, `discovered kafka topics`, `batch upsert ok`, `schema drift detected`) đều mất trên SigNoz.

## Root cause xác định
File: `centralized-data-service/pkgs/observability/otel.go`
- `severityAwareCore.Check` (line 161-180): nếu ratio < 1, dice roll `rng.Float64() < ratio`; trượt → return `ce` (CheckedEntry pointer) không gọi `s.Core.Check` → entry KHÔNG forward sang inner exporter.
- Config `config-local.yml:60` set `info: 0.1` → 90% Info bị drop tại wrapper, trước cả khi tới OTLP HTTP exporter.
- `config-production.yml:84` set `info: 0.05` → 95% drop.
- Console branch KHÔNG được wrap (xem comment otel.go:430-431) → stdout đầy đủ. Đó là lý do stdout vs SigNoz divergent.

## Loại trừ
- KHÔNG phải fallback mute path: user không thấy log `OTel log sink degraded — switching to console-only` trên stdout → `exportErrorTracker` chưa hit threshold 10 errors/min. OTLP localhost:4318 healthy.
- KHÔNG phải buffer overflow: BatchProcessor `MaxQueueSize=256*1024=262144`, `ExportMaxBatchSize=512`, `ExportInterval=5s` — capacity dư thừa cho local.

## Scope phase này
1. **Option 1**: Set `info: 1.0` cho local + sample. KHÔNG động production (giữ 0.05 vì cost-sensitive).
2. **Option 3**: Extend `severityAwareCore` để bypass sampling cho entry có field `audit=true`. Mute vẫn applies.
3. **Option 2**: Gắn `zap.Bool("audit", true)` vào 5 milestone Info log:
   - `internal/handler/command_handler.go:1195` `introspect.mongo.databases.start`
   - `internal/handler/command_handler.go:1209` `introspect.mongo.databases.ok`
   - `internal/handler/command_handler.go:1266` `introspect.mongo.collections.start`
   - `internal/handler/command_handler.go:1321` `introspect.mongo.collections.ok`
   - `internal/service/schema_inspector.go:162` `schema drift detected (batch summary)`

## Out of scope
- Không động `batch upsert ok` (high volume, sampling đúng mục đích).
- Không động `V2 metadata registry reloaded` (every 60s, sampling đúng mục đích).
- Không động `discovered kafka topics` (volume trung bình).
- Không refactor zap bridge cấu trúc.
- Không thêm unit test mới (defer).
