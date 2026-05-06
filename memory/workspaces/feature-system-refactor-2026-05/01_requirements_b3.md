# B3 — Requirements (Full Pipeline Hardening + Add-Source-DB E2E)

**Workspace**: `feature-system-refactor-2026-05`
**Phase**: B3 (sau B1+B2 closed 2026-05-04 17:25)
**Brain decision date**: 2026-05-05 00:05 ICT
**User mandate**: "hệ thống chạy mượt, ko api lỗi, ko FE function lỗi, ko worker lỗi, follow đủ 1 vòng làm việc, add source DB mới (mariasql / mongo / postgres) có bước auto/manual để có shadow + master tương ứng. Brain ra quyết định, ko hỏi lại"

---

## 1. Functional Requirements

### FR-B3-1 — Redpanda Console
Bring up `gpay-redpanda-console` v2.7.2 (compose đã định nghĩa, container bị purge). Smoke `:18088/` → 200.

### FR-B3-2 — Cross-service health probe drift
- `cdc-cms-service/internal/service/system_health_collector.go:267` đang gọi `<workerURL>/health` (admin-api Phase F1 đã auth-gate → 401 → status=down giả). Đổi sang `/healthz` (no-auth dev probe trả `{"ok":true}`).
- `cdc-cms-service/internal/service/prom_client.go:200` gọi `<workerURL>/metrics` 401. Hai lựa chọn: (a) skip silently khi 401 thay vì coi như error; (b) inject admin token nếu config có. Brain chọn **(a)** — log debug, set status="degraded", không break overall health.

### FR-B3-3 — Makefile drift
`cdc-cms-service/Makefile` migrate target dùng `-U user -d goopay_dw` (DB không tồn tại). Sửa thành đúng credentials + DB từ `config/config-local.yml`.

### FR-B3-4 — kafka_exporter sidecar
Compose KHÔNG có service `kafka-exporter`. cms `kafkaExporterUrl: http://localhost:9308/metrics` dẫn đến alert "consumer_lag unknown". Brain quyết: clear `kafkaExporterUrl=""` (Redpanda Console v2.7.2 đã hiển thị consumer lag UI; metric automation defer).

### FR-B3-5 — SchemaAdapter auto-CREATE shadow
`internal/service/schema_adapter.go::PrepareForCDCInsertInSchema` fail nếu shadow table chưa tồn tại. Operator phải `CREATE TABLE` thủ công. Sửa: auto-create với V1 CDC cols (`_raw_data`, `_synced_at`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`) — idempotent `IF NOT EXISTS`.

### FR-B3-6 — Prune V1 legacy seed
10 hàng `legacy_*` trong `cdc_system.source_object_registry` (migration 035) gây route cache collision với V2 source name trùng. Idempotent SQL deactivate `is_active=false` + soft-stamp notes.

### FR-B3-7 — Operator add-source-DB E2E (3 engine)
**Goal**: Brain document đầy đủ 1 vòng làm việc cho operator UI:
- (1) Login `admin/admin123`
- (2) Navigate FE Wizard / SourceToMasterWizard
- (3) Khai báo source connection (host/port/db/user/password) cho 1 trong 3 engine: Mongo / MariaDB / PostgreSQL
- (4) BE auto: scan namespace, detect timestamp field, register `source_object_registry` row, create shadow_binding, ALTER schema if needed
- (5) Operator approve schema_proposal (FE list → click approve)
- (6) Brain auto-create master_binding (admin-api `/v1/masters` POST)
- (7) Brain auto-create transmute_schedule (cron mặc định */1)
- (8) Cron tick → transmute → master row visible trong DW
- (9) FE dashboard show schedule.last_status=success

Brain document mỗi engine path khác biệt ở đâu (Mongo dùng `_id` ObjectID, MariaDB dùng auto-increment + binlog, PG dùng logical replication slot).

### FR-B3-8 — Zero-error verification
- 4 service `/health|/healthz` → 200
- `/api/system/health` overall = `ok` (sau fix probe drift)
- Worker docker log 5 phút không có panic / repeated error (ngoài "table does not exist" cosmetic do legacy scheduler)
- FE root + 1 protected page (e.g. `/sources`) → 200
- Cron tick close-loop liên tục 5 phút (5×6 = 30 events)

---

## 2. Non-Functional Requirements

| ID | NFR | Mục tiêu |
|---|---|---|
| NFR-B3-A | No service downtime > 30s | Atomic restart cms-server + worker |
| NFR-B3-B | No data loss | Backup binary `*.bak` trước rebuild; SQL idempotent |
| NFR-B3-C | All change reversible | Git stash trước, commit sau verify |
| NFR-B3-D | Memory APPEND only | `05_progress.md` chỉ APPEND |
| NFR-B3-E | §12 compliance | Brain edits .md/.yml/.env/.sh, Muscle edits .go/.ts/.sql |
| NFR-B3-F | Lesson on drift | Add Global Pattern lesson cho cross-service probe drift |

---

## 3. Acceptance Criteria

| AC | Verify command | Expected |
|---|---|---|
| AC-B3-1 | `curl :18088/` | 200 |
| AC-B3-2 | `curl :8083/api/system/health \| jq .overall` | `"ok"` |
| AC-B3-3 | `curl :8083/api/system/health \| jq .cdc_pipeline.worker.status` | `"up"` |
| AC-B3-4 | `make migrate` (cdc-cms-service) | exit 0 |
| AC-B3-5 | `psql cdc_dw -c "SELECT count(*) FROM cdc_system.source_object_registry WHERE object_code LIKE 'legacy\_%' AND is_active=true"` | `0` |
| AC-B3-6 | DROP shadow table → INSERT source row → wait 60s | shadow table auto-created với V1 CDC cols + row landed |
| AC-B3-7 | 3 engine smoke (Mongo / MariaDB / PG) | mỗi engine: schedule.last_status=success + master count > 0 |
| AC-B3-8 | `docker logs gpay-cdc-worker --since 5m \| grep -E "panic\|fatal\|FATAL"` | empty |
| AC-B3-9 | DB query schedule sau 5 cron tick | 6/6 success, last_run_at < 65s ago |

---

## 4. Risk

| Risk | Mitigation |
|---|---|
| Worker restart làm mất NATS subscription | Worker sub auto-reconnect; verify by post-restart cron tick close-loop |
| schema_adapter auto-CREATE conflict với V2 SchemaManager | V1 path chỉ chạm shadow_<conn>_<engine> namespace cũ; V2 SchemaManager dùng schema khác |
| Prune legacy gây FK error | Soft deactivate `is_active=false` thay vì DELETE → an toàn |
| FE click test cần human | Document smoke endpoints data-plane thay click; user click verify cuối |
| `/metrics` skip 401 hide real auth issue | Log debug `worker /metrics auth required` để admin biết — không silent |

---

## 5. Definition of Done

- [ ] All 12 task #103-#114 status `completed`
- [ ] Report `report_phase_b3_*.md` viết với evidence từng AC
- [ ] APPEND `05_progress.md` không overwrite
- [ ] Lesson mới Global Pattern A/B/X/Y trong `lessons.md`
- [ ] Git commit (1 hoặc nhiều) các thay đổi `.go` `.sql` `.yml` `.mk`
- [ ] User check `report_phase_b3_*.md` trước khi mở B4
