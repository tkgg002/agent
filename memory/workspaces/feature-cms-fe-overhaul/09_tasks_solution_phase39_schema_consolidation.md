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

## T-39.8d — REWRITE `centralized-data-service/migrations/028_sonyflake_fallback_fn.sql`

Architect Option A: function move từ `cdc_internal` → `cdc_system`; helper signature đổi sang `(p_schema TEXT, p_table TEXT)` để target schema động (caller truyền `shadow_<src>`).

```sql
-- Phase 39 — Sonyflake fallback ở cdc_system, helper schema-aware.
-- Replaces version cũ (cdc_internal.*). Function global, helper attach
-- trigger lên bảng ở schema bất kỳ caller chỉ định.

-- 1) Sequence dùng cho seq slot (16 bits) — tránh đụng nếu 028 cũ đã tạo
--    cdc_internal.fencing_token_seq, ta tạo bản mới ở cdc_system.
CREATE SEQUENCE IF NOT EXISTS cdc_system.fencing_token_seq;

-- 2) Sonyflake ID generator. Custom epoch 2026-01-01 UTC = 1767225600000ms.
CREATE OR REPLACE FUNCTION cdc_system.gen_sonyflake_id()
RETURNS BIGINT AS $fn$
DECLARE
  v_ts_ms BIGINT; v_machine INTEGER; v_seq BIGINT;
BEGIN
  v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT - 1767225600000;
  BEGIN
    v_machine := COALESCE(NULLIF(current_setting('cdc.machine_id', true), '')::INTEGER, 0) & 65535;
  EXCEPTION WHEN OTHERS THEN v_machine := 0; END;
  v_seq := nextval('cdc_system.fencing_token_seq') & 65535;
  RETURN ((v_ts_ms & 4398046511103) << 22) | ((v_machine::BIGINT & 65535) << 6) | (v_seq & 63);
END;
$fn$ LANGUAGE plpgsql VOLATILE;

-- 3) Trigger body — gen ID nếu NEW.id IS NULL/0.
CREATE OR REPLACE FUNCTION cdc_system.tg_sonyflake_fallback()
RETURNS TRIGGER AS $tg$
BEGIN
  IF NEW.id IS NULL OR NEW.id = 0 THEN
    NEW.id := cdc_system.gen_sonyflake_id();
  END IF;
  RETURN NEW;
END;
$tg$ LANGUAGE plpgsql;

-- 4) Helper attach trigger — schema-aware (2-arg).
--    Caller truyền p_schema = 'shadow_<src>', p_table = target_table.
CREATE OR REPLACE FUNCTION cdc_system.ensure_shadow_sonyflake_trigger(
    p_schema TEXT, p_table TEXT
) RETURNS VOID AS $h$
DECLARE v_trigger_name TEXT;
BEGIN
  v_trigger_name := 'trg_' || p_table || '_sonyflake_fallback';
  EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I.%I',
                 v_trigger_name, p_schema, p_table);
  EXECUTE format(
    'CREATE TRIGGER %I BEFORE INSERT ON %I.%I '
    || 'FOR EACH ROW EXECUTE FUNCTION cdc_system.tg_sonyflake_fallback()',
    v_trigger_name, p_schema, p_table);
END;
$h$ LANGUAGE plpgsql;

-- 5) Drop legacy single-arg helper từ cdc_internal nếu còn (idempotent).
DROP FUNCTION IF EXISTS cdc_internal.ensure_shadow_sonyflake_trigger(TEXT);
DROP FUNCTION IF EXISTS cdc_internal.tg_sonyflake_fallback();
DROP FUNCTION IF EXISTS cdc_internal.gen_sonyflake_id();
```

> Lưu ý: nếu 028 cũ vẫn còn trên đĩa và migration runner skip "Already applied" theo file path → file 028 vẫn nguyên byte-for-byte sẽ KHÔNG re-run. Cách an toàn: REWRITE inline + tăng minor version content; runner so sánh checksum sẽ re-apply. Nếu runner chỉ track tên file, Muscle phải truncate `cdc_system.schema_migrations WHERE version='028'` rồi chạy lại make migrate.

## T-39.8e — NEW `centralized-data-service/migrations/043_normalize_shadow_binding_schema.sql`

Phase 38 còn data legacy `cdc_system.shadow_binding.shadow_schema = 'cdc_internal'`. Sau khi wipe + bootstrap V2, bảng này TRUNCATE rồi nên migration 043 chỉ là defense-in-depth cho môi trường staging/prod sau này.

```sql
-- Phase 39 — Normalize shadow_binding.shadow_schema từ legacy 'cdc_internal'
-- sang 'shadow_<source_db>' chuẩn V2.
-- Lookup source_db qua source_object_registry FK (binding.source_object_id).

UPDATE cdc_system.shadow_binding sb
SET shadow_schema = 'shadow_' || lower(regexp_replace(sor.source_db, '[^a-zA-Z0-9_]', '_', 'g'))
FROM cdc_system.source_object_registry sor
WHERE sb.source_object_id = sor.id
  AND (sb.shadow_schema IS NULL OR sb.shadow_schema = 'cdc_internal' OR sb.shadow_schema = '');

-- Verify: 0 rows trỏ về cdc_internal sau update.
DO $$
DECLARE n INT;
BEGIN
  SELECT count(*) INTO n FROM cdc_system.shadow_binding
   WHERE shadow_schema = 'cdc_internal' OR shadow_schema IS NULL OR shadow_schema = '';
  IF n > 0 THEN
    RAISE EXCEPTION 'shadow_binding still has % unnormalized rows', n;
  END IF;
END $$;
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

## T-39.10d — REFACTOR `cdc-cms-service/internal/service/shadow_automator.go`

Architect Option A: signature `EnsureShadowTable` nhận thêm `shadowSchema string`. Bỏ block `CREATE SCHEMA cdc_internal`. Bỏ hẳn `ensureSonyflakeFunction` (function ở `cdc_system` qua migration 028 — không bootstrap inline). Helper trigger gọi 2-arg `cdc_system.ensure_shadow_sonyflake_trigger(p_schema, p_table)`.

```go
// File: internal/service/shadow_automator.go
package service

import (
	"context"
	"fmt"

	"cdc-cms-service/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ShadowAutomator struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewShadowAutomator(db *gorm.DB, logger *zap.Logger) *ShadowAutomator {
	return &ShadowAutomator{db: db, logger: logger}
}

// EnsureShadowTable creates <shadowSchema>.<target> + attaches sonyflake
// trigger via cdc_system.ensure_shadow_sonyflake_trigger. shadowSchema
// MUST be resolved by caller (registry_handler) — pragma:
// "shadow_" + normalizeIdent(reg.SourceDB).
func (s *ShadowAutomator) EnsureShadowTable(
	ctx context.Context, reg *model.TableRegistry, shadowSchema string,
) error {
	if err := validateIdent(reg.TargetTable); err != nil {
		return fmt.Errorf("invalid target_table: %w", err)
	}
	if err := validateIdent(shadowSchema); err != nil {
		return fmt.Errorf("invalid shadow_schema: %w", err)
	}
	if err := s.createShadowDDL(ctx, reg, shadowSchema); err != nil {
		return fmt.Errorf("create shadow ddl: %w", err)
	}
	if err := s.attachSonyflakeTrigger(ctx, shadowSchema, reg.TargetTable); err != nil {
		return fmt.Errorf("attach trigger: %w", err)
	}
	if err := s.markCreated(ctx, reg); err != nil {
		return fmt.Errorf("mark created: %w", err)
	}
	s.logger.Info("shadow table ensured",
		zap.String("schema", shadowSchema),
		zap.String("target", reg.TargetTable),
		zap.Uint("registry_id", reg.ID))
	return nil
}

// createShadowDDL — 8-col CDC layout, schema-aware. Caller bảo đảm
// schema đã tồn tại (bootstrap_cdc_system_v2_local.sql tạo shadow_<src>
// theo source_object). DDL không CREATE SCHEMA nữa để tránh leak schema
// nhầm namespace.
func (s *ShadowAutomator) createShadowDDL(
	ctx context.Context, reg *model.TableRegistry, schema string,
) error {
	target := reg.TargetTable
	ddl := fmt.Sprintf(`
        CREATE SCHEMA IF NOT EXISTS %[2]q;
        CREATE TABLE IF NOT EXISTS %[2]q.%[1]q (
            id BIGINT PRIMARY KEY,
            source_id VARCHAR(200) NOT NULL,
            _raw_data JSONB NOT NULL,
            _source VARCHAR(20) NOT NULL DEFAULT 'debezium',
            _synced_at TIMESTAMP NOT NULL DEFAULT NOW(),
            _version BIGINT NOT NULL DEFAULT 1,
            _hash VARCHAR(64),
            _deleted BOOLEAN DEFAULT FALSE,
            _created_at TIMESTAMP DEFAULT NOW(),
            _updated_at TIMESTAMP DEFAULT NOW(),
            CONSTRAINT %[3]q UNIQUE (source_id)
        );
        CREATE INDEX IF NOT EXISTS %[4]q ON %[2]q.%[1]q (_synced_at);
        CREATE INDEX IF NOT EXISTS %[5]q ON %[2]q.%[1]q (_source);
        CREATE INDEX IF NOT EXISTS %[6]q ON %[2]q.%[1]q USING GIN(_raw_data);
    `,
		target,
		schema,
		target+"_source_id_unique",
		"idx_"+target+"_synced_at",
		"idx_"+target+"_source",
		"idx_"+target+"_raw",
	)
	return s.db.WithContext(ctx).Exec(ddl).Error
}

func (s *ShadowAutomator) attachSonyflakeTrigger(
	ctx context.Context, schema, table string,
) error {
	return s.db.WithContext(ctx).Exec(
		"SELECT cdc_system.ensure_shadow_sonyflake_trigger(?, ?)", schema, table,
	).Error
}

func (s *ShadowAutomator) markCreated(ctx context.Context, reg *model.TableRegistry) error {
	return s.db.WithContext(ctx).Model(&model.TableRegistry{}).
		Where("id = ?", reg.ID).
		Update("is_table_created", true).Error
}

func validateIdent(s string) error {
	if len(s) == 0 || len(s) > 63 {
		return fmt.Errorf("identifier length")
	}
	for _, c := range s {
		if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return fmt.Errorf("identifier char: %q", c)
		}
	}
	return nil
}
```

Mất 3 hàm legacy: `ensureSonyflakeFunction` (xóa), bản single-arg attach (xóa), block CREATE SCHEMA cdc_internal trong DDL (xóa).

## T-39.10e — Patch caller `cdc-cms-service/internal/api/registry_handler.go:113`

Resolve `shadowSchema` từ `reg.SourceDB` trước khi gọi automator. Pragma normalize: lowercase + `[^a-zA-Z0-9_]` → `_`.

```go
// BEFORE (line 112-120):
if h.automator != nil {
    if err := h.automator.EnsureShadowTable(c.Context(), &entry); err != nil {
        if delErr := h.db.Delete(&model.TableRegistry{}, entry.ID).Error; delErr != nil {
            h.logger.Error("registry rollback failed after shadow err",
                zap.Uint("id", entry.ID), zap.Error(delErr))
        }
        return c.Status(500).JSON(fiber.Map{"error": "shadow DDL failed: " + err.Error()})
    }
}

// AFTER:
if h.automator != nil {
    shadowSchema := "shadow_" + normalizeShadowIdent(entry.SourceDB)
    if err := h.automator.EnsureShadowTable(c.Context(), &entry, shadowSchema); err != nil {
        if delErr := h.db.Delete(&model.TableRegistry{}, entry.ID).Error; delErr != nil {
            h.logger.Error("registry rollback failed after shadow err",
                zap.Uint("id", entry.ID), zap.Error(delErr))
        }
        return c.Status(500).JSON(fiber.Map{"error": "shadow DDL failed: " + err.Error()})
    }
}
```

Helper (đặt cuối file `registry_handler.go` hoặc trong `internal/util/ident.go` mới):

```go
// normalizeShadowIdent — lower-snake-case theo convention shadow_<src>.
// Postgres identifier safe: chỉ giữ a-z 0-9 _, mọi ký tự khác → _.
func normalizeShadowIdent(s string) string {
    out := make([]byte, 0, len(s))
    for i := 0; i < len(s); i++ {
        c := s[i]
        switch {
        case c >= 'A' && c <= 'Z':
            out = append(out, c+32)
        case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
            out = append(out, c)
        default:
            out = append(out, '_')
        }
    }
    return string(out)
}
```

## T-39.10f — Patch `cdc-cms-service/internal/api/mapping_preview_handler.go:47`

Trước khi SELECT shadow rows, lookup `shadow_schema` từ `cdc_system.shadow_binding`. Miss → 404 `binding_not_found`.

```go
// BEFORE (line 41-52):
// Fetch shadow sample rows — _raw_data + _gpay_source_id.
var rows []struct {
    GpayID   int64  `gorm:"column:_gpay_id"`
    SourceID string `gorm:"column:_gpay_source_id"`
    RawData  []byte `gorm:"column:_raw_data"`
}
q := `SELECT _gpay_id, _gpay_source_id, _raw_data FROM cdc_internal.` + `"` + req.ShadowTable + `"` + ` ORDER BY _synced_at DESC LIMIT ?`
if err := h.db.WithContext(c.Context()).Raw(q, limit).Scan(&rows).Error; err != nil {
    h.logger.Error("preview: shadow read failed",
        zap.String("table", req.ShadowTable), zap.Error(err))
    return c.Status(500).JSON(fiber.Map{"error": "shadow_read_failed", "detail": err.Error()})
}

// AFTER:
// Resolve shadow_schema từ binding (Phase 39 — schema-aware).
var shadowSchema string
if err := h.db.WithContext(c.Context()).Raw(
    `SELECT shadow_schema FROM cdc_system.shadow_binding
      WHERE shadow_table = ? AND is_active = true LIMIT 1`,
    req.ShadowTable,
).Scan(&shadowSchema).Error; err != nil {
    h.logger.Error("preview: binding lookup failed",
        zap.String("table", req.ShadowTable), zap.Error(err))
    return c.Status(500).JSON(fiber.Map{"error": "binding_lookup_failed"})
}
if shadowSchema == "" {
    return c.Status(404).JSON(fiber.Map{"error": "binding_not_found", "shadow_table": req.ShadowTable})
}
if !propIdentRe.MatchString(shadowSchema) {
    return c.Status(400).JSON(fiber.Map{"error": "invalid_shadow_schema"})
}

// Fetch shadow sample rows — _raw_data + _gpay_source_id.
var rows []struct {
    GpayID   int64  `gorm:"column:_gpay_id"`
    SourceID string `gorm:"column:_gpay_source_id"`
    RawData  []byte `gorm:"column:_raw_data"`
}
q := `SELECT _gpay_id, _gpay_source_id, _raw_data FROM "` + shadowSchema + `"."` + req.ShadowTable + `" ORDER BY _synced_at DESC LIMIT ?`
if err := h.db.WithContext(c.Context()).Raw(q, limit).Scan(&rows).Error; err != nil {
    h.logger.Error("preview: shadow read failed",
        zap.String("schema", shadowSchema),
        zap.String("table", req.ShadowTable), zap.Error(err))
    return c.Status(500).JSON(fiber.Map{"error": "shadow_read_failed", "detail": err.Error()})
}
```

## T-39.10g — Patch `cdc-cms-service/internal/api/schema_proposal_handler.go:131-138`

Trong block `case "shadow"`, lookup shadow_schema theo `row.TableName` rồi substitute schema động vào ALTER TABLE.

```go
// BEFORE (line 131-138):
case "shadow":
    stmt := fmt.Sprintf(
        `ALTER TABLE cdc_internal.%q ADD COLUMN IF NOT EXISTS %q %s`,
        row.TableName, row.ColumnName, finalType,
    )
    if err := tx.Exec(stmt).Error; err != nil {
        return fmt.Errorf("alter shadow: %w", err)
    }

// AFTER:
case "shadow":
    var shadowSchema string
    if err := tx.Raw(
        `SELECT shadow_schema FROM cdc_system.shadow_binding
          WHERE shadow_table = ? AND is_active = true LIMIT 1`,
        row.TableName,
    ).Scan(&shadowSchema).Error; err != nil {
        return fmt.Errorf("binding lookup: %w", err)
    }
    if shadowSchema == "" {
        return fmt.Errorf("binding_not_found for shadow table %q", row.TableName)
    }
    if !propIdentRe.MatchString(shadowSchema) {
        return fmt.Errorf("invalid_shadow_schema: %q", shadowSchema)
    }
    stmt := fmt.Sprintf(
        `ALTER TABLE %q.%q ADD COLUMN IF NOT EXISTS %q %s`,
        shadowSchema, row.TableName, row.ColumnName, finalType,
    )
    if err := tx.Exec(stmt).Error; err != nil {
        return fmt.Errorf("alter shadow: %w", err)
    }
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

1. **Trật tự apply migration**: 028 (rewrite) → 040 → 041 → 042 → 043. Migration 042 ghi đè effect 039 — giữ 039 làm history.
2. **Restart auth-service trước cms-service**: Auth GORM `TableName()` đọc schema mới ngay sau bind. Nếu cms-service start trước khi auth-service ready, login sẽ fail (chưa có JWT issuer).
3. **`go build` mỗi service**: Nếu fail, đọc lỗi → KHÔNG `--no-verify` skip. Fix root cause.
4. **shadow_<src> schema tạo bởi đâu**: Sau Phase 39, `shadow_automator.createShadowDDL` có `CREATE SCHEMA IF NOT EXISTS <shadowSchema>` dưới namespace động — không hardcode `cdc_internal`. `bootstrap_cdc_system_v2_local.sql` cũng tạo trước theo `source_object_registry` rows. Defense in depth.
5. **Final grep audit**: Sau khi patch xong 4 file Go, grep `cdc_internal` trong toàn repo (loại trừ migration history 018-038 là lịch sử migration cũ): expected 0 lines runtime code. Nếu còn → re-grep + patch trước khi exec wipe.
6. **DoD B 2-step verify**:
   - B1 (ngay sau migrate): `cdc_internal` schema = 0
   - B2 (sau khi register 1 source object qua Wizard + worker chạy ≥5 phút): `cdc_internal` vẫn = 0. Nếu B2 > 0 → có code path runtime nào đó còn hardcode → re-grep, fix, lặp.
