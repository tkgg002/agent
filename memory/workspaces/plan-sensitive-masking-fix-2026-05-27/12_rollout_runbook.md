# 12_rollout_runbook — Sensitive Masking Compliance Fix

> Created: 2026-06-01.
> Mục tiêu: định nghĩa thứ tự deploy + rollback procedure tránh race condition (R-06).

## Pre-flight checklist (T-1 ngày)

- [ ] Migration `068_add_mask_strategy.sql` đã review code + có file `068_add_mask_strategy.down.sql` (rollback).
- [ ] K8s Secret `cdc-masking-keys` đã được tạo trên cluster đích, encrypted at rest (Sealed Secrets hoặc etcd encryption ON).
- [ ] Image `centralized-data-service:v<masking>` đã build PASS + scan PASS.
- [ ] Image `cdc-cms-service:v<masking>` đã build PASS.
- [ ] Bundle `cdc-cms-web` đã build PASS.
- [ ] Backup dump `cdc_system.cdc_mapping_rules` + shadow PG (1 table sample).
- [ ] `mask_audit_log` + `mask_config_audit` retention/partition policy đã merge migration.
- [ ] Performance baseline benchmark đã chạy (M-5b).
- [ ] Runbook on-call đã share team Ops.

## Rollout order (TUYỆT ĐỐI KHÔNG ĐẢO)

### Stage 1 — Migration DDL only (no seed)
1. Apply migration `068_add_mask_strategy.sql` PHẦN DDL:
   - `CREATE TYPE cdc_system.mask_strategy AS ENUM (...)`.
   - `ALTER TABLE cdc_mapping_rules ADD COLUMN mask_strategy DEFAULT 'NONE'`.
   - `ADD COLUMN mask_options`, `mask_key_version`.
   - `CREATE TABLE mask_audit_log`, `mask_config_audit`.
2. **KHÔNG chạy UPDATE seed** (defer Stage 4).
3. Verify:
   ```sql
   SELECT mask_strategy, COUNT(*) FROM cdc_system.cdc_mapping_rules GROUP BY 1;
   -- Kỳ vọng: chỉ có "NONE" với count = total rows.
   ```

### Stage 2 — Deploy Worker code mới (đọc strategy column, default NONE)
1. Deploy image `centralized-data-service` mới (1 pod canary trước, 5 phút quan sát).
2. Verify:
   - Logs không có error `column "mask_strategy" does not exist`.
   - Metric `masking_strategy_applied_total{strategy="NONE"}` tăng theo throughput.
   - Pod CPU/mem không spike (≤ +5% baseline).
3. Rollout đầy đủ replicas (rolling update).

### Stage 3 — Deploy CMS API mới (GET/PUT mask-config + audit endpoint)
1. Deploy `cdc-cms-service` mới.
2. Smoke test:
   ```bash
   curl -X GET :8080/api/v1/admin/mapping-rules/<id>/mask-config -H "Authorization: Bearer $ADMIN"
   # Expected: 200 với mask_strategy = "NONE"
   ```
3. Deploy `cdc-cms-web` mới (tab Sensitive Masking visible nhưng chưa active strategy).

### Stage 4 — Seed default strategy (KHI Worker đã ready)
1. Chạy UPDATE seed (3 statement trong migration `068` phần seed):
   ```sql
   UPDATE cdc_mapping_rules SET mask_strategy='DROP' WHERE ... password/pin/otp/cvv/secret/token;
   UPDATE cdc_mapping_rules SET mask_strategy='HASH_HMAC' WHERE ... card_number/cccd/cmnd/account_number;
   UPDATE cdc_mapping_rules SET mask_strategy='PARTIAL' WHERE ... phone/email;
   ```
2. Verify distribution:
   ```sql
   SELECT mask_strategy, COUNT(*) FROM cdc_mapping_rules GROUP BY 1;
   -- Kỳ vọng: distribution theo ADR-003 (NONE ~70%, DROP ~10%, HASH_HMAC ~10%, PARTIAL ~10%).
   ```
3. **Trong vòng 5 phút sau seed**:
   - Verify `mask_audit_log` có record mới: `SELECT COUNT(*) FROM cdc_system.mask_audit_log WHERE masked_at > now() - interval '5 min'` > 0.
   - Verify shadow PG: `SELECT _raw_data FROM shadow.users LIMIT 5` — field password đã thành `null` (không phải `"***"`).

### Stage 5 — Backfill shadow legacy
1. Dry-run cho table lớn nhất trước:
   ```bash
   go run -tags=backfill scripts/backfill_mask.go \
     -dsn="$SHADOW_DSN" -table=shadow.users -dry-run=true -batch=1000
   ```
2. Review output sample → confirm logic ADR-013 (re-snapshot OR set null đúng policy).
3. Thực thi: `-dry-run=false`. Monitor disk I/O + Postgres lock.
4. Verify post-backfill:
   ```sql
   SELECT COUNT(*) FROM shadow.users WHERE _raw_data::text LIKE '%"***"%';
   -- Expected: 0
   ```

### Stage 6 — Compliance evidence sign-off
1. Snapshot `mask_audit_log` 24h gần nhất → export CSV.
2. Snapshot `mask_config_audit` toàn bộ → export CSV.
3. Update `report_sensitive_masking_fix_2026-05-27.md` final section.
4. /security-agent scan → confirm pass.

## Rollback procedure

### Trường hợp 1 — Worker code mới crash (Stage 2 fail)
- Rollback pod image cũ (`kubectl set image ...`).
- Migration DDL Stage 1 vẫn để (column NULLable default NONE → backward compat).
- Investigate logs, fix, re-deploy.

### Trường hợp 2 — Seed UPDATE gây sự cố (Stage 4 fail)
- Revert seed:
  ```sql
  UPDATE cdc_mapping_rules SET mask_strategy='NONE';
  ```
- Worker tự fallback → NONE → giữ giá trị gốc, không crash.
- **Lưu ý**: Field nhạy cảm sẽ KHÔNG được mask trong cửa sổ này. Cần báo Compliance team.

### Trường hợp 3 — Bỏ hoàn toàn (full rollback)
- Run `068_add_mask_strategy.down.sql`:
  ```sql
  DROP TABLE cdc_system.mask_audit_log;
  DROP TABLE cdc_system.mask_config_audit;
  ALTER TABLE cdc_mapping_rules
    DROP COLUMN mask_strategy,
    DROP COLUMN mask_options,
    DROP COLUMN mask_key_version;
  DROP TYPE cdc_system.mask_strategy;
  ```
- Deploy image Worker version cũ (giữ `"***"` literal).
- **Caveat**: Vi phạm compliance trở lại → coordinate Legal trước khi rollback toàn bộ.

### Trường hợp 4 — Backfill sai (Stage 5 fail)
- Stop script (`Ctrl-C` hoặc kill).
- Restore từ backup pre-backfill:
  ```bash
  pg_restore -d shadow_db backup-pre-backfill-<table>.dump
  ```
- Investigate logic ADR-013 vi phạm chỗ nào, fix, re-run.

## Monitoring checkpoint

| Metric | Threshold | Action |
|---|---|---|
| `masking_strategy_applied_total{strategy="NONE"}` | Drop > 30% trong 5 phút | Có thể bình thường (seed apply); xác nhận stage 4 đúng giờ. |
| `masking_apply_duration_p99_ms` | > 10ms | Investigate HMAC key load latency hoặc cache miss. |
| `mask_audit_log_writer_errors_total` | > 0 | Inspect DB connection / table missing. |
| `shadow_pg_raw_data_starred_count` (custom) | Trend > 0 sau Stage 5 | Backfill chưa cover hết → re-run cho table còn lại. |
| Pod `centralized-data-service` restart count | > 1/giờ | Rollback Stage 2. |

## Communication

| Phase | Ai báo | Báo cho |
|---|---|---|
| Pre-deploy notify | Ops lead | Eng team + Compliance |
| Stage 4 (seed) start | On-call eng | #cdc-incident |
| Stage 5 backfill done | Backfill operator | Compliance + Audit team |
| Final sign-off | Brain + Muscle | Stakeholder Luật + Ngân hàng team |

## On-call contact

- Eng on-call: rotation (xem PagerDuty schedule).
- DBA on-call: hỗ trợ rollback migration.
- Legal contact: chỉ liên hệ khi rollback toàn phần (vi phạm compliance trở lại).
