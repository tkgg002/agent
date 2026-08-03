# Implementation Plan: Auto-Migration for Recon Indexes (`096_optimize_recon_indexes.sql`)

## User Review Required
- Trình User duyệt việc tạo file migration tự động trong `cdc-cms-service`:
  - **File mới**: [096_optimize_recon_indexes.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/096_optimize_recon_indexes.sql)
  - **Cơ chế**: File sẽ tự động embed vào binary Go của `cdc-cms-service` via `//go:embed schema/*/*.sql` và tự động apply lên DB khi service khởi động.

## Proposed Changes

### 1. Database Migration: `cdc-cms-service`
#### [NEW] [096_optimize_recon_indexes.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/096_optimize_recon_indexes.sql)

```sql
-- Migration 096: Add performance indexes for recon latest queries (slow SQL optimization)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cdc_smoke_result_latest_distinct 
ON cdc_system.cdc_recon_smoke_result (
    COALESCE(shadow_schema, ''), 
    shadow_table, 
    COALESCE(NULLIF(master_schema, ''), ''), 
    COALESCE(NULLIF(master_table, ''), ''), 
    COALESCE(segment, 'source_shadow'), 
    checked_at DESC
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_shadow_binding_active_rn 
ON cdc_system.shadow_binding (shadow_schema, shadow_table, updated_at DESC, id DESC) 
WHERE is_active = TRUE;
```

## Verification Plan
### Automated Tests
- Run `go test ./test/...` in `cdc-cms-service` to ensure embedded migration files parse and compile cleanly.
