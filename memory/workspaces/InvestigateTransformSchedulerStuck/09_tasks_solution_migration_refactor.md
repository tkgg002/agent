# 09 — Tasks Solution: Migration System Refactor (Technical Design)

**Phase**: `migration_refactor`
**Workspace**: `InvestigateTransformSchedulerStuck`
**Date**: 2026-05-14 15:32 (Asia/Ho_Chi_Minh)
**Status**: Draft — design dự kiến, **CHƯA execute**.

> File này chứa hồ sơ kỹ thuật cụ thể (code diff dự kiến, SQL snippet). User duyệt → Muscle áp dụng theo `08_tasks_migration_refactor.md`.

---

## 1. Phase 1 artifact — `scripts/bootstrap_cms_db.sql`

Đường dẫn: `cdc-cms-service/scripts/bootstrap_cms_db.sql` (NEW, KHÔNG embed).

```sql
-- ============================================================
-- bootstrap_cms_db.sql — One-shot DBA script.
-- Run BY: cluster superuser, BEFORE first CMS startup against a
--          fresh control-plane database.
-- Usage: psql -v ON_ERROR_STOP=1 -1 \
--             -h <host> -p <port> -U <superuser> \
--             -d <cms_db_name> \
--             -v cms_user=cdc-cms-user \
--             -f bootstrap_cms_db.sql
-- ============================================================

\if :{?cms_user}
\else
  \echo 'ERROR: -v cms_user=<role> required'
  \quit
\endif

BEGIN;

-- 1. Schema + tracker bảng
CREATE SCHEMA IF NOT EXISTS cdc_system;

CREATE TABLE IF NOT EXISTS cdc_system.schema_migrations (
    version    VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    applied_by TEXT         NOT NULL DEFAULT 'runtime-migrator'
);

-- Additive ALTER cho env đã có bảng nhưng chưa có column
ALTER TABLE cdc_system.schema_migrations
    ADD COLUMN IF NOT EXISTS applied_by TEXT NOT NULL DEFAULT 'runtime-migrator';

-- 2. Mark 3 cluster-bootstrap migrations là applied (skip body)
INSERT INTO cdc_system.schema_migrations (version, applied_by) VALUES
  ('005_pg_users',              'cluster-bootstrap'),
  ('039_set_search_path',       'cluster-bootstrap'),
  ('042_search_path_with_auth', 'cluster-bootstrap')
ON CONFLICT (version) DO NOTHING;

-- 3. Quyền tối thiểu cho cms_user để chạy migration L3 còn lại
--    (CREATE TABLE / INDEX / FUNCTION / EXTENSION).
--    KHÔNG cấp SUPERUSER, KHÔNG cấp CREATEROLE.
GRANT USAGE, CREATE ON SCHEMA cdc_system TO :"cms_user";
GRANT USAGE, CREATE ON SCHEMA public     TO :"cms_user";

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA cdc_system, public TO :"cms_user";
GRANT USAGE, SELECT                   ON ALL SEQUENCES IN SCHEMA cdc_system, public TO :"cms_user";
GRANT EXECUTE                         ON ALL FUNCTIONS IN SCHEMA cdc_system, public TO :"cms_user";

ALTER DEFAULT PRIVILEGES IN SCHEMA cdc_system GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"cms_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public     GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"cms_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA cdc_system GRANT USAGE, SELECT ON SEQUENCES TO :"cms_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public     GRANT USAGE, SELECT ON SEQUENCES TO :"cms_user";

-- 4. Extension cần cho migration 052 (gen_random_uuid)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 5. Verification block
DO $$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count
      FROM cdc_system.schema_migrations
     WHERE applied_by = 'cluster-bootstrap';
    IF v_count <> 3 THEN
        RAISE EXCEPTION 'Bootstrap verification failed: expected 3 cluster-bootstrap rows, got %', v_count;
    END IF;
END $$;

COMMIT;

\echo '✅ bootstrap_cms_db.sql OK — CMS can now start.'
```

**Rationale**:
- `\if :{?cms_user}` — fail-fast nếu thiếu `-v` (tránh GRANT empty).
- `BEGIN..COMMIT` bao trùm cả thao tác → rollback tự động nếu fail giữa chừng.
- `ON CONFLICT DO NOTHING` đảm bảo re-run idempotent.
- `IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` chịu cả case bảng đã tồn tại từ deploy cũ.
- KHÔNG cấp `SUPERUSER`/`CREATEROLE` cho `cms_user` (least privilege).
- Verification block raise EXCEPTION → BEGIN rollback nếu count sai → tracker không bị partial state.

---

## 2. Phase 2 artifact — code change

### 2.1 `internal/migrate/skip_list.go` (NEW)

```go
// Package migrate — skip-list cho cluster-bootstrap migrations.
//
// 3 file dưới đây tạo role / GRANT cross-database / ALTER ROLE — chỉ
// có cluster superuser mới chạy được. Embed runtime migrator KHÔNG
// được exec body của chúng; chỉ record vào tracker với marker
// 'cluster-bootstrap' nếu chưa có.
//
// Các env đã apply trước đó (vd cdc_dw từ thời 2026-03) có row với
// applied_by='runtime-migrator' — runner detect "đã applied" sẽ skip
// tự nhiên, không re-record.
//
// Khi thêm version mới thuộc loại cluster-bootstrap, thêm vào map này
// + tạo file tương ứng trong migrations/cluster_bootstrap/ + KHÔNG
// thêm file vào migrations/*.sql top-level (sẽ bị embed).
package migrate

// ClusterBootstrap chứa exact filename (không có .sql suffix) của các
// migration phải chạy ngoài runtime app. Tra cứu O(1).
var ClusterBootstrap = map[string]bool{
	"005_pg_users":              true,
	"039_set_search_path":       true,
	"042_search_path_with_auth": true,
}
```

### 2.2 `internal/migrate/runner.go` — diff dự kiến

```diff
@@ runner.go: const trackerTable = "cdc_system.schema_migrations"
+
+// clusterBootstrapMarker là giá trị của cột applied_by trên tracker khi
+// runtime migrator detect 1 version thuộc skip-list nhưng tracker chưa
+// có row. Phân biệt với 'runtime-migrator' (default).
+const clusterBootstrapMarker = "cluster-bootstrap"

@@ func ensureTracker
 	if _, err := conn.ExecContext(ctx, `
 		CREATE TABLE IF NOT EXISTS `+trackerTable+` (
 			version    VARCHAR(255) PRIMARY KEY,
 			applied_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
 		)`); err != nil {
 		return fmt.Errorf("migrate: create tracker table: %w", err)
 	}
+	// Additive: thêm column applied_by nếu chưa có (env cũ).
+	if _, err := conn.ExecContext(ctx, `
+		ALTER TABLE `+trackerTable+`
+		ADD COLUMN IF NOT EXISTS applied_by TEXT NOT NULL DEFAULT 'runtime-migrator'
+	`); err != nil {
+		return fmt.Errorf("migrate: ensure applied_by column: %w", err)
+	}
 	return nil

@@ func Run, in loop over files
 	for _, name := range files {
 		version := strings.TrimSuffix(name, ".sql")
 		if applied[version] {
 			continue
 		}
+		if ClusterBootstrap[version] {
+			if err := recordSkippedBootstrap(ctx, conn, version); err != nil {
+				return err
+			}
+			logger.Warn("migration skipped (cluster-bootstrap)",
+				zap.String("version", version),
+				zap.String("hint", "run cdc-cms-service/migrations/cluster_bootstrap/*.sql via DBA before service start"))
+			pending++ // count as work done (recorded) for log clarity
+			continue
+		}
 		body, err := fs.ReadFile(migrations.Files, name)
 		...
 	}

+// recordSkippedBootstrap inserts tracker row WITHOUT executing body.
+// Marker 'cluster-bootstrap' distinguishes from runtime-applied rows
+// in audit query and dashboards.
+func recordSkippedBootstrap(ctx context.Context, conn *sql.Conn, version string) error {
+	_, err := conn.ExecContext(ctx,
+		`INSERT INTO `+trackerTable+` (version, applied_by) VALUES ($1, $2)
+		 ON CONFLICT (version) DO NOTHING`,
+		version, clusterBootstrapMarker)
+	if err != nil {
+		return fmt.Errorf("migrate: record skipped %s: %w", version, err)
+	}
+	return nil
+}
```

**Diff size**: ~30 dòng (CLAUDE.md §6 minimal impact).

### 2.3 `migrations/054_tracker_applied_by.sql` (NEW)

```sql
-- 054_tracker_applied_by.sql
-- Additive column trên tracker cho phép phân biệt nguồn migration
-- (runtime-migrator vs cluster-bootstrap). Idempotent qua
-- ADD COLUMN IF NOT EXISTS. Phục vụ audit + dashboard.

ALTER TABLE cdc_system.schema_migrations
    ADD COLUMN IF NOT EXISTS applied_by TEXT NOT NULL DEFAULT 'runtime-migrator';
```

**Lưu ý**: Cùng nội dung với `ALTER TABLE` trong `ensureTracker` (2.2) và `bootstrap_cms_db.sql` (1) — tam giác đảm bảo column tồn tại bất kể luồng nào chạy trước.

### 2.4 `migrations/cluster_bootstrap/001_pg_roles.sql` (NEW)

```sql
-- ============================================================
-- 001_pg_roles.sql — Cluster bootstrap (DBA-only, NOT embedded).
-- Replaces legacy migrations/005_pg_users.sql.
--
-- Usage:
--   psql -v ON_ERROR_STOP=1 -1 \
--        -h <host> -p <port> -U <superuser> -d <dw_db> \
--        -v worker_password=$CDC_WORKER_PASSWORD \
--        -v cms_password=$CMS_SERVICE_PASSWORD \
--        -v readonly_password=$CDC_READONLY_PASSWORD \
--        -v dw_db_name=cdc_dw \
--        -f 001_pg_roles.sql
-- ============================================================

\if :{?worker_password}\else \echo 'ERROR: worker_password required' \quit \endif
\if :{?cms_password}\else \echo 'ERROR: cms_password required' \quit \endif
\if :{?readonly_password}\else \echo 'ERROR: readonly_password required' \quit \endif
\if :{?dw_db_name}\else \echo 'ERROR: dw_db_name required' \quit \endif

BEGIN;

CREATE SCHEMA IF NOT EXISTS cdc_system;

-- 1. cdc_worker
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'cdc_worker') THEN
    EXECUTE format('CREATE ROLE cdc_worker WITH LOGIN PASSWORD %L', :'worker_password');
  ELSE
    EXECUTE format('ALTER ROLE cdc_worker WITH PASSWORD %L', :'worker_password');
  END IF;
END $$;

EXECUTE format('GRANT CONNECT ON DATABASE %I TO cdc_worker', :'dw_db_name');
-- ... (full GRANT block giống 005 cũ, nhưng dw_db_name parameterized)

-- 2. cms_service (same pattern)
-- 3. cdc_readonly (same pattern)

COMMIT;
```

(Đầy đủ trong file thật khi Muscle execute Phase 2.5.)

### 2.5 `migrations/cluster_bootstrap/002_search_path.sql` (NEW)

```sql
-- ============================================================
-- 002_search_path.sql — Sets default search_path for the
-- DW-owner role. Replaces 039_set_search_path.sql + 042_search_
-- path_with_auth.sql.
--
-- Usage:
--   psql -v ON_ERROR_STOP=1 \
--        -v owner_role=gpay_admin \
--        -f 002_search_path.sql
-- ============================================================

\if :{?owner_role}\else \echo 'ERROR: owner_role required' \quit \endif

ALTER ROLE :"owner_role" SET search_path = cdc_system, public;

\echo 'OK: search_path set for ' :owner_role
```

### 2.6 `migrations/embed.go` — verify

Hiện tại: `//go:embed *.sql` — chỉ match top-level. Thư mục `cluster_bootstrap/` con không bị match. Có thể thêm comment khẳng định:

```go
// IMPORTANT: chỉ embed *.sql top-level. Thư mục cluster_bootstrap/
// chứa DBA-only scripts cần parameterize qua psql -v, không phù hợp
// runtime migrator.
//
//go:embed *.sql
var Files embed.FS
```

---

## 3. Phase 3 artifact — secret + doc

### 3.1 Comment deprecate (legacy file)

Đầu file `005_pg_users.sql`, `039_set_search_path.sql`, `042_search_path_with_auth.sql` thêm 1 dòng đầu **TRƯỚC** dòng `BEGIN;` hiện có:

```sql
-- DEPRECATED 2026-05-14: superseded by migrations/cluster_bootstrap/*.
-- Body retained for tracker checksum stability on environments already
-- applied. New deploys: file is skip-listed in internal/migrate/skip_list.go.
```

Lý do: KHÔNG sửa body để không phá idempotency của các env đã apply (lesson L755 cross-service scope).

### 3.2 `docs/migrations.md` skeleton

```markdown
# CMS Service Migrations

## 3 Layer Model

### L1 — Cluster Bootstrap
... (role, GRANT cross-DB, ALTER ROLE) ...

### L2 — Database Bootstrap
... (CREATE DATABASE, CREATE EXTENSION) ...

### L3 — Schema Migration
... (table, index, function — runtime-migrator handles) ...

## Bootstrap Runbook (env mới)

1. DBA tạo DB + user limited.
2. Chạy `cluster_bootstrap/001_pg_roles.sql` (chỉ trên DW).
3. Chạy `cluster_bootstrap/002_search_path.sql` (chỉ trên DW).
4. Chạy `scripts/bootstrap_cms_db.sql` (chỉ trên control-plane DB).
5. Start CMS — L3 migration tự áp.

## Skip-list rationale
... (link đến lesson + report 1408 + report mới) ...

## Rotate Password
... (procedure step-by-step) ...
```

### 3.3 Lesson append (vào `agent/memory/global/lessons.md`)

Format CLAUDE.md §13 (Global Pattern, dùng biến):

```markdown
## [2026-05-14] Global Pattern [A embeds cluster-level migration M into runtime app B's auto-migrator] → Result Y fatal on isolated database D

- **Trigger**: CMS service runtime migrator auto-apply file `005_pg_users.sql` (CREATE ROLE + GRANT cross-DB + hardcode password) trên control-plane DB tách riêng → fail SQLSTATE 42501 (permission denied to create role) — service không start được trên prod.
- **Root Cause**: Migration system trộn 3 lifecycle khác nhau (cluster-bootstrap / DB-bootstrap / schema) vào 1 thư mục + 1 runtime applier. Cluster-bootstrap yêu cầu SUPERUSER/CREATEROLE; service runtime user chỉ có CREATE TABLE → mismatch quyền là tất yếu khi deploy sang env đa-DB.
- **Global Pattern [A embeds resource R requiring privilege P into auto-applier of service B] → Result Y fail nếu service B chạy với user thiếu P**: Áp dụng cho CREATE ROLE / CREATE DATABASE / CREATE EXTENSION (yêu cầu owner/superuser) / GRANT cross-database (yêu cầu owner database đích).
- **Đúng**:
  1. Tách thư mục `cluster_bootstrap/` (DBA-only, không embed) khỏi `*.sql` top-level (runtime-embed).
  2. Runner có skip-list explicit + log warn (không silent skip — vi phạm L63 silent-skip masking).
  3. File legacy giữ nguyên body + thêm comment DEPRECATED → không phá checksum env đã apply.
  4. Tracker thêm column `applied_by` phân biệt nguồn → audit trail rõ.
  5. Password role qua psql `-v` parameterize, đọc từ secret manager — không hardcode trong git.
- **Anti-pattern**: chữa cháy bằng `GRANT SUPERUSER` cho service user (mở rộng quyền không cần thiết, vi phạm least privilege). Hoặc xoá file `005_pg_users.sql` (phá idempotency trên env đã apply — checksum mismatch).
- **Tags**: #migration-discipline #cluster-bootstrap #least-privilege #secret-hygiene #idempotency #prod-safety
```

---

## 4. Phase 4 artifact — verification queries

### 4.1 Tracker integrity

```sql
-- Trên cdc_cms_database prod
SELECT applied_by, COUNT(*)
  FROM cdc_system.schema_migrations
 GROUP BY applied_by
 ORDER BY 1;
-- Expected:
--   cluster-bootstrap | 3
--   runtime-migrator  | 50 (hoặc số migration L3 hiện có)

-- Trên cdc_dw dev
SELECT applied_by, COUNT(*)
  FROM cdc_system.schema_migrations
 GROUP BY applied_by;
-- Expected:
--   runtime-migrator | 53 (tất cả, không có cluster-bootstrap marker
--                         vì đã applied từ trước Phase 2)
```

### 4.2 Endpoint smoke

```bash
# CMS prod
curl -fsS http://cms-prod:8083/health           | jq .status
curl -fsS http://cms-prod:8083/ready            | jq .ready
curl -fsS http://cms-prod:8083/api/jobs/<known> | jq .status

# Worker prod
curl -fsS http://worker-prod:8090/health
curl -fsS http://worker-prod:8090/api/v1/internal/stats | jq .scheduler.last_tick_gap_seconds
# Expected: 60 ± 0.05
```

### 4.3 Worker scheduler gap

```sql
-- DB cdc_dw, tracking activity log như báo cáo 1121.
WITH samples AS (
  SELECT started_at,
         EXTRACT(EPOCH FROM (started_at - LAG(started_at) OVER (ORDER BY started_at))) AS gap_s
    FROM cdc_system.cdc_activity_log
   WHERE operation = 'transform'
     AND target_table = 'sd_export_jobs'
     AND started_at >= NOW() - INTERVAL '10 minutes'
)
SELECT MIN(gap_s), MAX(gap_s), AVG(gap_s), COUNT(*)
  FROM samples WHERE gap_s IS NOT NULL;
-- Expected: MIN/MAX ∈ [59.95, 60.05], AVG ~60.00, COUNT ≥ 9
```

---

## 5. Test matrix

| Scenario | Pre-state | Action | Expected |
|---|---|---|---|
| T1: Env mới (DB sạch) | tracker empty, user limited | Bootstrap → start CMS | Tracker có 3 cluster-bootstrap row + N runtime row; CMS healthy |
| T2: Env cũ đã apply | tracker có 53 row applied_by=runtime-migrator | Deploy new CMS | `applied_now=0`, applied_by NULL→DEFAULT migration column thành 'runtime-migrator' |
| T3: Env mid-state | tracker có 5 row, thiếu 005/039/042 | Bootstrap → start CMS | Bootstrap insert 3 row cluster-bootstrap; CMS chạy nốt 45 row còn lại |
| T4: Skip-list miss | thêm file 060 vào skip-list nhưng tracker chưa có | Start CMS | Log warn + tracker insert row 060 với marker; KHÔNG exec body |
| T5: Re-run bootstrap | đã chạy 1 lần | Chạy lại bootstrap script | ON CONFLICT DO NOTHING — không lỗi, không double-row |

---

## 6. Files inventory (sẽ tạo/sửa khi Muscle execute)

| Path | Operation |
|---|---|
| `cdc-cms-service/scripts/bootstrap_cms_db.sql` | NEW |
| `cdc-cms-service/scripts/bootstrap_cms_db.md` | NEW |
| `cdc-cms-service/internal/migrate/skip_list.go` | NEW |
| `cdc-cms-service/internal/migrate/runner.go` | EDIT (≤30 dòng) |
| `cdc-cms-service/internal/migrate/runner_test.go` | NEW (4 test) |
| `cdc-cms-service/migrations/054_tracker_applied_by.sql` | NEW |
| `cdc-cms-service/migrations/cluster_bootstrap/001_pg_roles.sql` | NEW |
| `cdc-cms-service/migrations/cluster_bootstrap/002_search_path.sql` | NEW |
| `cdc-cms-service/migrations/cluster_bootstrap/README.md` | NEW |
| `cdc-cms-service/migrations/005_pg_users.sql` | EDIT (chỉ thêm comment đầu file) |
| `cdc-cms-service/migrations/039_set_search_path.sql` | EDIT (chỉ thêm comment) |
| `cdc-cms-service/migrations/042_search_path_with_auth.sql` | EDIT (chỉ thêm comment) |
| `cdc-cms-service/migrations/embed.go` | EDIT (comment giải thích — optional) |
| `cdc-cms-service/docs/migrations.md` | NEW |
| `agent/memory/global/lessons.md` | APPEND lesson mới |
| `agent/memory/global/active_plans.md` | UPDATE bảng status |
| `agent/memory/workspaces/InvestigateTransformSchedulerStuck/05_progress.md` | APPEND mỗi step |

---

## 7. Sign-off gate (cho phase này)

Code change PASS được khi:
- Tất cả task `08_tasks_migration_refactor.md` ✅.
- Acceptance criteria `01_requirements_migration_refactor.md` §8 đạt 6/6.
- `/security-agent` review pass.
- Worker + CMS prod healthy verify thực tế (không suy diễn).
- Lesson + report + active_plans + progress đã APPEND.

Sau đó workspace `InvestigateTransformSchedulerStuck` có thể chuyển status sang ✅ Done.
