# 08_tasks_cleanup — Checklist xóa DEAD keys

## Tasks

- [ ] T1: Xóa `airbyte:` block (3 sub-key: apiUrl, clientId, clientSecret).
- [ ] T2: Xóa `sources.postgres_primary` (giữ `sources.mongodb_primary`).
- [ ] T3: Xóa `worker.fetchSize`.
- [ ] T4: Xóa `worker.transformInterval`.
- [ ] T5: Xóa `worker.scanInterval`.
- [ ] T6: Xóa `jwt.expiration` (giữ `jwt.secret` vì validateConfig require).
- [ ] T7: Xóa `debezium.connectorName` (handler hardcode, cfg field không reader).
- [ ] V1: `go build ./...` PASS trong `centralized-data-service/`.
- [ ] V2: `go test ./config/...` PASS.
- [ ] D1: APPEND `05_progress.md` với diff trước/sau.
- [ ] D2: Cập nhật `report_config_local_audit_2026-05-15.md` thêm section "Đã thực thi".
- [ ] D3: Cập nhật `07_status_report.md`.

## KHÔNG xoá (giữ lại với lý do)

- `db.{host,port,username,password,database,sslMode,url}`: LEGACY fallback, `validateConfig` chấp nhận `hasLegacy OR hasSplit`. Hiện tại `systemDb.url` đầy đủ → fallback không kick in, nhưng giữ để backwards-compat (chưa có verb của user yêu cầu prune legacy DSN).
- `db.maxOpenConn`, `db.maxIdleConn`, `db.connMaxLifetime`: ACTIVE pool tuning.
- `jwt.secret`: `validateConfig:447-449` require non-empty → nếu xoá thì boot fail. Giá trị placeholder `change-me-in-production` chỉ bị reject khi `server.mode == "production"` (hiện tại `mode: worker` → pass). Cảnh báo bảo mật đã ghi ở report — chưa cleanup vì cần verb riêng từ user.
- `nats.user`, `nats.pass`: KHÔNG có trong YAML (creds inline trong URL) — no-op.
- `debezium.connectorStatusUrl`: KHÔNG có trong YAML local nhưng có trong production (`config-production.yml:92`) — giữ struct field, không sửa YAML local.
