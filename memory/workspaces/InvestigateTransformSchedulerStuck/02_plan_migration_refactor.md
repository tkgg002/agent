# 02 — Plan: Migration System Refactor

**Phase**: `migration_refactor`
**Workspace**: `InvestigateTransformSchedulerStuck`
**Date**: 2026-05-14 15:32 (Asia/Ho_Chi_Minh)
**Owner**: Muscle (CC CLI)
**Status**: Draft — chờ user duyệt từng Phase

---

## 0. Nguyên tắc xuyên suốt

- **Core systems first**: tách rõ 3 loại migration (L1 cluster / L2 database / L3 schema). Không patch cục bộ.
- **Append-only**: KHÔNG sửa nội dung file migration đã apply trên bất kỳ env (giữ checksum + idempotency); chỉ tách / add file mới + sửa runner.
- **Rollback-friendly**: mỗi phase đảo ngược được bằng `git revert` + 1 câu SQL `DELETE FROM tracker WHERE …`.
- **Prod safety**: phase 1 unblock không sửa code; code change tách phase riêng, deploy có canary (dev → staging → prod).
- **Verify-before-done** (CLAUDE.md §3): mỗi phase có acceptance check thực tế, không suy diễn.

---

## 1. Tổng quan giải pháp

Đề xuất kiến trúc 3 lớp:

```
cdc-cms-service/
├── migrations/
│   ├── cluster_bootstrap/          ← L1 (DBA-only, KHÔNG embed)
│   │   ├── 001_pg_roles.sql        ← rename từ 005_pg_users, parameterize
│   │   ├── 002_search_path.sql     ← merge 039 + 042
│   │   └── README.md               ← runbook DBA
│   ├── *.sql                       ← L3 schema migration (giữ nguyên 50 file portable)
│   ├── embed.go                    ← chỉ embed *.sql top-level
│   └── _meta/
│       └── skip_list.go            ← const skipMigrations cho 3 version cũ
└── docs/
    └── migrations.md               ← documentation 3 layer
```

`internal/migrate/runner.go` thay đổi:
1. Đọc `clusterBootstrapMigrations` skip-list.
2. Nếu version ∈ skip-list và **chưa applied** → INSERT tracker với marker `applied_by='cluster-bootstrap'` + log warn "skipped: must run via DBA script".
3. Nếu version ∈ skip-list và **đã applied** → no-op (log debug).
4. Nếu version ∉ skip-list → flow cũ (exec body + INSERT tracker).

Tracker table mở rộng (additive):
```sql
ALTER TABLE cdc_system.schema_migrations
    ADD COLUMN IF NOT EXISTS applied_by TEXT NOT NULL DEFAULT 'runtime-migrator';
```

---

## 2. Phase breakdown

### Phase 1 — Unblock CMS prod (urgent, NO CODE CHANGE)

**Goal**: CMS prod start được trong session DBA tới — không cần wait deploy code.

**Owner**: User + DBA (Muscle soạn runbook + verify SQL).

**Steps**:

1. Muscle viết file `scripts/bootstrap_cms_db.sql` (NEW, không embed) với 3 block:
   - Tạo schema `cdc_system` + tracker bảng nếu chưa có.
   - INSERT 3 version (`005_pg_users`, `039_set_search_path`, `042_search_path_with_auth`) vào tracker với `applied_at = NOW()`. `ON CONFLICT DO NOTHING`.
   - GRANT tối thiểu cho `cdc-cms-user`: USAGE + CREATE trên schema, ALL trên tables/sequences đã có, default privileges, EXTENSION pgcrypto (cho migration 052).
2. Muscle viết runbook `scripts/bootstrap_cms_db.md` — pre-condition, command, verification query, rollback.
3. **DBA** chạy script bằng superuser trên `cdc_cms_database` prod (1 lần).
4. **User** restart CMS prod.
5. Verify: log CMS prod cho thấy `migrations done applied_now=N` với N ≥ 50 (chạy migration L3 còn lại), KHÔNG có fatal CREATEROLE.

**DoD**:
- CMS prod healthy: `/health` 200, `/ready` 200.
- Tracker prod có ≥ 53 row (tất cả migration trong embed).
- Worker (`cdc_dw`) không bị động chạm.

**Risk + rollback**:
- Nếu CMS start xong rồi fail ở migration sau (vd 052_create_cdc_jobs do thiếu pgcrypto) → DBA `CREATE EXTENSION pgcrypto;` manual hoặc thêm vào script bootstrap. Rollback = `DELETE FROM cdc_system.schema_migrations WHERE applied_by = 'cluster-bootstrap';` + revert tracker schema.

### Phase 2 — Code refactor (low-risk, dev → staging → prod)

**Goal**: CMS deploy mới ở bất kỳ env không cần Phase 1 thủ công.

**Owner**: Muscle thực thi sau khi user approve `08_tasks` + `09_tasks_solution`.

**Steps**:

1. **Tạo skip-list**: `internal/migrate/skip_list.go` với map exact filename string của 3 version.
2. **Sửa `runner.go`**:
   - Sau `loadApplied`: bổ sung loop kiểm tra skip-list, INSERT tracker với marker, log warn 1 lần / startup.
   - Bổ sung column `applied_by` qua `ensureTracker` (additive ALTER `ADD COLUMN IF NOT EXISTS`).
3. **Thêm migration mới `054_tracker_applied_by.sql`** (L3, portable): ALTER tracker column. Idempotent với `IF NOT EXISTS`.
4. **Tách thư mục `migrations/cluster_bootstrap/`** (NEW, KHÔNG embed):
   - `001_pg_roles.sql`: parameterize password qua `\set worker_password :worker_password` → `CREATE ROLE cdc_worker WITH LOGIN PASSWORD :'worker_password'`. Chạy bằng `psql -v worker_password=$CDC_WORKER_PASSWORD`.
   - `002_search_path.sql`: parameterize role name `\set role_name :role_name` → `ALTER ROLE :role_name SET search_path = cdc_system, public`.
   - `README.md`: runbook 3 step + table required env vars.
5. **Update `embed.go`**: comment đoạn `//go:embed *.sql` đảm bảo KHÔNG embed `cluster_bootstrap/`.
6. **Verify dev**:
   - Drop tracker dev (`docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "DELETE FROM cdc_system.schema_migrations WHERE version IN ('005_pg_users','039_set_search_path','042_search_path_with_auth');"` — chỉ để TEST PATH, sau đó re-insert).
   - Restart CMS dev với new binary → tracker phải có row của 3 version với `applied_by='cluster-bootstrap'`, body không exec lại.
   - Worker rebuild + restart → tick scheduler vẫn 60s gap.
7. **Test prod-like (staging hoặc dev với DB sạch)**:
   - Tạo DB mới `cdc_cms_test` + user limited (no CREATEROLE).
   - Chạy `scripts/bootstrap_cms_db.sql` (Phase 1).
   - Start CMS với env trỏ `cdc_cms_test` → migration L3 chạy ok.

**DoD**:
- CMS dev start ok với new code, `applied_now=0` cho 3 version skip.
- CMS prod (sau khi deploy binary) start ok, không cần chạy lại Phase 1.
- Tracker mọi env có `applied_by` column non-NULL.

**Risk + rollback**:
- Nếu skip-list miss 1 version (vd 042 vẫn cố exec) → fail ở dev, không lên staging/prod. Rollback = `git revert` commit.
- Nếu `ADD COLUMN IF NOT EXISTS` fail trên PG cũ → fallback `DO $$ ALTER TABLE ... ADD COLUMN applied_by; EXCEPTION WHEN duplicate_column THEN NULL; END $$`.

### Phase 3 — Secret hygiene + documentation (parallel với Phase 2)

**Goal**: Loại bỏ password khỏi git; chuẩn hoá runbook.

**Steps**:

1. **Rotate password** 3 role trên DW prod (chỉ ảnh hưởng `cdc_dw`, worker + CMS dev):
   - DBA generate password mới, store vào secret manager.
   - Update worker config + cms config qua env var.
   - Rolling restart worker + cms dev.
2. **File `005_pg_users.sql` cũ** trong `migrations/` (đã apply trên DW):
   - **KHÔNG xoá** (giữ tracker checksum). Thêm dòng comment đầu file: `-- DEPRECATED: replaced by migrations/cluster_bootstrap/001_pg_roles.sql. See docs/migrations.md.`
   - **KHÔNG sửa body** (sẽ phá checksum nếu có verify-mode).
3. **Doc `docs/migrations.md`** mới:
   - Layer model (L1/L2/L3) + ví dụ.
   - Bootstrap runbook step-by-step.
   - Skip-list rationale.
   - Rotate password flow.
4. **Lesson** ghi vào `agent/memory/global/lessons.md`:
   - Global Pattern theo CLAUDE.md §13.

**DoD**:
- `grep "PASSWORD '" cdc-cms-service/migrations/` chỉ còn match trong file cũ (đã DEPRECATED comment).
- Worker + CMS prod connect được DB sau rotate password.
- `docs/migrations.md` được merge.
- `lessons.md` có entry mới với date + Global Pattern.

**Risk + rollback**:
- Rotate password sai → service không connect. Mitigation: prepare new password, test ở dev trước, rolling restart từng instance.

### Phase 4 — Security audit + verification cuối phiên

**Goal**: Đáp ứng CLAUDE.md §10 (Security Auto-Check) + §3 (Verification Before Done).

**Steps**:

1. Muscle chạy `/security-agent` review:
   - Skip-list không bypass migration không phải L1.
   - Tracker ALTER không phá RLS hoặc default privileges.
   - Bootstrap script không leak password vào log.
2. Smoke test 4 endpoint chính sau deploy prod:
   - CMS `/health`, `/ready`, `/api/v1/source-objects/registry/1/dispatch-status`, `/api/jobs/<id>`.
   - Worker `/health`, `/ready`, `/api/v1/internal/stats`, scheduler tick gap.
3. Verify tracker đếm: `SELECT COUNT(*), COUNT(*) FILTER (WHERE applied_by='cluster-bootstrap') FROM cdc_system.schema_migrations`.
4. Append `05_progress.md` + commit lesson.

**DoD**:
- Security gate pass.
- 4 endpoint xanh trên prod.
- Worker scheduler 60s gap (xem báo cáo 1121 / 1141).
- Report `report_*.md` final với evidence runtime.

---

## 3. Sequencing + dependency

```
Phase 1 (DBA SQL) ──┐
                    ├──> Phase 4 (verify)
Phase 2 (code) ─────┤
Phase 3 (secrets) ──┘
```

- Phase 1 **CÓ THỂ chạy trước** Phase 2/3 — chính là mục tiêu unblock prod sớm.
- Phase 2 + Phase 3 **chạy song song được** vì không share file.
- Phase 4 **phải sau** cả 3 phase trên.

---

## 4. Effort estimate

| Phase | Muscle effort | DBA effort | Calendar |
|---|---|---|---|
| 1 | 30' (viết script + runbook) | 10' (chạy) | < 1h |
| 2 | 2h (code + dev verify) | 0 | 1 ngày (dev → staging → prod canary) |
| 3 | 1h (doc + lesson) | 30' (rotate password) | 0.5 ngày |
| 4 | 30' (security + smoke) | 0 | 0.5 ngày |

Tổng ≈ 4h Muscle work + 2 ngày calendar (deploy gates).

---

## 5. Open questions cho user

| # | Question | Trả lời cần để Muscle thực thi |
|---|---|---|
| Q1 | DBA có account superuser trên prod `cdc_cms_database` không? | Phase 1 needs YES |
| Q2 | Có cho phép Muscle tạo file `scripts/bootstrap_cms_db.sql` + commit Git không? | Phase 1 NO commit yet, NO push yet |
| Q3 | Phase 2 deploy theo channel nào? (canary, blue-green, rolling?) | Anh hưởng test plan |
| Q4 | Có cho rotate password 3 role trên DW cùng phiên không? | Phase 3 |
| Q5 | Secret manager hiện tại là gì? (K8s Secret, Vault, env file?) | Phase 3 cần biết để parameterize đúng |
| Q6 | Có verification window cho prod (giờ off-peak) không? | Phase 4 smoke test |

---

## 6. Cross-reference

- Lessons liên quan: L590 (hardcode), L755 (cross-service scope), L799/L803 (migration persist), L933/L953 (reconstruction vs migration anti-pattern).
- CLAUDE.md §3 (Plan & Verify), §6 (Simplicity First), §7 (Doc Set), §10 (Security Auto-Check), §13 (Lesson writing).
- Báo cáo nền: `report_2026-05-14_1141.md` (chunk size 130), `report_2026-05-14_1408.md` (CMS↔worker call map).

---

## 7. Sign-off gate

Plan PASS được khi user trả lời 6 open question + duyệt taxonomy L1/L2/L3 + duyệt skip-list approach. Sau đó Muscle mới được:

1. Soạn `08_tasks_migration_refactor.md` chi tiết.
2. Soạn `09_tasks_solution_migration_refactor.md` (code diff dự kiến).
3. Thực thi Phase 1 → 4 theo thứ tự.
