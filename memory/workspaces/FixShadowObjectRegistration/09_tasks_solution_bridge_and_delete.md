# Technical Solution - Bridge Sources to Connection Registry

## Problem
The V2 Mapping sync logic (`SyncFromLegacyTx`) fails because it cannot find the source connection in `cdc_system.connection_registry`. This happens because the System Connector API only populates `cdc_system.sources`.

## Solution
Mirror the `cdc_system.sources` data into `cdc_system.connection_registry` within the repository layer. Also enhance the deletion logic to clean up both tables.

## Proposed Changes

### 1. Update `SystemConnectorRepo` [persistence]

#### [MODIFY] [system_connector_repo_gorm.go](file:///Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/infra/persistence/system_connector_repo_gorm.go)

- Add `splitHostPort` helper function.
- Update `Upsert` to perform a second query that mirrors to `connection_registry`.
- Update `MarkDeleted` to also update the corresponding `connection_registry` row to `retired`.

```go
func splitHostPort(addr string) (string, int) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", 0
	}
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return addr, 0
	}
	host := addr[:idx]
	port, _ := strconv.Atoi(addr[idx+1:])
	return host, port
}

func (r *systemConnectorRepoGorm) Upsert(ctx context.Context, s *model.Source) error {
    // 1. Existing Upsert to cdc_sources
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "connector_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source_type", "connector_class", "topic_prefix", "server_address",
			"database_include_list", "collection_include_list", "raw_config_sanitized",
			"status", "updated_at",
		}),
	}).Create(s).Error
    if err != nil {
        return err
    }

    // 2. Mirror to connection_registry
    host, port := splitHostPort(s.ServerAddress)
    return r.db.WithContext(ctx).Exec(`
		INSERT INTO cdc_system.connection_registry
			(connection_code, display_name, role_type, engine_type,
			 host, port, default_database, secret_ref, status)
		VALUES (?, ?, 'source', ?, ?, ?, ?, ?, 'active')
		ON CONFLICT (connection_code) DO UPDATE
		   SET engine_type      = EXCLUDED.engine_type,
		       host             = EXCLUDED.host,
		       port             = EXCLUDED.port,
		       default_database = EXCLUDED.default_database,
		       status           = 'active',
		       updated_at       = NOW()
	`,
		s.ConnectorName,
		s.ConnectorName,
		normalizeSourceEngine(s.SourceType),
		nullIfEmpty(host),
		nullIfZero(port),
		nullIfEmpty(s.DatabaseIncludeList),
		"v1:"+s.ConnectorName,
	).Error
}

func (r *systemConnectorRepoGorm) MarkDeleted(ctx context.Context, connectorName string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1. Delete from sources
        if err := tx.Model(&model.Source{}).
            Where("connector_name = ?", connectorName).
            Update("status", "deleted").Error; err != nil {
            return err
        }
        // 2. Retire from connection_registry
        return tx.Exec(`
            UPDATE cdc_system.connection_registry 
            SET status = 'retired', updated_at = NOW()
            WHERE connection_code = ?
        `, connectorName).Error
    })
}
```

### 2. Refactor `SourcesHandler` [api]

#### [MODIFY] [sources_handler.go](file:///Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/sources_handler.go)

- Inject `ports.SystemConnectorRepo` into `SourcesHandler`.
- Update `Create` to use `h.sourceRepo.Upsert`.

### 3. Add Delete Button to UI [frontend]

#### [MODIFY] [SourceConnectors.tsx](file:///Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/SourceConnectors.tsx)

- Add a `Delete` button to the `Actions` column in the `Connections` tab.
- This button will trigger the `deleteMut` using the `connector_name`.

```tsx
<Button
  size="small"
  danger
  icon={<DeleteOutlined />}
  onClick={() => setDeletePending(row.connector_name)}
>
  Delete
</Button>
```

## Verification Plan
1. Create a connector via CMS UI (calls `SystemConnectorsHandler.Create` -> `Repo.Upsert`).
2. Verify both tables have the data.
3. Register a source object via CMS UI (calls `RegistryHandler.Register` -> `SyncFromLegacy`).
4. Verify V2 tables are populated.
5. Delete the connector via CMS UI (calls `SystemConnectorsHandler.Delete` -> `Repo.MarkDeleted`).
6. Verify both tables are updated (status=deleted/retired).
