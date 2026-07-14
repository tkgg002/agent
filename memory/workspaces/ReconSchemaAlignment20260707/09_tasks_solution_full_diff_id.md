# Hồ sơ giải pháp kỹ thuật - Khắc phục Hiển thị Dữ liệu ID Diff

## Sửa file `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`

Bổ sung các trường ID diff và heal metrics vào mệnh đề SELECT của truy vấn UNION ALL.

#### Diff chi tiết:
```diff
diff --git a/internal/infra/persistence/recon/recon_read_repo_gorm.go b/internal/infra/persistence/recon/recon_read_repo_gorm.go
index xxxxxxx..xxxxxxx 100644
--- a/internal/infra/persistence/recon/recon_read_repo_gorm.go
+++ b/internal/infra/persistence/recon/recon_read_repo_gorm.go
@@ -233,7 +233,22 @@ func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSc
 			total_dest_count,
 			check_type,
 			tier,
 			master_table,
-			master_schema
+			master_schema,
+			missing_count,
+			missing_ids,
+			stale_count,
+			stale_ids,
+			field_diffs,
+			orphan_count,
+			healed_at,
+			healed_count,
+			healed_duration_ms,
+			healed_mismatched_count,
+			healed_mismatched_duration_ms,
+			healed_missing_dest_count,
+			healed_missing_dest_duration_ms,
+			pruned_missing_src_count,
+			pruned_missing_src_duration_ms
 		FROM cdc_system.cdc_reconciliation_report
 		UNION ALL
 		SELECT
@@ -256,7 +271,22 @@ func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSc
 			'smoke' AS check_type,
 			1 AS tier,
 			master_table,
-			master_schema
+			master_schema,
+			0::integer AS missing_count,
+			NULL::jsonb AS missing_ids,
+			0::integer AS stale_count,
+			NULL::jsonb AS stale_ids,
+			NULL::jsonb AS field_diffs,
+			0::integer AS orphan_count,
+			NULL::timestamp without time zone AS healed_at,
+			0::integer AS healed_count,
+			0::integer AS healed_duration_ms,
+			0::integer AS healed_mismatched_count,
+			0::integer AS healed_mismatched_duration_ms,
+			0::integer AS healed_missing_dest_count,
+			0::integer AS healed_missing_dest_duration_ms,
+			0::integer AS pruned_missing_src_count,
+			0::integer AS pruned_missing_src_duration_ms
 		FROM cdc_system.cdc_recon_smoke_result
 	`
```
