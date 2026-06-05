# 03_implementation_phase_p0 — Chi tiết kỹ thuật P0

> **REVIEW ROUND 1 (2026-06-01) NOTES** (apply trước khi Muscle execute):
> - Migration number `015` → **`068`** (C1: actual max migration hiện tại = 067).
> - Path prefix: tất cả `centralized-data-service/...` → **`data-hub/centralized-data-service/...`** (C2). Tương tự `cdc-cms-service/` → `data-hub/cdc-cms-service/`.
> - MaskingService signature: **DUAL-METHOD** giữ `MaskTableData()` legacy + thêm `MaskTableDataCtx(ctx, meta, table, data) (map, error)` (C4).
> - Recursive walker xử lý nested object (C5, ADR-009).
> - Bổ sung sub-task M-1b (audit log partition), M-2b (rule cache), M-4b (schema_inspector preview), M-5b (benchmark baseline).

## M-1 — Migration thêm mask_strategy + mask_options + audit tables

### File NEW: `data-hub/cdc-cms-service/migrations/schema/core/068_add_mask_strategy.sql`
> (Đổi số 015 → 068 theo C1. Sequencing seed UPDATE ở Stage 4 runbook theo ADR-010 — KHÔNG apply UPDATE cùng lúc DDL.)

```sql
-- =====================================================================
-- 015_mask_strategy.sql
-- Thêm masking strategy per-field + audit log để tuân thủ
-- Luật 91/2025/QH15 + Nghị định 356/2025 + VBHN 25/VBHN-NHNN.
-- =====================================================================

BEGIN;

-- 1. Enum strategy (DB-side để check constraint, đồng bộ Go enum)
CREATE TYPE cdc_system.mask_strategy AS ENUM (
    'NONE',
    'DROP',
    'HASH_HMAC',
    'PARTIAL'
);

-- 2. Mở rộng cdc_mapping_rules
ALTER TABLE cdc_system.cdc_mapping_rules
    ADD COLUMN mask_strategy cdc_system.mask_strategy
        NOT NULL DEFAULT 'NONE',
    ADD COLUMN mask_options JSONB
        NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN mask_key_version SMALLINT
        NOT NULL DEFAULT 1;

COMMENT ON COLUMN cdc_system.cdc_mapping_rules.mask_strategy IS
    'Strategy áp dụng cho field này khi sync sang shadow. Mặc định NONE.';
COMMENT ON COLUMN cdc_system.cdc_mapping_rules.mask_options IS
    'JSON options: PARTIAL={prefix:int,suffix:int,placeholder:string}; HASH_HMAC={key_ref:string}';
COMMENT ON COLUMN cdc_system.cdc_mapping_rules.mask_key_version IS
    'Phiên bản salt key, hỗ trợ rotation.';

-- 3. Audit log mỗi field bị mask (cho compliance trace)
-- M-1b: PARTITION BY RANGE (masked_at) → retention 13 tháng (H4)
CREATE TABLE cdc_system.mask_audit_log (
    id BIGSERIAL,
    event_id TEXT NOT NULL,
    source_code TEXT NOT NULL,
    table_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    strategy cdc_system.mask_strategy NOT NULL,
    key_version SMALLINT,
    masked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, masked_at)
) PARTITION BY RANGE (masked_at);

-- Partition 6 tháng đầu (rolling — Ops cron tạo tiếp).
CREATE TABLE cdc_system.mask_audit_log_2026_06 PARTITION OF cdc_system.mask_audit_log
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE cdc_system.mask_audit_log_2026_07 PARTITION OF cdc_system.mask_audit_log
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
-- ... 2026_08..2026_12 + 2027_01..2027_05 tương tự.

CREATE INDEX idx_mask_audit_log_masked_at
    ON cdc_system.mask_audit_log (masked_at DESC);
CREATE INDEX idx_mask_audit_log_table_field
    ON cdc_system.mask_audit_log (table_name, field_name);

-- Backfill loss log (ADR-013): record nào set null vì source expired.
CREATE TABLE cdc_system.mask_backfill_loss_log (
    id BIGSERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    row_pk TEXT NOT NULL,
    reason TEXT NOT NULL,  -- 'source_expired' | 'unknown_strategy' | ...
    lost_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 4. Audit log mỗi lần đổi config (CRUD)
CREATE TABLE cdc_system.mask_config_audit (
    id BIGSERIAL PRIMARY KEY,
    mapping_rule_id BIGINT NOT NULL
        REFERENCES cdc_system.cdc_mapping_rules(id) ON DELETE CASCADE,
    actor TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('CREATE','UPDATE','DELETE')),
    old_strategy cdc_system.mask_strategy,
    new_strategy cdc_system.mask_strategy,
    old_options JSONB,
    new_options JSONB,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ====================================================================
-- ⚠️  SEED UPDATE BELOW — APPLY Ở STAGE 4 (sau khi Worker code đã deploy)
-- Xem `12_rollout_runbook.md` ADR-010. Tách commit riêng để Ops control timing.
-- ====================================================================

-- 5. Seed default: field nhạy cảm hiện có → DROP (phá huỷ "***" path)
-- Lý do DROP thay vì HASH: password/OTP/PIN/CVV không có giá trị đối soát ở shadow.
UPDATE cdc_system.cdc_mapping_rules r
SET mask_strategy = 'DROP'
WHERE EXISTS (
    SELECT 1
    FROM cdc_system.cdc_table_registry t,
         jsonb_array_elements_text(t.sensitive_fields) AS sf(field)
    WHERE t.table_name = r.target_table
      AND lower(sf.field) = lower(r.target_column)
      AND lower(sf.field) IN ('password','pin','otp','cvv','secret','token')
);

-- Field đối soát được (card_number, cccd, account_number, phone, email) → HASH_HMAC
UPDATE cdc_system.cdc_mapping_rules r
SET mask_strategy = 'HASH_HMAC',
    mask_options = '{"key_ref": "masking.hmac.v1"}'::jsonb
WHERE EXISTS (
    SELECT 1
    FROM cdc_system.cdc_table_registry t,
         jsonb_array_elements_text(t.sensitive_fields) AS sf(field)
    WHERE t.table_name = r.target_table
      AND lower(sf.field) = lower(r.target_column)
      AND lower(sf.field) IN ('card_number','cccd','cmnd','account_number')
);

-- Field display (card last-4, phone last-3) → PARTIAL
UPDATE cdc_system.cdc_mapping_rules r
SET mask_strategy = 'PARTIAL',
    mask_options = '{"prefix": 0, "suffix": 4, "placeholder": "*"}'::jsonb
WHERE EXISTS (
    SELECT 1
    FROM cdc_system.cdc_table_registry t,
         jsonb_array_elements_text(t.sensitive_fields) AS sf(field)
    WHERE t.table_name = r.target_table
      AND lower(sf.field) = lower(r.target_column)
      AND lower(sf.field) IN ('phone','email')
);

COMMIT;
```

### File NEW kèm: `data-hub/cdc-cms-service/migrations/schema/core/068_add_mask_strategy.down.sql` (ADR-015)

```sql
BEGIN;
DROP TABLE IF EXISTS cdc_system.mask_backfill_loss_log;
DROP TABLE IF EXISTS cdc_system.mask_config_audit;
DROP TABLE IF EXISTS cdc_system.mask_audit_log CASCADE;
ALTER TABLE cdc_system.cdc_mapping_rules
    DROP COLUMN IF EXISTS mask_key_version,
    DROP COLUMN IF EXISTS mask_options,
    DROP COLUMN IF EXISTS mask_strategy;
DROP TYPE IF EXISTS cdc_system.mask_strategy;
COMMIT;
```

### File sửa: `data-hub/centralized-data-service/internal/model/mapping_rule.go`

```go
type MappingRule struct {
    // ... existing fields ...

    MaskStrategy    string          `gorm:"column:mask_strategy;type:cdc_system.mask_strategy;default:NONE" json:"mask_strategy"`
    MaskOptions     json.RawMessage `gorm:"column:mask_options;type:jsonb" json:"mask_options"`
    MaskKeyVersion  int16           `gorm:"column:mask_key_version" json:"mask_key_version"`
}
```

### Verify
- `psql -c "\d cdc_system.cdc_mapping_rules"` → thấy 3 column mới.
- `psql -c "SELECT mask_strategy, COUNT(*) FROM cdc_system.cdc_mapping_rules GROUP BY 1"` → distribution kỳ vọng.
- `psql -c "SELECT COUNT(*) FROM cdc_system.mask_audit_log"` → 0 (chưa có event).

---

## M-2 — Strategy interface + 4 implementation

### File NEW: `data-hub/centralized-data-service/internal/service/masking/strategy.go`

```go
package masking

import "context"

// Strategy là contract cho mọi masking strategy.
// Apply nhận giá trị raw + field metadata, trả về giá trị đã mask
// + boolean shouldDrop (true → caller set field = nil ở output).
type Strategy interface {
    Name() string
    Apply(ctx context.Context, in Input) (out Output, err error)
}

type Input struct {
    EventID   string
    Table     string
    Field     string
    RawValue  any
    Options   map[string]any
    KeyVersion int16
}

type Output struct {
    Value       any
    ShouldDrop  bool
    StrategyUsed string
}

// Registry map strategy name → Strategy.
type Registry struct {
    strategies map[string]Strategy
}

func NewRegistry() *Registry {
    return &Registry{strategies: make(map[string]Strategy)}
}

func (r *Registry) Register(s Strategy) { r.strategies[s.Name()] = s }

func (r *Registry) Resolve(name string) (Strategy, bool) {
    s, ok := r.strategies[name]
    return s, ok
}
```

### File NEW: `internal/service/masking/none.go`

```go
package masking

import "context"

type NoneStrategy struct{}

func (NoneStrategy) Name() string { return "NONE" }

func (NoneStrategy) Apply(_ context.Context, in Input) (Output, error) {
    return Output{Value: in.RawValue, StrategyUsed: "NONE"}, nil
}
```

### File NEW: `internal/service/masking/drop.go`

```go
package masking

import "context"

type DropStrategy struct{}

func (DropStrategy) Name() string { return "DROP" }

// DROP → caller set field = nil (JSON null) ở shadow.
func (DropStrategy) Apply(_ context.Context, _ Input) (Output, error) {
    return Output{Value: nil, ShouldDrop: true, StrategyUsed: "DROP"}, nil
}
```

### Bổ sung helper: `internal/service/masking/normalize.go` (H2)

```go
package masking

import (
    "fmt"
    "strconv"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "golang.org/x/text/unicode/norm"
)

// normalizeValue chuẩn hoá value bất kỳ thành string đại diện deterministic
// cho HMAC input. Tránh float scientific notation, BSON ObjectID format khác,
// Unicode normalization khác nhau giữa producer.
func normalizeValue(v any) string {
    if v == nil {
        return ""
    }
    switch x := v.(type) {
    case string:
        return norm.NFC.String(x)  // Unicode NFC chuẩn hoá (M-r7)
    case []byte:
        return string(x)
    case int, int8, int16, int32, int64:
        return fmt.Sprintf("%d", x)
    case uint, uint8, uint16, uint32, uint64:
        return fmt.Sprintf("%d", x)
    case float32:
        return strconv.FormatFloat(float64(x), 'f', -1, 32)
    case float64:
        return strconv.FormatFloat(x, 'f', -1, 64)
    case bool:
        return strconv.FormatBool(x)
    case primitive.ObjectID:
        return x.Hex()
    default:
        return fmt.Sprintf("%v", x)
    }
}
```

### File NEW: `internal/service/masking/hmac.go`

```go
package masking

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

type HmacStrategy struct {
    keyProvider KeyProvider // resolved từ vault, có cache + rotation
}

func NewHmacStrategy(kp KeyProvider) *HmacStrategy { return &HmacStrategy{keyProvider: kp} }

func (h *HmacStrategy) Name() string { return "HASH_HMAC" }

func (h *HmacStrategy) Apply(ctx context.Context, in Input) (Output, error) {
    if in.RawValue == nil {
        return Output{Value: nil, StrategyUsed: "HASH_HMAC"}, nil
    }
    s := normalizeValue(in.RawValue)  // H2: deterministic cho mọi type
    if s == "" {  // ADR-011: empty string → nil, không hash
        return Output{Value: nil, StrategyUsed: "HASH_HMAC"}, nil
    }
    key, err := h.keyProvider.Get(ctx, in.KeyVersion)
    if err != nil {
        return Output{}, fmt.Errorf("masking: key lookup: %w", err)
    }
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(s))
    return Output{
        Value:        hex.EncodeToString(mac.Sum(nil)),
        StrategyUsed: "HASH_HMAC",
    }, nil
}

// KeyProvider interface — implementation đọc từ vault hoặc env.
type KeyProvider interface {
    Get(ctx context.Context, version int16) ([]byte, error)
}
```

### File NEW: `internal/service/masking/partial.go`

```go
package masking

import (
    "context"
    "fmt"
    "strings"
)

type PartialStrategy struct{}

func (PartialStrategy) Name() string { return "PARTIAL" }

// Options: {"prefix": 0, "suffix": 4, "placeholder": "*"}
// Ví dụ "1234567890123456" prefix=0 suffix=4 → "************3456".
func (PartialStrategy) Apply(_ context.Context, in Input) (Output, error) {
    if in.RawValue == nil {
        return Output{Value: nil, StrategyUsed: "PARTIAL"}, nil
    }
    s := fmt.Sprintf("%v", in.RawValue)
    prefix := getInt(in.Options, "prefix", 0)
    suffix := getInt(in.Options, "suffix", 4)
    placeholder := getString(in.Options, "placeholder", "*")

    if len(s) <= prefix+suffix {
        return Output{Value: strings.Repeat(placeholder, len(s)), StrategyUsed: "PARTIAL"}, nil
    }
    masked := s[:prefix] + strings.Repeat(placeholder, len(s)-prefix-suffix) + s[len(s)-suffix:]
    return Output{Value: masked, StrategyUsed: "PARTIAL"}, nil
}

func getInt(m map[string]any, k string, def int) int {
    if v, ok := m[k]; ok {
        if n, ok := v.(float64); ok { return int(n) }
        if n, ok := v.(int); ok { return n }
    }
    return def
}
func getString(m map[string]any, k, def string) string {
    if v, ok := m[k]; ok { if s, ok := v.(string); ok { return s } }
    return def
}
```

### Verify
- `go test ./internal/service/masking/ -run TestStrategy -v` PASS với 4 strategy.

---

### File NEW: `internal/service/masking/rule_cache.go` (M-2b, H1 — hot-path cache)

```go
package masking

import (
    "context"
    "sync"
    "time"

    "go.uber.org/zap"
)

// RuleSnapshot là view immutable của mapping rule cho 1 field.
type RuleSnapshot struct {
    Strategy     string
    Options      map[string]any
    KeyVersion   int16
}

// RuleCache giữ rule per (table, field) trong sync.Map.
// Invalidation qua hook khi CMS PUT mask-config (pub-sub Kafka/Redis).
type RuleCache struct {
    inner    sync.Map         // key: "table|field" -> RuleSnapshot
    loader   RuleLoader       // fallback load từ DB nếu miss
    ttl      time.Duration    // soft TTL (e.g. 5 phút) tránh stale vô hạn
    lastLoad sync.Map         // key: "table|field" -> time.Time
    logger   *zap.Logger
}

type RuleLoader interface {
    Load(ctx context.Context, table string) (map[string]RuleSnapshot, error)
}

func NewRuleCache(loader RuleLoader, ttl time.Duration, logger *zap.Logger) *RuleCache {
    return &RuleCache{loader: loader, ttl: ttl, logger: logger}
}

// Get trả rule cho (table, field). Miss → load all rule của table, cache.
func (c *RuleCache) Get(ctx context.Context, table, field string) (RuleSnapshot, bool) {
    key := table + "|" + field
    if v, ok := c.inner.Load(key); ok {
        if t, ok := c.lastLoad.Load(key); ok && time.Since(t.(time.Time)) < c.ttl {
            return v.(RuleSnapshot), true
        }
    }
    rules, err := c.loader.Load(ctx, table)
    if err != nil {
        c.logger.Warn("rule cache load failed", zap.Error(err))
        if v, ok := c.inner.Load(key); ok { return v.(RuleSnapshot), true } // stale fallback
        return RuleSnapshot{Strategy: "NONE"}, false
    }
    now := time.Now()
    for f, r := range rules {
        c.inner.Store(table+"|"+f, r)
        c.lastLoad.Store(table+"|"+f, now)
    }
    if v, ok := c.inner.Load(key); ok { return v.(RuleSnapshot), true }
    return RuleSnapshot{Strategy: "NONE"}, false
}

// Invalidate gọi từ event Kafka/Redis khi CMS PUT mask-config.
func (c *RuleCache) Invalidate(table string) {
    c.inner.Range(func(k, _ any) bool {
        if key, ok := k.(string); ok && len(key) > len(table) && key[:len(table)+1] == table+"|" {
            c.inner.Delete(key)
            c.lastLoad.Delete(key)
        }
        return true
    })
}
```

---

## M-3 — HMAC key vault integration

### File NEW: `data-hub/centralized-data-service/pkgs/vault/key_loader.go`

```go
package vault

import (
    "context"
    "fmt"
    "os"
    "sync"
)

// KeyLoader cache + rotation. Env-based ban đầu (đơn giản), Vault sau.
type KeyLoader struct {
    mu     sync.RWMutex
    cache  map[int16][]byte
}

func NewKeyLoader() *KeyLoader { return &KeyLoader{cache: make(map[int16][]byte)} }

func (k *KeyLoader) Get(_ context.Context, version int16) ([]byte, error) {
    k.mu.RLock()
    if b, ok := k.cache[version]; ok {
        k.mu.RUnlock()
        return b, nil
    }
    k.mu.RUnlock()

    envKey := fmt.Sprintf("MASKING_HMAC_KEY_V%d", version)
    val := os.Getenv(envKey)
    if val == "" {
        return nil, fmt.Errorf("masking: missing env %s", envKey)
    }
    if len(val) < 32 {
        return nil, fmt.Errorf("masking: key %s too short (need ≥32 chars)", envKey)
    }
    b := []byte(val)

    k.mu.Lock()
    k.cache[version] = b
    k.mu.Unlock()
    return b, nil
}
```

### File sửa: `data-hub/centralized-data-service/internal/config/config.go`

```go
type MaskingConfig struct {
    Enabled       bool   `mapstructure:"enabled"`
    DefaultKeyVer int16  `mapstructure:"defaultKeyVersion"`
    AuditSampleRate float64 `mapstructure:"auditSampleRate"` // 0.0..1.0
}
```

### Deployment K8s Secret (template)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cdc-masking-keys
type: Opaque
stringData:
  MASKING_HMAC_KEY_V1: "<64-char-random-base64>"
```

### Verify
- `MASKING_HMAC_KEY_V1="$(openssl rand -hex 32)" go test ./pkgs/vault/...` PASS.
- Missing env → return error rõ ràng, không panic.

---

## M-4 — MaskingService refactor (bỏ "***" literal) — DUAL METHOD + RECURSIVE WALKER

### File sửa: `data-hub/centralized-data-service/internal/service/masking_service.go`

**Trước (hiện tại)**: hardcode `"***"` tại 5 nơi (lines 71, 77, 91, 133, 153) + `maskAnyRecursive` recursive map/array.

**Sau (refactor)**: DUAL-METHOD (C4) + RECURSIVE WALKER (C5, ADR-009).

```go
package service

import (
    "context"
    "fmt"

    "go.uber.org/zap"

    "data-hub/centralized-data-service/internal/service/masking"
)

type MaskMeta struct {
    EventID    string
    SourceCode string
}

type MaskingService struct {
    registry  *masking.Registry
    ruleCache *masking.RuleCache       // M-2b H1
    auditCh   chan masking.AuditRecord
    logger    *zap.Logger
}

func NewMaskingService(
    registry *masking.Registry,
    ruleCache *masking.RuleCache,
    auditCh chan masking.AuditRecord,
    logger *zap.Logger,
) *MaskingService {
    return &MaskingService{registry: registry, ruleCache: ruleCache, auditCh: auditCh, logger: logger}
}

// LEGACY signature — KHÔNG xoá để 22 caller không break (C4).
// Route nội bộ qua *Ctx với empty meta.
func (s *MaskingService) MaskTableData(table string, data map[string]any) map[string]any {
    out, _ := s.MaskTableDataCtx(context.Background(), MaskMeta{}, table, data)
    return out
}

// MỚI — full signature ctx + meta + error. Dùng cho caller được migrate.
func (s *MaskingService) MaskTableDataCtx(
    ctx context.Context, meta MaskMeta, table string, data map[string]any,
) (map[string]any, error) {
    if data == nil { return nil, nil }
    out, err := s.walk(ctx, meta, table, data)
    if err != nil { return nil, err }
    return out.(map[string]any), nil
}

// walk đệ quy vào map + array (ADR-009 — sửa regression nested object C5).
func (s *MaskingService) walk(ctx context.Context, meta MaskMeta, table string, value any) (any, error) {
    switch v := value.(type) {
    case map[string]any:
        out := make(map[string]any, len(v))
        for k, child := range v {
            rule, _ := s.ruleCache.Get(ctx, table, k)
            if rule.Strategy == "" || rule.Strategy == "NONE" {
                // Đệ quy vào nested map/array — giữ nguyên semantic cũ.
                processed, err := s.walk(ctx, meta, table, child)
                if err != nil { return nil, err }
                out[k] = processed
                continue
            }
            strat, ok := s.registry.Resolve(rule.Strategy)
            if !ok {
                s.logger.Warn("masking: unknown strategy, fallback DROP",
                    zap.String("strategy", rule.Strategy),
                    zap.String("field", k))
                out[k] = nil
                continue
            }
            res, err := strat.Apply(ctx, masking.Input{
                EventID: meta.EventID, Table: table, Field: k,
                RawValue: child, Options: rule.Options, KeyVersion: rule.KeyVersion,
            })
            if err != nil { return nil, err }
            if res.ShouldDrop {
                out[k] = nil
            } else {
                out[k] = res.Value
            }
            // Audit non-blocking.
            select {
            case s.auditCh <- masking.AuditRecord{
                EventID: meta.EventID, Source: meta.SourceCode, Table: table,
                Field: k, Strategy: res.StrategyUsed, KeyVersion: rule.KeyVersion,
            }:
            default:
            }
        }
        return out, nil
    case []any:
        out := make([]any, len(v))
        for i, item := range v {
            processed, err := s.walk(ctx, meta, table, item)
            if err != nil { return nil, err }
            out[i] = processed
        }
        return out, nil
    default:
        return v, nil
    }
}

// MaskJSONPayload + *Ctx (DUAL) — tương tự dual-method.
func (s *MaskingService) MaskJSONPayload(table string, data []byte) json.RawMessage {
    out, _ := s.MaskJSONPayloadCtx(context.Background(), MaskMeta{}, table, data)
    return out
}
func (s *MaskingService) MaskJSONPayloadCtx(
    ctx context.Context, meta MaskMeta, table string, data []byte,
) (json.RawMessage, error) {
    var parsed any
    if err := json.Unmarshal(data, &parsed); err != nil {
        return json.RawMessage(`null`), nil  // M-r9: invalid JSON → null, không "***"
    }
    walked, err := s.walk(ctx, meta, table, parsed)
    if err != nil { return nil, err }
    return json.Marshal(walked)
}
```

### M-4b — Schema inspector preview refactor (R-22)

`internal/service/schema_inspector.go:211` chuyển từ trả `"***"` sang metadata.

```go
// File sửa: schema_inspector.go
// Signature mới — return type+length info thay vì masked value literal.
type FieldSample struct {
    Type     string  `json:"type"`
    Length   int     `json:"length"`
    IsMasked bool    `json:"is_masked"`
    Strategy string  `json:"strategy,omitempty"`
    Sample   *string `json:"sample,omitempty"` // nil nếu IsMasked
}

func (si *SchemaInspector) PreviewField(ctx context.Context, table, field string, value any) FieldSample {
    rule, _ := si.ruleCache.Get(ctx, table, field)
    if rule.Strategy == "" || rule.Strategy == "NONE" {
        s := fmt.Sprintf("%v", value)
        return FieldSample{Type: typeOf(value), Length: len(s), IsMasked: false, Sample: &s}
    }
    return FieldSample{
        Type:     typeOf(value),
        Length:   len(fmt.Sprintf("%v", value)),
        IsMasked: true,
        Strategy: rule.Strategy,
    }
}
```

### File: `internal/service/text_sanitizer.go`
**KHÔNG sửa** — `text_sanitizer.go` dùng cho log/error free-form text (acceptable theo Điều luật, vì log không phải DB persistence + đã hash trace ID). Giữ nguyên.

### Verify
- `grep -rn '"\*\*\*"' internal/service/masking_service.go` → 0 match.
- `grep -rn '"\*\*\*"' internal/service/text_sanitizer.go` → vẫn còn (acceptable).
- Build PASS: `go build ./internal/service/...`.

---

## M-5 — Unit test masking_service_test.go

### File NEW: `internal/service/masking_service_test.go`

```go
package service_test

import (
    "context"
    "encoding/json"
    "os"
    "testing"

    "github.com/stretchr/testify/require"
    "go.uber.org/zap"

    "centralized-data-service/internal/service"
    "centralized-data-service/internal/service/masking"
    "centralized-data-service/pkgs/vault"
)

func setupRegistry(t *testing.T) *masking.Registry {
    t.Helper()
    os.Setenv("MASKING_HMAC_KEY_V1", "this_is_a_test_key_32_bytes_long!!")
    kp := vault.NewKeyLoader()
    r := masking.NewRegistry()
    r.Register(masking.NoneStrategy{})
    r.Register(masking.DropStrategy{})
    r.Register(masking.NewHmacStrategy(kp))
    r.Register(masking.PartialStrategy{})
    return r
}

func TestMaskingService_NoneStrategy(t *testing.T) {
    r := setupRegistry(t)
    repo := newMockRuleRepo(map[string]masking.RuleSnapshot{
        "trans_id": {MaskStrategy: "NONE"},
    })
    ch := make(chan masking.AuditRecord, 100)
    svc := service.NewMaskingService(r, repo, ch, zap.NewNop())

    out, err := svc.MaskTableData(context.Background(), "evt-1", "src", "transactions", map[string]any{
        "trans_id": "TX-001",
    })
    require.NoError(t, err)
    require.Equal(t, "TX-001", out["trans_id"])
}

func TestMaskingService_DropStrategy(t *testing.T) {
    r := setupRegistry(t)
    repo := newMockRuleRepo(map[string]masking.RuleSnapshot{
        "password": {MaskStrategy: "DROP"},
    })
    ch := make(chan masking.AuditRecord, 100)
    svc := service.NewMaskingService(r, repo, ch, zap.NewNop())

    out, err := svc.MaskTableData(context.Background(), "evt-2", "src", "users", map[string]any{
        "password": "secret123",
    })
    require.NoError(t, err)
    require.Nil(t, out["password"], "DROP phải set nil, không phải '***'")
    require.NotEqual(t, "***", out["password"], "tuyệt đối không còn '***' literal")
}

func TestMaskingService_HmacStrategy_Deterministic(t *testing.T) {
    r := setupRegistry(t)
    repo := newMockRuleRepo(map[string]masking.RuleSnapshot{
        "card_number": {MaskStrategy: "HASH_HMAC", MaskKeyVersion: 1},
    })
    ch := make(chan masking.AuditRecord, 100)
    svc := service.NewMaskingService(r, repo, ch, zap.NewNop())

    out1, _ := svc.MaskTableData(context.Background(), "evt-3", "src", "cards", map[string]any{"card_number": "4111111111111111"})
    out2, _ := svc.MaskTableData(context.Background(), "evt-4", "src", "cards", map[string]any{"card_number": "4111111111111111"})

    require.Equal(t, out1["card_number"], out2["card_number"], "HMAC phải deterministic")
    require.Len(t, out1["card_number"], 64, "SHA256 hex = 64 chars")
    require.NotContains(t, out1["card_number"].(string), "4111", "HMAC không leak prefix")
}

func TestMaskingService_PartialStrategy(t *testing.T) {
    r := setupRegistry(t)
    repo := newMockRuleRepo(map[string]masking.RuleSnapshot{
        "phone": {MaskStrategy: "PARTIAL", MaskOptions: json.RawMessage(`{"prefix":0,"suffix":3,"placeholder":"*"}`)},
    })
    ch := make(chan masking.AuditRecord, 100)
    svc := service.NewMaskingService(r, repo, ch, zap.NewNop())

    out, _ := svc.MaskTableData(context.Background(), "evt-5", "src", "users", map[string]any{
        "phone": "0901234567",
    })
    require.Equal(t, "*******567", out["phone"])
}

func TestMaskingService_AuditEmitted(t *testing.T) {
    r := setupRegistry(t)
    repo := newMockRuleRepo(map[string]masking.RuleSnapshot{
        "cccd": {MaskStrategy: "HASH_HMAC", MaskKeyVersion: 1},
    })
    ch := make(chan masking.AuditRecord, 10)
    svc := service.NewMaskingService(r, repo, ch, zap.NewNop())

    _, _ = svc.MaskTableData(context.Background(), "evt-6", "src", "users", map[string]any{"cccd": "001234567890"})

    select {
    case rec := <-ch:
        require.Equal(t, "cccd", rec.Field)
        require.Equal(t, "HASH_HMAC", rec.Strategy)
    default:
        t.Fatal("expected audit record")
    }
}
```

### Verify
- `go test ./internal/service -run TestMaskingService -v -cover` → coverage ≥ 90%.
- Test assert `NotEqual(t, "***", ...)` để khóa anti-pattern.

---

## M-5b — Benchmark baseline (M-r5)

### File NEW: `internal/service/masking_bench_test.go`

```go
package service_test

import (
    "context"
    "testing"
)

func BenchmarkMaskTableData_Baseline(b *testing.B) {
    svc := buildLegacyMaskingService(b) // version với "***" literal
    data := sampleEvent()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = svc.MaskTableData("transactions", data)
    }
}

func BenchmarkMaskTableData_NewStrategy(b *testing.B) {
    svc := buildNewMaskingService(b) // version với strategy engine
    data := sampleEvent()
    ctx := context.Background()
    meta := MaskMeta{EventID: "bench"}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = svc.MaskTableDataCtx(ctx, meta, "transactions", data)
    }
}
```

**Acceptance**: `New / Baseline` ≤ 1.10 (overhead ≤ 10%, NFR-1).

## Composite impact P0 (revised)
- Loại bỏ hoàn toàn `"***"` literal khỏi DB persistence path (production path; log path giữ).
- Strategy engine + recursive walker + rule cache + key normalization sẵn sàng.
- HMAC key versioning + audit log partitioned tuân thủ NĐ 356.
- Dual-method preserve backward compat → 22 caller không break.
- Schema inspector preview trả metadata thay vì literal `"***"`.
