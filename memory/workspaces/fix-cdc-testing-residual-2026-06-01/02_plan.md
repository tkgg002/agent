# 02_plan — Fix CDC Testing Residual (2026-06-01)

## Ưu tiên (P0 → P2)
- **Wave 1 (P0)** G5/G8/G3/G1 — ✅ ĐÃ HOÀN THÀNH trước compaction.
- **Wave 2 (P1)** G4 (k6 CDC data path) → G2 (WAL auto resume).
- **Wave 3 (P2)** G7 (throttle-down) → G6 (chaos pumba).

## G4-RES — k6 CDC data path
- Script: `data-hub/centralized-data-service/scripts/load_test_cdc.js`.
- Cần extension xk6-sql + driver mysql (cdc đang đọc MariaDB shadow).
- Pattern:
  1. Setup: open 2 DB conn (source + shadow).
  2. VU loop: INSERT source row có `_probe_uuid` unique.
  3. Poll shadow table mỗi 200ms, until 5s hoặc tìm thấy row.
  4. Metric custom: `cdc_e2e_latency_ms` (Trend), `cdc_e2e_success` (Rate).
  5. Thresholds: success rate ≥99%, p95 < 3000ms.
- Build runner: `xk6 build --with github.com/grafana/xk6-sql=...
   --with github.com/grafana/xk6-sql-driver-mysql`.
- Verify: run với 5 VUs × 30s, đảm bảo thresholds GREEN.

## G2-RES — WAL auto snapshot resume
- File mới: `internal/service/wal_monitor.go`.
- Loop ticker 30s: query `pg_replication_slots` (active=false / restart_lsn lag).
- Khi vi phạm ngưỡng → publish event `snapshot.resume` (NATS) +
  inc metric `cdc_wal_snapshot_resume_total{reason}`.
- Idempotent: dedupe trong 5 phút bằng in-memory map.
- Unit test: mock slot row → expect publish + metric tăng đúng 1 lần.

## G7-RES — Throttle-down adaptive batch
- Patch `internal/handler/adaptive_batcher.go`.
- Thêm field `destHealth func() bool` (inject từ handler).
- Trong `adjust(lag)`, nếu `!destHealth()` → set batchSize = base, skip burst.
- Counter `cdc_dest_throttled_total{reason}` inc khi throttle.
- Unit test: mock health=false → batchSize không tăng dù lag cao.

## G6-RES — Pumba chaos
- File mới: `scripts/chaos_network.sh` (rewrite).
- Dùng `docker run --rm gaiaadm/pumba netem --duration 60s loss --percent 20`.
- Target container: kafka_cdc.
- Doc: README `scripts/CHAOS.md` ghi prerequisite (docker, network mode).

## Risk & Rollback
- G2/G7 sửa runtime → cần feature flag `WAL_AUTO_RESUME_ENABLED=false`
   default → bật manual sau verify.
- G4 chỉ là script test, không ảnh hưởng runtime.
- G6 chỉ chạy local/staging, không bao giờ CI prod.

## Definition of Done (per gap)
1. Code/script tồn tại trên filesystem.
2. Build + Vet + Test PASS (exit 0).
3. Có log/output thực để chứng minh.
4. APPEND `05_progress.md` workspace.
5. Update `08_tasks.md` checkbox.
