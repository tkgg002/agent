# Hồ sơ Giải pháp kỹ thuật - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

## 1. File cần thay đổi
- `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`

## 2. Chi tiết mã nguồn cần thay đổi

### Thay đổi 1: Cập nhật biến `listLatestPrimary`
Sửa lại phần SELECT đầu tiên trong UNION của `listLatestPrimary` để sử dụng `LEFT JOIN LATERAL` với `cdc_recon_smoke_result`:

```go
const listLatestPrimary = `
	SELECT r.id,
	       r.run_id,
	       r.cycle_id,
	       r.segment,
	       r.source_type,
	       r.source_host,
	       r.source_db,
	       r.source_total,
	       r.source_active,
	       r.shadow_total,
	       r.shadow_active,
	       r.master_schema,
	       r.master_table,
	       r.master_total,
	       r.master_active,
	       r.diff,
	       r.status,
	       r.error_message,
	       r.duration_ms,
	       r.checked_at,
	       -- Tương thích ngược với ReconciliationReport
	       CASE WHEN r.segment = 'shadow_master' THEN r.master_table ELSE r.shadow_table END AS target_table,
	       r.source_count,
	       r.dest_count,
	       r.source_count AS nullable_source_count,
	       NULL AS error_code,
	       -- Enrichment metadata
	       reg.sync_engine, reg.timestamp_field,
	       reg.timestamp_field_source, reg.timestamp_field_confidence,
	       reg.full_source_count, reg.full_dest_count, reg.full_count_at,
	       lag.ingest_lag_ms, lag.transmute_lag_ms, lag.worker_backlog,
	       sb.source_object_id,
	       COALESCE(r.source_table, so.source_object_name) AS source_table,
	       COALESCE(r.shadow_schema, sb.shadow_schema) AS shadow_schema,
	       COALESCE(r.shadow_table, sb.shadow_table) AS shadow_table,
	       cr.connection_code AS source_connection_code,
	       r.master_schema,
	       COALESCE(scope_counts.binding_count, 0) > 1 AS scope_ambiguous
	  FROM (
		SELECT DISTINCT ON (shadow_schema, shadow_table, master_schema, master_table, segment) *
		  FROM (
			SELECT
				r.id,
				r.run_id,
				NULL::bigint AS cycle_id,
				r.segment,
				NULL::text AS source_type,
				NULL::text AS source_host,
				r.source_db,
				COALESCE(s.source_total, r.total_source_count) AS source_total,
				COALESCE(s.source_active, r.source_count) AS source_active,
				COALESCE(s.shadow_total, r.total_dest_count) AS shadow_total,
				COALESCE(s.shadow_active, r.dest_count) AS shadow_active,
				r.master_schema,
				r.master_table,
				s.master_total AS master_total,
				s.master_active AS master_active,
				r.diff,
				r.status,
				r.error_message,
				r.duration_ms,
				r.checked_at,
				r.shadow_schema,
				r.shadow_table,
				NULL::text AS source_table,
				COALESCE(
					CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.shadow_active, 0) ELSE COALESCE(s.source_active, 0) END,
					r.source_count
				) AS source_count,
				COALESCE(
					CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.master_active, 0) ELSE COALESCE(s.shadow_active, 0) END,
					r.dest_count
				) AS dest_count
			FROM cdc_system.cdc_reconciliation_report r
			LEFT JOIN LATERAL (
				SELECT source_total, source_active, shadow_total, shadow_active, master_total, master_active
				FROM cdc_system.cdc_recon_smoke_result
				WHERE shadow_schema = r.shadow_schema
				  AND shadow_table = r.shadow_table
				  AND (master_schema IS NOT DISTINCT FROM r.master_schema)
				  AND (master_table IS NOT DISTINCT FROM r.master_table)
				  AND segment = r.segment
				ORDER BY checked_at DESC
				LIMIT 1
			) s ON TRUE
			UNION ALL
			SELECT
				id,
				run_id,
				cycle_id,
				segment,
				source_type,
				source_host,
				source_db,
				source_total,
				source_active,
				shadow_total,
				shadow_active,
				master_schema,
				master_table,
				master_total,
				master_active,
				diff,
				status,
				error_message,
				duration_ms,
				checked_at,
				shadow_schema,
				shadow_table,
				source_table,
				CASE WHEN segment = 'shadow_master' THEN COALESCE(shadow_active, 0) ELSE COALESCE(source_active, 0) END AS source_count,
				CASE WHEN segment = 'shadow_master' THEN COALESCE(master_active, 0) ELSE COALESCE(shadow_active, 0) END AS dest_count
			FROM cdc_system.cdc_recon_smoke_result
		  ) unioned
		 ORDER BY shadow_schema, shadow_table, master_schema, master_table, segment, checked_at DESC
	  ) r
      ...
```
