# Phase 2: Domain `schema`

## Mục tiêu
Tạo `internal/schema/` — quản lý mapping rules, DDL, pending fields.

---

## Bước 2.1: `internal/schema/model.go`

| Struct | Table |
|---|---|
| `MappingRuleV2` | `cdc_system.mapping_rule_v2` |
| `MappingRule` ⚠️ (V1 deprecated) | `cdc_system.cdc_mapping_rules` |
| `PendingField` | `cdc_system.pending_fields` |
| `SensitiveField` | `cdc_system.sensitive_fields` |

---

## Bước 2.2: `internal/schema/repository.go` (Port Interface — MỚI)

```go
package schema

type MappingRuleV2Repository interface {
    ListBySourceObject(ctx, sourceObjectID int64) ([]MappingRuleV2, error)
    ListActiveByMasterBinding(ctx, masterBindingID int64) ([]MappingRuleV2, error)
    ListActiveBySourceObject(ctx, sourceObjectID int64) ([]MappingRuleV2, error)
    ListActiveBySourceObjectAndBinding(ctx, sourceObjectID, shadowBindingID int64) ([]MappingRuleV2, error)
    Create(ctx, item *MappingRuleV2) error
    Update(ctx, item *MappingRuleV2) error
    GetActiveRulesBySourceTable(ctx, sourceTable string) ([]MappingRuleV2, error)
    ListGlobalSensitiveFields(ctx) ([]SensitiveField, error)
}

type PendingFieldRepository interface {
    GetByID(ctx, id uint) (*PendingField, error)
    GetByStatus(ctx, status string, sourceDB, tableName *string, page, pageSize int) ([]PendingField, int64, error)
    Update(ctx, pf *PendingField) error
    UpsertPendingField(ctx, tableName, sourceDB, fieldName, sampleValue, suggestedType string) error
    GetTableColumns(ctx, tableName string) (map[string]bool, error)
    GetTableColumnsInSchema(ctx, schemaName, tableName string) (map[string]bool, error)
}
```

---

## Bước 2.3: Move GORM Repos → `internal/schema/repository/`

| File cũ | File mới |
|---|---|
| `internal/repository/mapping_rule_v2_repo.go` | `internal/schema/repository/gorm_mapping_rule_v2_repo.go` |
| `internal/repository/mapping_rule_repo.go` | `internal/schema/repository/gorm_mapping_rule_repo.go` ⚠️ deprecated |
| `internal/repository/pending_field_repo.go` | `internal/schema/repository/gorm_pending_field_repo.go` |

---

## Bước 2.4: Move Services → `internal/schema/service/`

| File cũ | File mới |
|---|---|
| `internal/service/master_ddl_generator.go` | `internal/schema/service/master_ddl_generator.go` |
| `internal/service/schema_inspector.go` | `internal/schema/service/schema_inspector.go` |
| `internal/service/schema_validator.go` | `internal/schema/service/schema_validator.go` |
| `internal/service/type_resolver.go` | `internal/schema/service/type_resolver.go` |
| `internal/service/text_sanitizer.go` | `internal/schema/service/text_sanitizer.go` |
| `internal/service/transform_registry.go` | `internal/schema/service/transform_registry.go` |
| `internal/service/transmute/strategy.go` | `internal/schema/service/transmute/strategy.go` |
| `internal/service/transmute/copy_1_to_1.go` | `internal/schema/service/transmute/copy_1_to_1.go` |
| `internal/service/transmute/flatten.go` | `internal/schema/service/transmute/flatten.go` |

**Key functions trong `master_ddl_generator.go`** (14 funcs):

| Func | Hành động |
|---|---|
| `NewMasterDDLGenerator(systemDB, connMgr, mappingRepo, runtimeRepo, logger)` | Move, update import |
| `SetCacheInvalidator(fn)` | Move |
| `Generate(ctx, masterName)` | Move |
| `Apply(ctx, masterName)` | Move |
| `EnsureMaster(ctx, masterName)` | Move |
| `loadBinding(ctx, masterName)` | Move (private) |
| `parsePKFromSpec(spec)` | Move (private) |
| `parseIndexesFromSpec(spec)` | Move (private) |
| `markDDLStatus(ctx, masterBindingID, status, opErr)` | Move (private) |
| `ReconcileColumn(ctx, masterName, renameFrom, targetColumn, dataType)` | Move |
| `DropColumn(ctx, masterName, column)` | Move |
| `quoteDefaultValue(v, dataType)` | Move (private) |
| `quoteDDLIdent(v)` | Move (private) |
| `quoteDDLQualified(schemaName, tableName)` | Move (private) |

---

## Bước 2.5: Move Handlers → `internal/schema/handler/`

**Tách từ `command_handler.go`** → `internal/schema/handler/ddl_handler.go`:

| Func | Từ dòng | Move sang |
|---|---|---|
| `HandleStandardize(msg)` | L.360 | `schema/handler/ddl_handler.go` |
| `HandleCreateDefaultColumns(msg)` | L.688 | `schema/handler/ddl_handler.go` |
| `HandleAlterColumn(msg)` | L.2985 | `schema/handler/ddl_handler.go` |
| `HandleDropGINIndex(msg)` | L.2568 | `schema/handler/ddl_handler.go` |
| `processDiscoveryRows(ctx, registryID, shadowBindingID, sourceTable, rows, autoApprove)` | L.574 | `schema/handler/ddl_handler.go` (private) |
| `bridgeMappingRulesToV2(ctx, sourceID, sourceTable)` | L.1137 | `schema/handler/ddl_handler.go` (private) |
| `ensureCDCColumns(tableName)` | L.136 | `schema/handler/ddl_handler.go` (private) |
| `ensureCDCColumnsInSchema(schemaName, tableName)` | L.177 | `schema/handler/ddl_handler.go` (private) |
| `hasColumn(tableName, columnName)` | L.237 | `schema/handler/ddl_handler.go` (private) |
| `hasColumnInSchema(schemaName, tableName, columnName)` | L.241 | `schema/handler/ddl_handler.go` (private) |
| `tableExists(tableName)` | L.256 | `schema/handler/ddl_handler.go` (private) |
| `tableExistsInSchema(schemaName, tableName)` | L.260 | `schema/handler/ddl_handler.go` (private) |
| `listShadowColumns(schemaName, tableName)` | L.278 | `schema/handler/ddl_handler.go` (private) |
| `listShadowColumnsWithType(schemaName, tableName)` | L.299 | `schema/handler/ddl_handler.go` (private) |
| `normalizePGType(t)` | L.323 | `schema/handler/ddl_handler.go` (private) |
| `isSafeIdent(s)` | L.3140 | `schema/handler/ddl_handler.go` (private) |
| `isSafeType(t)` | L.3154 | `schema/handler/ddl_handler.go` (private) |
| `systemFieldSet()` | L.2648 | `schema/handler/ddl_handler.go` (private) |
| `normalizeMappingRuleDataType(dt)` | L.3323 | `schema/handler/ddl_handler.go` (private) |

**Move từ `master_ddl_handler.go`** → `internal/schema/handler/master_ddl_handler.go`:

| Func | Hành động |
|---|---|
| `NewMasterDDLHandler(gen, conn, logger)` | Move |
| `HandleMasterAlterColumn(msg)` | Move |
| `HandleMasterCreate(msg)` | Move |
| `reply(msg, resp)` | Move (private) |
| `replyErr(msg, correlationID, errMsg)` | Move (private) |

**Struct mới** `SchemaDDLHandler`:
```go
type SchemaDDLHandler struct {
    shadowDB         *gorm.DB
    metadataRegistry MetadataRegistry
    mappingRuleRepo  MappingRuleV2Repository
    pendingFieldRepo PendingFieldRepository
    natsConn         *nats.Conn
    logger           *zap.Logger
}
```

---

## Bước 2.6: Compile Check

```bash
go build ./internal/schema/...
go test ./internal/schema/...
```
