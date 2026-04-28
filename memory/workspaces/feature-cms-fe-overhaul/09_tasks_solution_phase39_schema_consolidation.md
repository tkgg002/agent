# Phase 39 — Solution Reference (drafts cho Muscle)

## T-39.8a — `centralized-data-service/migrations/040_admin_actions_in_cdc_system.sql`

```sql
-- Phase 39 — Move admin_actions từ public sang cdc_system.
-- Replaces cdc-cms-service/migrations/005_admin_actions.sql (orphan, không có runner).
-- Audit log cho destructive admin actions trên CDC stack. Partitioned by month on
-- created_at. Primary key bắt buộc include partition key → (created_at, id).

CREATE TABLE IF NOT EXISTS cdc_system.admin_actions (
    id              BIGSERIAL,
    user_id         TEXT        NOT NULL,
    action          TEXT        NOT NULL,
    target          TEXT,
    payload         JSONB,
    reason          TEXT        NOT NULL,
    result          TEXT,
    idempotency_key TEXT,
    ip_address      TEXT,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (created_at, id)
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS cdc_system.admin_actions_2026_04
    PARTITION OF cdc_system.admin_actions
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE IF NOT EXISTS cdc_system.admin_actions_2026_05
    PARTITION OF cdc_system.admin_actions
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS cdc_system.admin_actions_2026_06
    PARTITION OF cdc_system.admin_actions
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE IF NOT EXISTS cdc_system.admin_actions_default
    PARTITION OF cdc_system.admin_actions DEFAULT;

CREATE INDEX IF NOT EXISTS idx_admin_actions_user
    ON cdc_system.admin_actions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_actions_action
    ON cdc_system.admin_actions (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_actions_idem
    ON cdc_system.admin_actions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
```

## T-39.8b — `centralized-data-service/migrations/041_cdc_alerts_in_cdc_system.sql`

```sql
-- Phase 39 — Move cdc_alerts từ public sang cdc_system.
-- Replaces cdc-cms-service/migrations/013_alerts.sql (orphan).
-- State store cho observability alerts (system_health_collector → alert_manager).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS cdc_system.cdc_alerts (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint       TEXT        NOT NULL UNIQUE,
    name              TEXT        NOT NULL,
    severity          TEXT        NOT NULL,
    labels            JSONB,
    description       TEXT,
    status            TEXT        NOT NULL,
    fired_at          TIMESTAMPTZ NOT NULL,
    resolved_at       TIMESTAMPTZ,
    ack_by            TEXT,
    ack_at            TIMESTAMPTZ,
    silenced_by       TEXT,
    silenced_until    TIMESTAMPTZ,
    silence_reason    TEXT,
    occurrence_count  INT         NOT NULL DEFAULT 1,
    last_fired_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alerts_status
    ON cdc_system.cdc_alerts (status, fired_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_severity_firing
    ON cdc_system.cdc_alerts (severity, status)
    WHERE status = 'firing';
CREATE INDEX IF NOT EXISTS idx_alerts_resolved_at
    ON cdc_system.cdc_alerts (resolved_at DESC)
    WHERE resolved_at IS NOT NULL;
```

## T-39.8c — `centralized-data-service/migrations/042_search_path_with_auth.sql`

```sql
-- Phase 39 — Bao gồm cdc_auth_service vào search_path role.
-- Supersedes 039_set_search_path.sql.
-- Yêu cầu services restart connection pool sau khi apply để session pickup.

ALTER ROLE "user" SET search_path = cdc_system, cdc_auth_service, public;
```

## T-39.9 — REWRITE `cdc-auth-service/migrations/001_auth_users.sql`

```sql
-- Phase 39 — auth_users sống trong schema cdc_auth_service riêng.
-- Database goopay_dw shared với CDC system, nhưng schema tách bạch:
--   cdc_system        → CDC control plane
--   cdc_auth_service  → Auth service tables (chỉ cdc-auth-service đọc/ghi)
-- Bounded context: cdc-cms-service KHÔNG đọc trực tiếp bảng này, chỉ verify JWT
-- do cdc-auth-service ký.

BEGIN;

CREATE SCHEMA IF NOT EXISTS cdc_auth_service;

CREATE TABLE IF NOT EXISTS cdc_auth_service.auth_users (
    id          SERIAL PRIMARY KEY,
    username    VARCHAR(100) NOT NULL UNIQUE,
    email       VARCHAR(200) NOT NULL UNIQUE,
    password    VARCHAR(255) NOT NULL,
    full_name   VARCHAR(200),
    role        VARCHAR(20)  NOT NULL DEFAULT 'operator',
    is_active   BOOLEAN      DEFAULT TRUE,
    created_at  TIMESTAMP    DEFAULT NOW(),
    updated_at  TIMESTAMP    DEFAULT NOW(),

    CONSTRAINT au_check_role CHECK (role IN ('admin', 'operator'))
);

CREATE INDEX IF NOT EXISTS idx_auth_users_username
    ON cdc_auth_service.auth_users (username);
CREATE INDEX IF NOT EXISTS idx_auth_users_role
    ON cdc_auth_service.auth_users (role);

INSERT INTO cdc_auth_service.auth_users (username, email, password, full_name, role)
VALUES (
    'admin',
    'admin@goopay.vn',
    '$2a$10$0koc2s0krtdFu5L62ltWzOtnBk0b.DFbcgJHjLl4.jXntdhFUd60y', -- admin123
    'System Admin',
    'admin'
) ON CONFLICT (username) DO NOTHING;

COMMIT;
```

## T-39.10a — Patch `cdc-auth-service/internal/model/user.go:17`

```go
// BEFORE
func (User) TableName() string { return "auth_users" }

// AFTER
func (User) TableName() string { return "cdc_auth_service.auth_users" }
```

## T-39.10b — Patch `cdc-cms-service/internal/model/alert.go:37`

```go
// BEFORE
func (Alert) TableName() string { return "cdc_alerts" }

// AFTER
func (Alert) TableName() string { return "cdc_system.cdc_alerts" }
```

## T-39.10c — Patch `cdc-cms-service/internal/middleware/audit.go:166`

```go
// BEFORE
sb.WriteString("INSERT INTO admin_actions ")

// AFTER
sb.WriteString("INSERT INTO cdc_system.admin_actions ")
```

## T-39.11 — UPDATE `centralized-data-service/deployments/sql/wipe_cdc_runtime_v2.sql`

Replace section "drop public residue" thành nuke toàn bộ public CASCADE:

```sql
-- ============================================================================
-- 4. Nuke public schema entirely. Recreate empty (extensions tự pin schema).
-- ============================================================================
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO PUBLIC;
COMMENT ON SCHEMA public IS 'Phase 39 — kept empty by convention. All app tables live in cdc_system / cdc_auth_service / shadow_<src> / dw_<binding>.';

-- ============================================================================
-- 5. Drop dynamic per-source / per-binding schemas.
-- ============================================================================
DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT nspname FROM pg_namespace
           WHERE nspname LIKE 'shadow_%' OR nspname LIKE 'dw_%'
  LOOP
    EXECUTE 'DROP SCHEMA IF EXISTS ' || quote_ident(r.nspname) || ' CASCADE';
  END LOOP;
END $$;

-- ============================================================================
-- 6. Drop deprecated cdc_internal.
-- ============================================================================
DROP SCHEMA IF EXISTS cdc_internal CASCADE;

-- ============================================================================
-- 7. Truncate cdc_system tables (giữ DDL, xoá rows).
-- ============================================================================
DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT tablename FROM pg_tables WHERE schemaname='cdc_system'
  LOOP
    EXECUTE 'TRUNCATE TABLE cdc_system.' || quote_ident(r.tablename) || ' RESTART IDENTITY CASCADE';
  END LOOP;
END $$;
```

## T-39.18 — Verify pack (ngắn gọn)

```bash
# Schema-level
docker exec gpay-postgres psql -U user -d goopay_dw -c "\
  SELECT schemaname, count(*) AS n \
  FROM pg_tables \
  WHERE schemaname NOT LIKE 'pg_%' AND schemaname <> 'information_schema' \
  GROUP BY schemaname ORDER BY schemaname;"
# Expected:
#   cdc_auth_service | 1
#   cdc_system       | ≥25 (23 base + admin_actions parent + 4 child = 28)
#   public           | 0
#   shadow_*         | from worker after first ingest
#   dw_*             | from transmute after first run

# Auth round-trip
TOKEN=$(curl -s -X POST http://localhost:8081/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")
echo "TOKEN=${TOKEN:0:40}..."

# 11 endpoints (xem 02_plan Step 9 để lấy list đầy đủ)
```

## Notes cho Muscle

1. **Trật tự apply migration**: 040 → 041 → 042. Migration 042 ghi đè effect 039 — không cần xoá 039, chỉ append 042.
2. **Restart auth-service trước cms-service**: Auth GORM `TableName()` đọc schema mới ngay sau bind. Nếu cms-service start trước khi auth-service ready, login sẽ fail (chưa có JWT issuer).
3. **`go build` mỗi service**: Nếu fail, đọc lỗi → KHÔNG `--no-verify` skip. Fix root cause.
4. **Khi wipe + bootstrap xong, nếu Worker không tự tạo `shadow_<src>` schema**: Đó là behavior cũ của shadow_automator — separate debt (D-39.C). Không block Phase 39 DoD.
