# 02_plan_phase_p3 — Smart Masking + Array Flatten (3 sub-phase)

> Status: PLAN (chưa execute).
> Trigger: User chốt 3 direction (2026-06-01):
> 1. Trigger sensitive-flag chỉ check **exact match** field name.
> 2. Bảng `sensitive_fields` thêm cột `mask_strategy` → keyword-level default.
> 3. Array nested (vd `orgs[]`) flatten thành **child shadow table** (Tier 2, không cắt).

---

## Tổng quan kiến trúc

```
┌──────────────────────────────────────────────────────────────────────┐
│ cdc_system.sensitive_fields                                          │
│ ─────────────────────────────────────                                │
│ id | field_name (EXACT, case-insensitive) | mask_strategy | created_at│
│                                            ('hmac'|'aes_gcm'|'none') │
└──────────────────────────────────────────────────────────────────────┘
                        │
                        │ trigger BEFORE INSERT/UPDATE
                        ▼
┌──────────────────────────────────────────────────────────────────────┐
│ cdc_system.mapping_rule_v2                                           │
│ ─────────────────────────────────────                                │
│ ... | is_sensitive_field (legacy, derived) | mask_strategy (per-rule)│
│     | explode_path (Tier 2) | child_shadow_binding_id                │
└──────────────────────────────────────────────────────────────────────┘
                        │
                        ▼
            DynamicMapper.MapData(rule, value)
                        │
              switch rule.MaskStrategy:
              ├─ 'none'    → plaintext
              ├─ 'hmac'    → HMAC-SHA256 hex (one-way, equality search)
              └─ 'aes_gcm' → AES-256-GCM base64 (reversible)

            Pipeline detect rule.ExplodePath != "":
              emit N rows vào shadow `child_shadow_binding.shadow_table`
              parent_id FK inject tự động
```

---

## Phase P3.1 — Trigger sensitive-flag exact match (1h)

### Requirement
- Trigger chỉ flag rule là sensitive **khi `source_field` HOẶC `target_column` BẰNG CHÍNH XÁC keyword** trong `sensitive_fields` (case-insensitive).
- Bỏ substring `LIKE '%...%'`.
- Backfill: re-compute toàn bộ rule cũ để xoá false-positive (`passwordHistory`, `passwordExpiredAt`, `resetPasswordTokenExpiredAt`, `lastUpdatedPassword`).

### Design
- 1 migration file `069_sensitive_exact_match.sql` (ĐÃ TẠO file vật lý ở entry 32, đúng direction).
- Trigger function thay `LIKE '%' || sf.field_name || '%'` → `= LOWER(sf.field_name)`.
- Backfill UPDATE re-compute.
- Không thêm cột, không thêm enum, không thay UI.

### File impact
| File | Action |
|---|---|
| `cdc-cms-service/migrations/schema/core/069_sensitive_exact_match.sql` | KEEP (đã đúng direction). |
| Nơi khác | KHÔNG đổi. |

### DoD
- Apply migration → trigger reload OK (`psql -c '\df+ cdc_system.fn_mapping_rule_v2_set_sensitive'`).
- Verify SQL: `SELECT source_field, is_sensitive_field FROM mapping_rule_v2 WHERE source_field IN ('passwordHistory','passwordExpiredAt','resetPasswordTokenExpiredAt','lastUpdatedPassword');` → tất cả FALSE.
- Verify SQL: rule có `source_field = 'password'` (exact) → TRUE.

---

## Phase P3.2 — Mask strategy enum (keyword-level default + per-rule override) (5-6h)

### Requirement
- `sensitive_fields` thêm cột `mask_strategy` (default `hmac`) → mỗi keyword có chiến lược mã hoá riêng.
- `mapping_rule_v2` thêm cột `mask_strategy` (default `hmac` khi `is_sensitive_field=TRUE`, `none` khi FALSE).
- Trigger v3: khi rule INSERT/UPDATE → match exact keyword → set `mapping_rule_v2.mask_strategy = sensitive_fields.mask_strategy` của keyword khớp.
- Per-rule override: user toggle Select trên `MappingFieldsPage` → ghi đè `mask_strategy` (KHÔNG bị trigger re-compute lại trên update khác).
- Pipeline mã hoá theo `mask_strategy`:
  - `none` → plaintext.
  - `hmac` → HMAC-SHA256 hex (KHÔNG decrypt được, equality search OK).
  - `aes_gcm` → AES-256-GCM base64 với version prefix `aesv1:` (decrypt được).
- Read pipeline có method `Decrypt` (chỉ AES-GCM). UI reveal cần plan riêng (defer).

### Design

#### Schema
```sql
-- migration 070_add_mask_strategy.sql
ALTER TABLE cdc_system.sensitive_fields
  ADD COLUMN mask_strategy VARCHAR(16) NOT NULL DEFAULT 'hmac'
    CHECK (mask_strategy IN ('none', 'hmac', 'aes_gcm'));

ALTER TABLE cdc_system.mapping_rule_v2
  ADD COLUMN mask_strategy VARCHAR(16) NOT NULL DEFAULT 'none'
    CHECK (mask_strategy IN ('none', 'hmac', 'aes_gcm'));

-- Seed: keyword phổ thông → strategy đúng nghĩa.
UPDATE cdc_system.sensitive_fields SET mask_strategy='hmac'    WHERE field_name IN ('password','token','secret','otp','pin');
UPDATE cdc_system.sensitive_fields SET mask_strategy='aes_gcm' WHERE field_name IN ('email','phone','address','card','account','ssn','balance');

-- Migrate cờ boolean → enum cho rule cũ:
UPDATE cdc_system.mapping_rule_v2 mr SET mask_strategy = COALESCE((
  SELECT sf.mask_strategy FROM cdc_system.sensitive_fields sf
  WHERE LOWER(sf.field_name) = LOWER(mr.source_field)
     OR LOWER(sf.field_name) = LOWER(mr.target_column)
  LIMIT 1
), 'hmac')
WHERE mr.is_sensitive_field = TRUE;
```

#### Trigger v3
```sql
CREATE OR REPLACE FUNCTION cdc_system.fn_mapping_rule_v2_set_sensitive()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
  v_strategy VARCHAR(16);
BEGIN
  -- Skip nếu user đã override (NEW.mask_strategy đã set khác trigger default).
  -- Pattern: chỉ trigger compute khi mask_strategy = NULL hoặc UPDATE không động vào cột mask_strategy.
  IF TG_OP = 'UPDATE' AND OLD.mask_strategy IS DISTINCT FROM NEW.mask_strategy THEN
    -- User chủ động đổi strategy → tôn trọng, không re-compute.
    NEW.is_sensitive_field := (NEW.mask_strategy <> 'none');
    RETURN NEW;
  END IF;

  SELECT sf.mask_strategy INTO v_strategy
  FROM cdc_system.sensitive_fields sf
  WHERE LOWER(sf.field_name) = LOWER(NEW.source_field)
     OR LOWER(sf.field_name) = LOWER(NEW.target_column)
  LIMIT 1;

  IF v_strategy IS NOT NULL THEN
    NEW.mask_strategy := v_strategy;
    NEW.is_sensitive_field := (v_strategy <> 'none');
  ELSE
    NEW.mask_strategy := 'none';
    NEW.is_sensitive_field := FALSE;
  END IF;
  RETURN NEW;
END;
$$;
```

#### Backend Go — cdc-cms-service
- `domain/mapping/rule.go`: thêm field `MaskStrategy string` (enum string).
- `infra/persistence/mapping_rule_repo_gorm.go`:
  - Row struct: thêm `MaskStrategy string gorm:"column:mask_strategy"`.
  - `baseSelect` SQL: thêm `mr.mask_strategy`.
  - `toDomain()`: map `MaskStrategy: r.MaskStrategy`.
- `api/dto/mapping_rule_dto.go`: `MappingRuleRow.MaskStrategy string json:"mask_strategy"` + map trong `RuleToRow`.
- Patch handler: validate enum `none|hmac|aes_gcm`. Khi user gửi `mask_strategy` mới → update + trigger sẽ tôn trọng.
- `internal/domain/sensitive/sensitive_field.go` (file existing): thêm `MaskStrategy string`.
- Repo + DTO + handler `sensitive-fields`: thêm field `mask_strategy` ở Create/Update.

#### Backend Go — centralized-data-service
- `internal/model/mapping_rule.go` + `mapping_rule_v2.go`: thêm `MaskStrategy string gorm:"column:mask_strategy;default:none"`.
- `internal/service/metadata_registry_service.go`:
  - `convertV2ToLegacyRule`: copy `MaskStrategy: v2.MaskStrategy`.
- `internal/service/masking_service.go`:
  - **Giữ nguyên** `hashValue` (HMAC) — đây là strategy `hmac`.
  - Thêm `encryptValue(v) string`:
    ```go
    func (ms *MaskingService) encryptValue(v interface{}) string {
        if v == nil { return "" }
        s := fmt.Sprintf("%v", v)
        if s == "" { return "" }
        if len(ms.aesKey) == 0 { /* warn once, fallback "***" */ }
        block, _ := aes.NewCipher(ms.aesKey)
        aead, _ := cipher.NewGCM(block)
        nonce := make([]byte, aead.NonceSize())
        io.ReadFull(rand.Reader, nonce)
        ct := aead.Seal(nonce, nonce, []byte(s), nil)
        return "aesv1:" + base64.StdEncoding.EncodeToString(ct)
    }
    ```
  - Thêm `DecryptValue(cipher string) (string, error)`:
    - Detect prefix `aesv1:` → strip + base64 decode → split nonce/ciphertext → GCM Open.
    - Không có prefix → return error "not encrypted".
  - Thêm public method `MaskByStrategy(strategy, v) string` để DynamicMapper gọi 1 entry-point.
  - Key derivation: `aesKey = sha256.Sum256(rawKey)` để chấp nhận key string bất kỳ độ dài (vẫn 256-bit entropy nếu key đủ random).
- `internal/service/dynamic_mapper.go`:
  - Rename `maybeHashColumn` → `maybeMaskColumn`.
  - Switch theo `rule.MaskStrategy`:
    ```go
    func (dm *DynamicMapper) maybeMaskColumn(rule model.MappingRule, value interface{}) interface{} {
        if dm.masking == nil { return value }
        switch rule.MaskStrategy {
        case "hmac":    return dm.masking.HashValue(value)
        case "aes_gcm": return dm.masking.EncryptValue(value)
        default:        return value
        }
    }
    ```
  - Backward compat: nếu `MaskStrategy == ""` và `IsSensitiveField == true` → fallback `hmac` (cho rule cũ chưa migrate).

#### Config
- `config/config.go`:
  - Thêm `MaskingAESKey string mapstructure:"maskingAesKey"`.
  - Env bind: `"maskingAesKey": {"CDS_MASKING_AES_KEY", "MASKING_AES_KEY"}`.
- `config-local.yml`/`config-sample.yml`/`config-production.yml`: thêm `maskingAesKey: ""` + comment 32-byte ≥ recommend.
- `worker_server.go`: gọi `maskingSvc.SetAESKey(cfg.MaskingAESKey)` sau `SetHMACKey`.

#### Frontend
- `src/pages/SensitiveFieldsPage.tsx`:
  - Form Add: thêm Select `mask_strategy` (default `hmac`) bên cạnh Input `field_name`.
  - Table: thêm cột "Strategy" hiển thị Tag màu (hmac=blue, aes_gcm=green, none=default).
  - Action thêm "Edit strategy" cho row existing.
  - Update interface `SensitiveField` thêm `mask_strategy: string`.
- `src/pages/MappingFieldsPage.tsx`:
  - Cột "Sensitive" Switch → đổi sang Select `[None / HMAC / AES-GCM]`.
  - Update interface `MappingRule` thêm `mask_strategy: string`.
  - Mutation patch rule: gửi `mask_strategy` thay vì `is_sensitive_field` boolean.

### File impact (Phase P3.2)
| Layer | File | Action |
|---|---|---|
| DB | `cdc-cms-service/migrations/schema/core/070_add_mask_strategy.sql` | NEW |
| Go (CMS) | `cdc-cms-service/internal/domain/mapping/rule.go` | EDIT — thêm field |
| Go (CMS) | `cdc-cms-service/internal/infra/persistence/mapping_rule_repo_gorm.go` | EDIT — SELECT + row + map |
| Go (CMS) | `cdc-cms-service/internal/api/dto/mapping_rule_dto.go` | EDIT — DTO field |
| Go (CMS) | `cdc-cms-service/internal/domain/sensitive/sensitive_field.go` | EDIT — thêm field |
| Go (CMS) | `cdc-cms-service/internal/api/dto/sensitive_field_dto.go` | EDIT |
| Go (CMS) | `cdc-cms-service/internal/api/sensitive_field_handler.go` | EDIT — validate enum |
| Go (CDS) | `centralized-data-service/internal/model/mapping_rule.go` | EDIT |
| Go (CDS) | `centralized-data-service/internal/model/mapping_rule_v2.go` | EDIT |
| Go (CDS) | `centralized-data-service/internal/service/metadata_registry_service.go` | EDIT — convertV2ToLegacy |
| Go (CDS) | `centralized-data-service/internal/service/masking_service.go` | EDIT — thêm encrypt/decrypt + MaskByStrategy |
| Go (CDS) | `centralized-data-service/internal/service/dynamic_mapper.go` | EDIT — maybeMaskColumn switch |
| Config | `centralized-data-service/config/config.go` | EDIT — MaskingAESKey |
| Config | `centralized-data-service/config/config-{local,sample,production}.yml` | EDIT |
| Config | `centralized-data-service/internal/server/worker_server.go` | EDIT — SetAESKey wire |
| FE | `cdc-cms-web/src/pages/SensitiveFieldsPage.tsx` | EDIT — Select strategy + Table column |
| FE | `cdc-cms-web/src/pages/MappingFieldsPage.tsx` | EDIT — Switch → Select |

### DoD Phase P3.2
- Migration 070 apply OK. Trigger v3 set `mask_strategy` đúng theo keyword.
- Build PASS: `go build ./...` (CMS + CDS), `npm run build` (FE).
- Test integration:
  - User Add keyword `email` với strategy `aes_gcm` → tạo rule mới có `source_field=email` → rule.mask_strategy = `aes_gcm` (trigger auto).
  - User Add keyword `password` với strategy `hmac` → rule mới `source_field=password` → rule.mask_strategy = `hmac`.
  - User PATCH rule.mask_strategy = `none` → trigger tôn trọng (không re-compute).
  - Snapshot 1 user record có `email='a@b.com'` + `password='hashedbcrypt'`:
    - shadow col `email` → `aesv1:<base64>` (decrypt được).
    - shadow col `password` → hex 64-char HMAC.
  - `MaskingService.DecryptValue('aesv1:...')` → restore `a@b.com`.

---

## Phase P3.3 — Child shadow table explode (8-12h)

### Requirement
- Array nested (vd `orgs[]` 1 cấp, `orgs[*].apps[*]` 2 cấp) flatten thành **child shadow table**.
- Mỗi element của array → 1 row trong shadow child.
- FK `parent_id` tự động inject (reference shadow parent PK).
- Pipeline snapshot + Kafka consumer đều phải hỗ trợ.

### Design

#### Schema mapping rule
```sql
-- migration 071_add_mapping_rule_explode.sql
ALTER TABLE cdc_system.mapping_rule_v2
  ADD COLUMN explode_path        TEXT,         -- e.g. '$.orgs[*]' (NULL = no explode)
  ADD COLUMN child_shadow_binding_id BIGINT
    REFERENCES cdc_system.shadow_binding(id);

-- Convention:
-- 1. Rule cha có explode_path != NULL → "explode marker" rule, KHÔNG tự map column.
--    Pipeline detect → loop array tại path → mỗi element gọi DynamicMapper.MapData
--    với rules của child_shadow_binding_id.
-- 2. Rule con (trong child_shadow_binding) có source_field tương đối (vd 'orgId').
--    Pipeline tự inject:
--      - `parent_id` = parent PK value (UUID/text)
--      - `_array_index` = vị trí trong array (0-based) — composite child PK
-- 3. Child shadow table convention: `<parent>__<field>` (vd `users__orgs`, `users__orgs__apps`).

CREATE INDEX idx_mr_v2_explode ON cdc_system.mapping_rule_v2 (binding_id)
  WHERE explode_path IS NOT NULL;
```

#### Shadow binding setup
- User tạo binding parent `users` (sources.users → shadow.users).
- User tạo binding child `users__orgs` (KHÔNG cần source riêng, parent's explode trigger).
- User tạo rule explode trên binding parent:
  - `source_field=orgs`, `target_column=NULL`, `explode_path=$.orgs[*]`, `child_shadow_binding_id=<id của users__orgs>`.
- User tạo rules thường trên child binding:
  - `source_field=orgId`, `target_column=org_id` (relative to exploded element).
  - `source_field=isPrimary`, `target_column=is_primary`.
  - `source_field=_id.$oid`, `target_column=_id` (child PK part).
  - Pipeline tự thêm `parent__id` + `_array_index` columns.

#### DynamicMapper changes
- New method `ExplodeAndMap(parentDoc, parentRule, childRules) []map[string]interface{}`:
  - Extract array tại `parentRule.ExplodePath` (dùng JSONPath).
  - Cho mỗi element + index → gọi `MapData(childRules, element)` → merge `parent__id` + `_array_index` → emit row.
- `MapData` chính: detect rule có `ExplodePath` → skip column emit, gọi `ExplodeAndMap` thay → return cả parent row + child rows.
- Return type mới: `MapResult { ParentRow map[string]any; ChildRows map[string][]map[string]any }` — key = child shadow table name.

#### Pipeline integration
- **Snapshot v2** (`internal/worker/snapshot_v2/`):
  - Sau khi MapData → batch insert parent row + child rows transaction.
  - Tracking progress: parent count + child count.
- **Kafka consumer** (`internal/worker/kafka_consumer/`):
  - INSERT event → emit parent + child.
  - UPDATE event → re-explode, replace-strategy: DELETE existing child by `parent__id` + bulk INSERT new (đơn giản, tránh diff phức tạp).
  - DELETE event → CASCADE delete child by `parent__id`.

#### Shadow child table provisioning
- `cdc-cms-service` migration runner đã có tự sinh shadow table từ binding. Cần mở rộng:
  - Detect binding là child (có parent reference qua rule explode) → tự thêm 2 cột:
    - `parent__id TEXT NOT NULL` + index.
    - `_array_index INT NOT NULL DEFAULT 0`.
  - PK composite: `(parent__id, _array_index)` thay vì auto PK.

#### Frontend
- `MappingFieldsPage.tsx`:
  - Form thêm 2 field cho rule explode: `explode_path` (Input) + `child_shadow_binding` (Select binding khác).
  - Table cột "Type": Tag `[EXPLODE → users__orgs]` cho rule có explode_path.
  - Click rule explode → navigate sang child binding mappings page.

### File impact (Phase P3.3)
| Layer | File | Action |
|---|---|---|
| DB | `cdc-cms-service/migrations/schema/core/071_add_mapping_rule_explode.sql` | NEW |
| Go (CMS) | `cdc-cms-service/internal/domain/mapping/rule.go` | EDIT — thêm `ExplodePath`, `ChildShadowBindingID` |
| Go (CMS) | `cdc-cms-service/internal/infra/persistence/mapping_rule_repo_gorm.go` | EDIT |
| Go (CMS) | `cdc-cms-service/internal/api/dto/mapping_rule_dto.go` | EDIT |
| Go (CMS) | `cdc-cms-service/internal/service/shadow_provisioner.go` (or equivalent) | EDIT — provision child cols |
| Go (CDS) | `centralized-data-service/internal/model/mapping_rule.go` + `v2.go` | EDIT |
| Go (CDS) | `centralized-data-service/internal/service/dynamic_mapper.go` | EDIT — ExplodeAndMap + JSONPath |
| Go (CDS) | `centralized-data-service/internal/service/dynamic_mapper_test.go` | NEW unit test |
| Go (CDS) | `centralized-data-service/internal/worker/snapshot_v2/processor.go` | EDIT — child batch insert |
| Go (CDS) | `centralized-data-service/internal/worker/kafka_consumer/handler.go` | EDIT — explode on event |
| Go (CDS) | `centralized-data-service/go.mod` | EDIT — add `github.com/PaesslerAG/jsonpath` |
| FE | `cdc-cms-web/src/pages/MappingFieldsPage.tsx` | EDIT — explode form + nav |

### DoD Phase P3.3
- Migration 071 apply OK.
- Build PASS (CMS + CDS + FE).
- Test e2e với `users` collection MongoDB:
  - Tạo binding `users` (parent) + `users__orgs` (child).
  - Rule explode `$.orgs[*]` → child binding.
  - Snapshot 1 user có 2 orgs → shadow `users` 1 row + shadow `users__orgs` 2 rows với `parent__id` + `_array_index` = 0, 1.
  - Update user (xoá 1 org, thêm 1 org) → child rows replace OK.
  - Delete user → child cascade.

### Defer / Out of scope
- Lồng cấp 2 (`orgs[*].apps[*]`): có thể làm sau bằng cách user tạo binding `users__orgs__apps` + rule explode trên binding `users__orgs`. **Recursive bằng config**, không cần code đặc biệt.
- UI generate auto từ sample MongoDB doc (đề xuất rule binding child) — defer.
- Read pipeline join parent + child trong 1 GET endpoint — defer.

---

## Thứ tự execute đề xuất
1. **P3.1 (1h)** — apply migration 069 đã có. Verify backfill data. Risk thấp.
2. **P3.2 (5-6h)** — schema + crypto + UI. Trigger v3 đảm bảo backward-compat. Risk trung bình.
3. **P3.3 (8-12h)** — pipeline explode. Risk cao, đụng vào snapshot + kafka consumer + migration runner. Cần test e2e kỹ.

Có thể parallel: P3.2 backend + FE (đụng nhau ít). P3.3 phải tuần tự sau P3.2 (vì DynamicMapper.maybeMaskColumn được dùng trong ExplodeAndMap loop).

---

## Risk register (bổ sung 11_risk_register)
| Risk | Mitigation |
|---|---|
| Trigger v3 mass-rewrite rule cũ → mất user override | Detect `OLD.mask_strategy IS DISTINCT FROM NEW.mask_strategy` để skip re-compute khi user PATCH chủ động. |
| AES-GCM key rotation | Version prefix `aesv1:` + lưu nhiều key version trong config, decrypt theo version. (Defer impl, document interface trong P3.2.) |
| Child shadow UPDATE strategy "delete + insert" gây spike write IO | Trong giai đoạn đầu chấp nhận. Optimize sang per-key upsert sau khi đo throughput. |
| `_array_index` thay đổi khi user xoá element giữa → child row "di chuyển" sai semantic | Document rõ: `_array_index` là **vị trí trong array hiện tại**, KHÔNG phải PK ổn định. Nếu cần PK ổn định → user phải map field nested có `_id` riêng (vd `_id.$oid`) làm composite key. |
