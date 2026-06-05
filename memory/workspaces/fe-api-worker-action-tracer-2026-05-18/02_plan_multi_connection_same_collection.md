# Plan — Multi connector cùng (db, collection) → tách shadow schema

**Phase**: multi_connection_same_collection
**Date**: 2026-05-19
**Status**: AWAITING USER DECISION — không implement trước khi user chọn Option

## 3 Options (so sánh trade-off)

### Option A — Identity-by-Connection (Core-aligned, RECOMMENDED)

Thêm `connection_id` (hoặc `connection_code`) vào identity tier `source_object_registry`. Đây là direction Core: 1 logical source = `(connection, db, object)` triplet, không phải `(db, object)` couple.

**Schema migration**:
1. **054_v1_add_connection_id.sql**: ADD COLUMN `cdc_table_registry.source_connection_id BIGINT REFERENCES cdc_system.connection_registry(id)`. Nullable cho backwards compat. Index `(source_connection_id, source_db, source_table)`.
2. **055_backfill_v1_connection_id.sql**: backfill từ first-wins lookup (giữ nguyên hành vi cũ cho rows hiện tại).
3. **056_relax_table_registry_unique_v2.sql** (chain sau 053): replace UNIQUE `(source_db, source_table, target_table)` thành `(source_connection_id, source_db, source_table, target_table)` để allow 2 connector cùng (db, table) → khác target.

**Go code change (4 file)**:

#### 1. `internal/model/table_registry.go` — thêm field
```go
type TableRegistry struct {
    ID                  uint   `gorm:"primaryKey" json:"id"`
    SourceConnectionID  *int64 `gorm:"column:source_connection_id" json:"source_connection_id,omitempty"`  // NEW
    SourceDB            string `gorm:"column:source_db;not null" json:"source_db"`
    // ... rest unchanged
}
```

#### 2. `internal/infra/persistence/source_object_v2_sync.go` — identity rebuild
```go
// BEFORE
normalizedSourceKey := strings.ToLower(fmt.Sprintf("%s:%s:%s", sourceEngine, sourceDB, sourceTable))
sourceConnectionID, err := s.resolveSourceConnectionID(ctx, tx, sourceEngine, sourceDB)
objectCode := buildSourceObjectCode(sourceEngine, sourceDB, sourceTable)
shadowSchema := normalizeShadowSchema(sourceDB)

// AFTER
sourceConnectionID, connectionCode, err := s.resolveSourceConnection(ctx, tx, entry, sourceEngine, sourceDB)
if err != nil { return err }
normalizedSourceKey := strings.ToLower(fmt.Sprintf("%s:%s:%s:%s",
    sourceEngine, connectionCode, sourceDB, sourceTable))
objectCode := buildSourceObjectCode(sourceEngine, connectionCode, sourceDB, sourceTable)
shadowSchema := normalizeShadowSchemaWithConnection(connectionCode, sourceDB)
```

```go
// resolveSourceConnection: priority order
//   (a) entry.SourceConnectionID nếu set (V1 đã có FK explicit từ FE) → SELECT connection_code by id.
//   (b) entry.SourceURL/connector hint nếu có → reverse lookup.
//   (c) Fallback first-wins (legacy backwards compat, log WARN audit trail).
func (s *SourceObjectV2SyncService) resolveSourceConnection(
    ctx context.Context, tx *gorm.DB, entry *model.TableRegistry,
    engine, sourceDB string,
) (id int64, code string, err error) {
    if entry.SourceConnectionID != nil && *entry.SourceConnectionID > 0 {
        var row struct {
            ID   int64  `gorm:"column:id"`
            Code string `gorm:"column:connection_code"`
        }
        e := tx.WithContext(ctx).Raw(`
            SELECT id, connection_code FROM cdc_system.connection_registry
            WHERE id = ? AND status = 'active'`, *entry.SourceConnectionID).Scan(&row).Error
        if e != nil { return 0, "", e }
        if row.ID == 0 {
            return 0, "", fmt.Errorf("connection_id=%d not active", *entry.SourceConnectionID)
        }
        return row.ID, row.Code, nil
    }
    // Backwards-compat fallback (legacy V1 rows pre-054 chưa có connection_id).
    return s.resolveFirstWinsConnection(ctx, tx, engine, sourceDB)
}
```

```go
func normalizeShadowSchemaWithConnection(connectionCode, sourceDB string) string {
    return naming.ShadowSchemaName(slugifyIdentifier(connectionCode) + "_" + slugifyIdentifier(sourceDB))
}

func buildSourceObjectCode(engine, connectionCode, sourceDB, sourceTable string) string {
    return "src_" + slugifyIdentifier(engine) + "_" +
        slugifyIdentifier(connectionCode) + "_" +
        slugifyIdentifier(sourceDB) + "_" +
        slugifyIdentifier(sourceTable)
}
```

#### 3. `internal/bootstrap/registry_mirror.go` — bootstrap mirror cũng dùng connection_id từ V1
Cùng pattern: priority `entry.SourceConnectionID` > first-wins. Update `normalized_source_key`, `object_code`, `shadow_schema` builds tương tự.

#### 4. `internal/api/registry_handler_register.go` (+ FE form) — thêm `source_connection_id` vào payload
```go
type RegisterPayload struct {
    SourceConnectionID int64  `json:"source_connection_id" binding:"required"`  // NEW
    SourceDB           string `json:"source_db"`
    // ... rest
}
```
FE: dropdown "Source Connector" populate từ `GET /api/v1/connections?role_type=source`.

#### 5. Worker `centralized-data-service` — cache keys
- `internal/service/metadata_registry_service.go:buildSourceLookupKeys` thêm variant include `connection_code` để lookup precision khi router cần.
- `targetCache`/`idCache` đã keyed by target/id (an toàn — không đụng).

**Side effects**:
- Existing data: 2 mongo connectors merged thành 1 source_object_registry row. Backfill cần manual review: pick `connection_id` đúng cho từng row hoặc default first-wins (legacy preservation).
- Worker `connectionOverrides` (phase trước) tiếp tục hoạt động — overlay vẫn keyed bởi `connection_code`, không đụng.

**Trade-off**:
- ✅ Semantic chuẩn — identity bao gồm connection ngay từ tier-1.
- ✅ User input rõ ràng — chọn connector cụ thể qua FE.
- ✅ Hỗ trợ multi-environment (local dev mongo vs prod cluster cùng db/collection).
- ❌ Scope to: schema migration + Go (4 file) + FE (Register form).
- ❌ Migration backfill có thể cần human review nếu legacy data ambiguous.

---

### Option B — Shadow-Schema-Only Composite (Pragmatic, partial fix)

Giữ `source_object_registry` identity by `(db, table)` (vẫn merge), nhưng tách `shadow_schema` theo connection. Mỗi connector tạo distinct shadow_binding với schema riêng.

**Schema migration**: KHÔNG đụng UNIQUE constraint nào.

**Go code change**: chỉ `shadow_binding` insertion path. `shadowSchema = "shadow_" + connectionCode + "_" + sourceDB`.

**Trade-off**:
- ✅ Scope nhỏ (1-2 file Go).
- ❌ Identity vẫn merge: `source_object_registry` chỉ 1 row cho 2 connector → metadata (sync_engine, profile_status, primary_key_field) shared giữa 2 connector. NẾU 2 connector physical cấu hình khác (e.g. different primary_key) → corruption.
- ❌ Resolver first-wins vẫn pick id=1 → connector id=2 (`goopay1`) source_connection_id metadata sai.
- ❌ KHÔNG đáp ứng user requirement "schemas postgres riêng" hoàn toàn — shadow_schema khác nhưng source_object metadata vẫn chung.

→ **Không recommend** trừ khi user accept compromise.

---

### Option C — V1-Level Identity Only (Lightweight, no V2 change)

Thêm `source_connection_id` vào V1 + relax V1 UNIQUE → V2 sync vẫn merge (như cũ). Có ích chỉ cho V1 readers.

**Trade-off**:
- ❌ V2 vẫn collision — bug user thấy KHÔNG fix.
- ❌ Half-measure → tech debt.

→ **Không recommend**.

---

## Recommendation: **Option A**

Lý do:
1. User report rõ "schemas postgres riêng" → cần tách identity tier-1.
2. Core direction (V2 model) already FK to `connection_registry` — completing identity contract.
3. Worker `connectionOverrides` (vừa làm) đã đứng trên `connection_code` — naming consistent.
4. FE Register form đã có structure để thêm dropdown (đã có ConnectionsList ở phase trước).

## Implementation Steps (sau khi user approve Option A)

| # | Step | File | Risk |
|---|---|---|---|
| 1 | Migration 054: ADD COLUMN `source_connection_id` vào V1 | `cdc-cms-service/migrations/schema/core/054_v1_add_source_connection_id.sql` | Low (nullable) |
| 2 | Migration 055: backfill từ first-wins | `055_backfill_v1_source_connection_id.sql` | Medium (cần human review nếu ambiguous) |
| 3 | Migration 056: relax UNIQUE → 4-cột `(source_connection_id, source_db, source_table, target_table)` | `056_relax_v1_unique_with_connection.sql` | Low (chỉ relax further) |
| 4 | Model `TableRegistry` thêm `SourceConnectionID *int64` | `internal/model/table_registry.go` | Low |
| 5 | V2 sync rebuild identity (`source_object_v2_sync.go`) | persistence | Medium (regenerate test) |
| 6 | Bootstrap mirror update (`registry_mirror.go`) | bootstrap | Low |
| 7 | API Register payload + validator | `internal/api/registry_handler_register.go` | Low |
| 8 | FE form: dropdown Source Connector | `cdc-cms-web` (out of scope here — user fix sau) | Medium |
| 9 | Worker `metadata_registry_service.go` cache key | `centralized-data-service` | Medium (regression test) |
| 10 | Workspace docs + report + global lesson | governance | Low |
| 11 | User: apply migrations + retry register `goopay1` | User | — |

## Verification gates

- [ ] `go build ./...` + `go vet ./...` EXIT=0 cả 2 service.
- [ ] `go test -count=1 ./internal/infra/persistence/... ./internal/api/...` PASS (CMS).
- [ ] `go test -count=1 ./internal/handler/... ./internal/service/...` PASS (worker).
- [ ] Manual: register `goopay1.centralized-export-service.export-jobs` → expected:
  - 2 source_object_registry rows (different connection_code).
  - 2 shadow_schema (`shadow_goopay_...` + `shadow_goopay1_...`) trong Postgres.
  - 2 shadow_binding under distinct source_object_id.

## Risk + Mitigation

| Risk | Mitigation |
|---|---|
| Backfill chọn sai connection_id cho legacy rows | Run dry-run query trước, log từng row affected; nếu ambiguous → set NULL + admin review qua CMS |
| Worker sourceCache lookup-by-source return wrong row | Cache key thêm connection_code variant; precision callers dùng `routeBySourceID` (id-based) |
| FE form chưa update kịp → API reject thiếu `source_connection_id` | Validator nullable cho backwards compat phase 1; warn log + first-wins fallback; FE update sau |
| Existing shadow tables (e.g. `shadow_centralized_export_service.sd_export_jobs`) phải re-key | Migration ALTER SCHEMA RENAME nếu cần rebrand; HOẶC giữ legacy + tạo schema mới song song |

## Câu hỏi cần user trả lời trước khi implement

1. **Chọn Option** — A (recommend), B, hay C?
2. **Backfill strategy** cho rows hiện tại (id=1 đang ambiguous giữa `goopay` và `goopay1`):
   - (a) First-wins → connection_id=1 (`goopay`), `goopay1` sẽ tạo row mới khi register lại.
   - (b) NULL → admin review qua CMS UI rồi assign.
   - (c) Delete + force re-register.
3. **FE update** trong scope phase này hay phase riêng?
4. **Shadow schema rename strategy**: giữ legacy `shadow_centralized_export_service` (cho id=1 row hiện có) + tạo `shadow_goopay1_centralized_export_service` cho id mới? Hay rename luôn cả 2 thành `shadow_<connection>_<db>` pattern?
