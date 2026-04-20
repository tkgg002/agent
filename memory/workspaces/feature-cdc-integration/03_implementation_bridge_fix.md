# Implementation: Bridge Fix

> Date: 2026-04-14
> Phase: bridge_fix

## Helper: ensureCDCColumns

File: `centralized-data-service/internal/handler/command_handler.go`

```go
func (h *CommandHandler) ensureCDCColumns(tableName string) error {
    // Check table exists
    var exists bool
    h.db.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = ? AND table_schema = 'public')", tableName).Scan(&exists)
    if !exists {
        return fmt.Errorf("table %s does not exist", tableName)
    }

    // Add CDC columns if missing
    cdcColumns := []struct{ name, def string }{
        {"_raw_data", "JSONB"},
        {"_source", "VARCHAR(20) DEFAULT 'airbyte'"},
        {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
        {"_version", "BIGINT DEFAULT 1"},
        {"_hash", "VARCHAR(64)"},
        {"_deleted", "BOOLEAN DEFAULT FALSE"},
        {"_created_at", "TIMESTAMP DEFAULT NOW()"},
        {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
    }
    for _, col := range cdcColumns {
        h.db.Exec(fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN IF NOT EXISTS %s %s`, tableName, col.name, col.def))
    }
    return nil
}
```

## Fix HandleAirbyteBridge

Before bridge SQL:
```go
if err := h.ensureCDCColumns(payload.TargetTable); err != nil {
    // Table doesn't exist yet → skip
    h.publishResult(msg, CommandResult{...Status: "skipped", Error: err.Error()})
    return
}
```

## Fix bridgeInPlace

Same — call `ensureCDCColumns` first.

## Fix HandleBatchTransform

```go
// Check _raw_data column exists
var hasRawData bool
h.db.Raw(`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = ? AND column_name = '_raw_data')`, targetTable).Scan(&hasRawData)
if !hasRawData {
    h.publishResult(msg, CommandResult{...Status: "skipped", Error: "table has no _raw_data column"})
    return
}
```

## Fix HandlePeriodicScan (HandleScanRawData)

Same check — skip tables without `_raw_data`.
