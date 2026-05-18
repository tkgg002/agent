# 07_status_report — Audit config-local.yml

> **Date**: 2026-05-15
> **Status**: ✅ DONE (audit + cleanup executed, build & test PASS)

## Summary

Đã audit toàn bộ 129 dòng `centralized-data-service/config/config-local.yml`. Kết luận:

- **Còn dùng (ACTIVE)**: server, db.pool tuning, systemDb, shadowDb, masterDb, controlPlane, sources.mongodb_primary (qua bridge), nats, redis, kafka, otel, đa số worker.*, jwt.secret (guard), đa số debezium.*.
- **Legacy nhưng còn parse**: db.{host,port,username,password,database,sslMode,url} — fallback DSN nếu systemDb.url rỗng.
- **DEAD (KHÔNG còn dùng theo flow hiện tại)**:
  1. `airbyte:` block (3 key) — Viper silently drop.
  2. `sources.postgres_primary` — không có caller.
  3. `worker.fetchSize` — không có caller.
  4. `worker.transformInterval` — không có caller.
  5. `worker.scanInterval` — không có caller.
  6. `jwt.expiration` — không có caller.
  7. `debezium.connectorName` — handler hardcode, không đọc cfg.

Chi tiết xem: [`report_config_local_audit_2026-05-15.md`](./report_config_local_audit_2026-05-15.md).

## Files changed

- **Source/config thay đổi**:
  - `data-hub/centralized-data-service/config/config-local.yml` — xoá 10 dòng / 7 mục DEAD (airbyte block, sources.postgres_primary, worker.{fetchSize,transformInterval,scanInterval}, jwt.expiration, debezium.connectorName). 128 → 118 lines.
- **Workspace docs (mới + cập nhật)**:
  - `00_context.md`, `01_requirements.md`, `02_plan.md` (mới — Phase audit)
  - `08_tasks_cleanup.md`, `09_tasks_solution_cleanup.md` (mới — Phase cleanup)
  - `05_progress.md` (APPEND audit log)
  - `07_status_report.md` (file này, cập nhật)
  - `report_config_local_audit_2026-05-15.md` (audit report + section "Đã thực thi")

## Service health check

- `go build ./...` → **EXIT=0** (no compile error). Worker / admin-api / sinkworker binaries build OK.
- `go test ./config/...` → **4 tests PASS** (TestUnmarshalKafka_{ScalarTopicPrefix, ListTopicPrefix, AliasTopicPrefixes, AliasUnionDedup}). 0.957s.
- **Smoke load test**: `config.NewConfig()` với YAML mới → LOAD OK, validateConfig pass, mọi ACTIVE keys giữ giá trị đúng, mọi DEAD field zero-value (đúng kỳ vọng).
- Không động vào runtime service đang chạy — cleanup chỉ là static YAML, nạp lại khi restart binary.
