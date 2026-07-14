# Hồ sơ Giải pháp Kỹ thuật - Đồng bộ Cấu trúc Database Đối soát (Reconciliation Schema Alignment)

Tài liệu này chứa mã nguồn chi tiết cần sửa đổi cho từng thành phần.

## 1. cdc-cms-service/migrations/schema/recon_dlq/089_recon_master_metadata.sql

```sql
-- 089_recon_master_metadata.sql — Recon V4: add master_schema and master_table columns to cdc_reconciliation_report
ALTER TABLE cdc_system.cdc_reconciliation_report
  ADD COLUMN IF NOT EXISTS master_schema TEXT,
  ADD COLUMN IF NOT EXISTS master_table  TEXT;
```

## 2. cdc-cms-service/internal/model/recon/reconciliation_report.go

```diff
@@ -47,2 +47,4 @@
 	PrunedMissingSrcDurationMs int `gorm:"column:pruned_missing_src_duration_ms;default:0" json:"pruned_missing_src_duration_ms"`
+	MasterSchema         string `gorm:"column:master_schema" json:"master_schema,omitempty"`
+	MasterTable          string `gorm:"column:master_table" json:"master_table,omitempty"`
 }
```

## 3. centralized-data-service/internal/model/recon/reconciliation_report.go

```diff
@@ -47,2 +47,4 @@
 	PrunedMissingSrcDurationMs int `gorm:"column:pruned_missing_src_duration_ms;default:0" json:"pruned_missing_src_duration_ms"`
+	MasterSchema         string `gorm:"column:master_schema" json:"master_schema,omitempty"`
+	MasterTable          string `gorm:"column:master_table" json:"master_table,omitempty"`
 }
```

## 4. centralized-data-service/internal/service/recon/recon_engine_segment_b.go

```diff
@@ -41,3 +41,4 @@
 func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) *recon.ReconciliationReport {
 	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
+	report.MasterSchema, report.MasterTable = ref.MasterSchema, ref.MasterTable
 	rc.db.Create(report)
 	return report
 }
```
