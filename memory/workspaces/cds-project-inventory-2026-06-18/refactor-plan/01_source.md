# Phase 1: Domain `source`

## Mục tiêu
Tạo `internal/source/` chứa toàn bộ logic quản lý kết nối & đăng ký nguồn.

---

## Bước 1.1: Tạo `internal/source/model.go`

**Move từ** `internal/model/`:

| Struct | Table DB |
|---|---|
| `ConnectionRegistry` | `cdc_system.connection_registry` |
| `SourceObjectRegistry` | `cdc_system.source_object_registry` |
| `TableRegistry` (legacy ⚠️) | `cdc_system.cdc_table_registry` |
| `SchemaChangeLog` | `cdc_system.schema_changes_log` |

```go
// internal/source/model.go
package source

// ConnectionRegistry giữ nguyên toàn bộ fields + TableName()
// SourceObjectRegistry giữ nguyên toàn bộ fields + methods
// TableRegistry giữ nguyên toàn bộ fields + GetCandidates() + QualifiedTarget()
// SchemaChangeLog giữ nguyên fields
```

---

## Bước 1.2: Tạo `internal/source/repository.go` (Port Interface — MỚI)

```go
// internal/source/repository.go
package source

import "context"

type ConnectionRegistryRepository interface {
    GetAll(ctx context.Context) ([]ConnectionRegistry, error)
    GetByID(ctx context.Context, id uint) (*ConnectionRegistry, error)
    GetByCode(ctx context.Context, code string) (*ConnectionRegistry, error)
    ListActive(ctx context.Context) ([]ConnectionRegistry, error)
}

type SourceObjectRegistryRepository interface {
    GetByID(ctx context.Context, id int64) (*SourceObjectRegistry, error)
    ListByConnection(ctx context.Context, connectionID int64) ([]SourceObjectRegistry, error)
    ListActive(ctx context.Context) ([]SourceObjectRegistry, error)
    GetBySourceTable(ctx context.Context, dbName, tableName string) (*SourceObjectRegistry, error)
}

type TableRegistryRepository interface {
    GetAllActive(ctx context.Context) ([]TableRegistry, error)
    GetByID(ctx context.Context, id uint) (*TableRegistry, error)
    GetByTargetTable(ctx context.Context, targetTable string) (*TableRegistry, error)
    GetAll(ctx context.Context, filter RegistryFilter) ([]TableRegistry, int64, error)
    Create(ctx context.Context, entry *TableRegistry) error
    Update(ctx context.Context, entry *TableRegistry) error
    BulkCreate(ctx context.Context, entries []TableRegistry) (int, error)
    GetStats(ctx context.Context) (*RegistryStats, error)
}

type SchemaChangeLogRepository interface {
    Insert(ctx context.Context, log *SchemaChangeLog) error
    GetByTable(ctx context.Context, tableName string) ([]SchemaChangeLog, error)
    ListRecent(ctx context.Context, limit int) ([]SchemaChangeLog, error)
}
```

---

## Bước 1.3: Move GORM Repos vào `internal/source/repository/`

| File cũ | File mới |
|---|---|
| `internal/repository/connection_registry_repo.go` | `internal/source/repository/gorm_connection_repo.go` |
| `internal/repository/source_object_registry_repo.go` | `internal/source/repository/gorm_source_object_repo.go` |
| `internal/repository/registry_repo.go` | `internal/source/repository/gorm_registry_repo.go` |
| `internal/repository/schema_log_repo.go` | `internal/source/repository/gorm_schema_log_repo.go` |

**Functions trong `connection_registry_repo.go`** (7 funcs) — giữ nguyên, chỉ update import:
- `NewConnectionRegistryRepo(db)`
- `GetAll(ctx)`
- `GetByID(ctx, id)`
- `GetByCode(ctx, code)`
- `ListActive(ctx)` — *nếu chưa có, cần thêm*
- `Create(ctx, item)`
- `Update(ctx, item)`

**Functions trong `source_object_registry_repo.go`** (7 funcs) — giữ nguyên:
- `NewSourceObjectRegistryRepo(db)`
- `GetByID(ctx, id)`
- `ListByConnection(ctx, connectionID)`
- `ListActive(ctx)`
- `GetBySourceTable(ctx, dbName, tableName)`
- `Create(ctx, item)`
- `Update(ctx, item)`

**Functions trong `registry_repo.go`** (9 funcs) — giữ nguyên:
- `NewRegistryRepo(db)`
- `GetAllActive(ctx)`
- `GetByID(ctx, id)`
- `GetByTargetTable(ctx, targetTable)`
- `GetAll(ctx, filter)`
- `Create(ctx, entry)`
- `Update(ctx, entry)`
- `BulkCreate(ctx, entries)`
- `GetStats(ctx)`

**Functions trong `schema_log_repo.go`** (3 funcs) — giữ nguyên:
- `NewSchemaLogRepo(db)`
- `Insert(ctx, log)`
- `GetByTable(ctx, tableName)`

---

## Bước 1.4: Move Services vào `internal/source/service/`

| File cũ | File mới |
|---|---|
| `internal/service/metadata_registry_service.go` | `internal/source/service/metadata_registry.go` |
| `internal/service/registry_service.go` | `internal/source/service/registry_service.go` |
| `internal/service/connection_manager.go` | `internal/source/service/connection_manager.go` |
| `internal/service/connection_overrides.go` | `internal/source/service/connection_overrides.go` |
| `internal/service/connector_resolver.go` | `internal/source/service/connector_resolver.go` |
| `internal/service/source_router.go` | `internal/source/service/source_router.go` |

**Key functions trong `metadata_registry_service.go`** (34 funcs):

| Func | Giữ/Thay đổi |
|---|---|
| `NewMetadataRegistryService(...)` | Giữ, update import |
| `ReloadAll(ctx)` | Giữ |
| `GetTableConfig(targetTable)` | Giữ |
| `GetTableConfigByID(id)` | Giữ |
| `GetTableConfigBySource(sourceTable)` | Giữ |
| `ListTableConfigs()` | Giữ |
| `GetMappingRules(bindingID)` | Giữ |
| `GetDebeziumTables()` | Giữ |
| `ResolveSourceRoute(sourceDB, sourceTable)` | Giữ |
| `ResolveSourceRoutes(sourceDB, sourceTable)` | Giữ |
| `ResolveTargetRoute(targetTable)` | Giữ |
| `GetSourceDSN(ctx, connectionCode)` | Giữ |
| `GetMinFlushIntervalSeconds()` | Giữ |
| `GetMaskMap(bindingID)` | Giữ |
| `GetChildBindings(parentBindingID)` | Giữ |
| `resolveSourceURIFromConn(conn)` | Giữ (private) |
| `convertV2ToLegacyRule(v2, sourceObjectName)` | Giữ (private) |
| `synthesizeLegacyTableRegistry(src, binding, sourceURI)` | Giữ (private) |
| `tryPlainDSN(s)` | Giữ (private) |
| `tryEnvPointer(s)` | Giữ (private) |
| `buildDSNFromFields(conn)` | Giữ (private) |
| `buildSourceLookupKeys(src, connectionCode)` | Giữ (private) |
| `buildRouteLookupKeys(sourceDB, sourceTable)` | Giữ (private) |
| `extractLogicalCloneOf(raw)` | Giữ (private) |
| Test helpers: `NewMetadataRegistryServiceForTest`, etc. | Giữ |

---

## Bước 1.5: Move Handlers vào `internal/source/handler/`

**Tách từ `internal/handler/command_handler.go`** → `internal/source/handler/sync_handler.go`:

| Func trong command_handler.go | Move sang |
|---|---|
| `HandleSyncRegister(msg)` L.2818 | `source/handler/sync_handler.go` |
| `HandleSyncState(msg)` L.2877 | `source/handler/sync_handler.go` |
| `HandleRestartDebezium(msg)` L.2945 | `source/handler/sync_handler.go` |
| `verifyDebeziumConnector(ctx)` L.2864 | `source/handler/sync_handler.go` (private) |
| `detectConnectorName(entry)` L.3169 | `source/handler/sync_handler.go` (private) |
| `connectGET/POST/PUT/Call(...)` L.3177-3221 | `source/handler/sync_handler.go` (private) |

**Struct mới** `SyncHandler` thay thế việc dùng `CommandHandler` cho domain này:
```go
type SyncHandler struct {
    db               *gorm.DB
    natsConn         *nats.Conn
    metadataRegistry MetadataRegistry
    kafkaConnectURL  string
    logger           *zap.Logger
}
```

---

## Bước 1.6: Compile Check

```bash
go build ./internal/source/...
go test ./internal/source/...
```

**Dependency sau P1**: `worker_server.go` sẽ import từ `internal/source/` thay vì `internal/service/` + `internal/repository/` cho các components này.
