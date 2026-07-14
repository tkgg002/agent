# Kế hoạch Triển khai Chi tiết - Đồng bộ Cấu trúc Database Đối soát (Reconciliation Schema Alignment)

Kế hoạch này giải quyết triệt để lỗi biên dịch SQL `SQLSTATE 42703 (column "master_schema" does not exist)` khi truy vấn danh sách báo cáo đối soát. Bằng cách thêm cột `master_schema` và `master_table` vào bảng `cdc_system.cdc_reconciliation_report` và ánh xạ chúng qua các GORM models tương ứng ở cả hai service.

## User Review Required

> [!IMPORTANT]
> - Chúng ta sẽ tạo một migration mới `089_recon_master_metadata.sql` trong `cdc-cms-service` để tự động cập nhật schema database khi khởi chạy.
> - Cập nhật GORM model `ReconciliationReport` ở cả hai dự án `cdc-cms-service` và `centralized-data-service` để đảm bảo tính nhất quán của dữ liệu.
> - Bổ sung việc gán giá trị `MasterSchema` và `MasterTable` trong logic ghi báo cáo phân đoạn B (`stampB`) tại `centralized-data-service`.

## Proposed Changes

### 1. Database Migration (cdc-cms-service)

#### [NEW] [089_recon_master_metadata.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/089_recon_master_metadata.sql)
- Tạo tệp tin migration mới để bổ sung hai cột thiếu vào bảng `cdc_reconciliation_report`:
  ```sql
  -- 089_recon_master_metadata.sql — Recon V4: add master_schema and master_table columns to cdc_reconciliation_report
  ALTER TABLE cdc_system.cdc_reconciliation_report
    ADD COLUMN IF NOT EXISTS master_schema TEXT,
    ADD COLUMN IF NOT EXISTS master_table  TEXT;
  ```

### 2. GORM Models Alignment

#### [MODIFY] [reconciliation_report.go (cdc-cms-service)](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go)
- Thêm hai trường `MasterSchema` và `MasterTable` vào struct `ReconciliationReport`:
  ```go
  MasterSchema string `gorm:"column:master_schema" json:"master_schema,omitempty"`
  MasterTable  string `gorm:"column:master_table" json:"master_table,omitempty"`
  ```

#### [MODIFY] [reconciliation_report.go (centralized-data-service)](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
- Thêm hai trường tương tự vào struct `ReconciliationReport` để đồng bộ GORM model ghi dữ liệu.

### 3. Data Ingestion Engine (centralized-data-service)

#### [MODIFY] [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)
- Cập nhật hàm `stampB` để gán giá trị cho hai cột này trước khi insert vào DB:
  ```diff
   func (rc *ReconCore) stampB(report *recon.ReconciliationReport, ref MasterBindingRef) *recon.ReconciliationReport {
   	report.ShadowSchema, report.ShadowTable, report.RunID = ref.ShadowSchema, ref.ShadowTable, ref.RunID
  +	report.MasterSchema, report.MasterTable = ref.MasterSchema, ref.MasterTable
   	rc.db.Create(report)
   	return report
   }
  ```

---

## Verification Plan

### Automated Tests & Compilation
- Kiểm tra biên dịch ở cả 2 service:
  ```bash
  # Tại centralized-data-service
  go build ./...
  
  # Tại cdc-cms-service
  go build ./...
  ```
- Chạy migrations để kiểm tra cấu trúc cơ sở dữ liệu mới.
- Chạy unit test của read repository `reconReadRepoGorm` trong `cdc-cms-service`:
  ```bash
  go test -v ./internal/infra/persistence/recon/...
  ```
