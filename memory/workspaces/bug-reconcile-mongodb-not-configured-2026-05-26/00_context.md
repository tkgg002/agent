# 00_context — Reconcile scheduler skipped vì "MongoDB not configured"

**Date**: 2026-05-26
**Reporter**: User (admin@homeproxy.vn)
**Trigger logs**:
```
15:18:54 26/5/2026   reconcile   ALL   skipped   scheduler   reconCore not initialized (MongoDB not configured)
15:17:54 26/5/2026   reconcile   ALL   skipped   scheduler   reconCore not initialized (MongoDB not configured)
```
Mỗi 60s scheduler tick → activity_log ghi "skipped" → reconcile **không chạy** thực tế.

## Hệ thống
- Repo: `data-hub` (monorepo)
- Service: `centralized-data-service` (Go worker, Fiber)
- Config: `config/config-local.yml` — KHÔNG có block `mongodb:` (cfg.MongoDB.URL = "")
- Code paths liên quan:
  - `internal/server/worker_server.go:174-198` — gate init reconCore bằng `if cfg.MongoDB.URL != ""`
  - `internal/server/worker_server.go:845-879` — `runReconcileCycle` viết "skipped" khi reconCore=nil
  - `internal/service/recon_core.go:111-198` — ReconCore struct (mongoClient field DEAD — chỉ trong comment)
  - `internal/service/recon_source_agent.go:160-216` — ReconSourceAgent đã hỗ trợ multi-source qua `sa.clients[sourceURL]` map
  - `internal/service/metadata_registry_service.go:525-563` — `synthesizeLegacyTableRegistry` **không** populate `SourceURL`

## Constraints (user)
1. Đọc lesson trước (đã đọc L985, L3100).
2. Theo core /agent (GEMINI.md).
3. Chỉ làm đúng yêu cầu — không scope creep.
4. KHÔNG cheat DB hay sửa config để đạt kết quả.
5. Plan rõ ràng, có code demo.
6. Report dựa trên kết quả tính toán thực, có note file thay đổi, không láo.
7. Kiểm tra service work mới báo done.
8. Phải có `report_*.md`.

## Lessons applicable
- **L985** (2026-04-20): silent-skip pattern → đã fix WARN log + fix_hint. Không re-violate.
- **L3100** (2026-05-18): conditional subscriber gating bởi legacy `cfg.MongoDB.URL` tạo asymmetry — fix đã apply cho NATS subscribers (stub trong else). Phải áp dụng cùng nguyên tắc cho **scheduler tick**: phải work, không gate bởi legacy config.
- **L-CDC-route-empty-silent-skip-2026-05-26** (mới hôm nay): defense-in-depth layered fix pattern.

## Key insight
V2 architecture (Phase 2026-05-19+) đã có `cdc_system.connection_registry` lưu mọi source connections (mongo / postgres / mysql). Worker đã có resolver `MetadataRegistryService.GetSourceDSN(connectionCode)`. Legacy `cfg.MongoDB.URL` là band-aid duy nhất gate cả pipeline — vi phạm L3100.
