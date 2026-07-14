# Hồ sơ giải pháp kỹ thuật - Đồng bộ Schema Đối soát Shadow/Master

## Giải pháp cụ thể cho cdc-cms-service

### Sửa file `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
Chúng ta sẽ bổ sung cột `master_schema` vào danh sách trường SELECT của UNION query ở hàm `GetTableHistory`.

#### Diff chi tiết:
```diff
diff --git a/internal/infra/persistence/recon/recon_read_repo_gorm.go b/internal/infra/persistence/recon/recon_read_repo_gorm.go
index xxxxxxx..xxxxxxx 100644
--- a/internal/infra/persistence/recon/recon_read_repo_gorm.go
+++ b/internal/infra/persistence/recon/recon_read_repo_gorm.go
@@ -232,7 +232,8 @@ func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSc
 			total_dest_count,
 			check_type,
 			tier,
-			master_table
+			master_table,
+			master_schema
 		FROM cdc_system.cdc_reconciliation_report
 		UNION ALL
 		SELECT
@@ -254,7 +255,8 @@ func (r *reconReadRepoGorm) GetTableHistory(ctx context.Context, table, shadowSc
 			CASE WHEN segment = 'shadow_master' THEN master_total ELSE shadow_total END AS total_dest_count,
 			'smoke' AS check_type,
 			1 AS tier,
-			master_table
+			master_table,
+			master_schema
 		FROM cdc_system.cdc_recon_smoke_result
 	`
```
