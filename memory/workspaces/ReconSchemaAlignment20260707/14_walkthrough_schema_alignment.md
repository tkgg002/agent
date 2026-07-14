# Báo cáo Kết quả (Walkthrough) - Đồng bộ Schema Đối soát Shadow/Master

## Tóm tắt công việc đã thực hiện
Đã khắc phục hoàn toàn sự lệch cấu trúc (schema drift) đối với cột `master_schema` và `master_table` trên bảng `cdc_reconciliation_report`. Đảm bảo luồng ghi của ingestion worker (`stampB`) và luồng đọc API (`GetTableHistory`) hoạt động đồng bộ, trả về đầy đủ metadata lineage cho UI dashboard đối soát.

## Các thay đổi chính

### 1. Database Schema
- Migration `089_recon_master_metadata.sql` bổ sung hai cột `master_schema` và `master_table` vào bảng `cdc_reconciliation_report`. Cả 2 cột đã được xác minh thành công trên database `cdc_dw`.

### 2. GORM Models Alignment
- Đồng bộ struct `ReconciliationReport` ở cả 2 service `cdc-cms-service` và `centralized-data-service` để khai báo GORM mappings với `master_schema` và `master_table`.

### 3. Ingestion Engine
- Cập nhật hàm `stampB` trong `centralized-data-service` để lưu thông tin `MasterSchema` và `MasterTable` từ `MasterBindingRef` vào `ReconciliationReport`.

### 4. API & Persistent Layer Query
- Cập nhật hàm `GetTableHistory` tại `recon_read_repo_gorm.go` để SELECT thêm cột `master_schema` trong UNION query (gộp từ cả table `cdc_reconciliation_report` và `cdc_recon_smoke_result`).

## Kết quả kiểm thử & xác minh

### 1. Unit Tests
- Chạy unit test của package queries trong `cdc-cms-service`:
  ```bash
  go test -v ./test/internal/app/queries/...
  ```
  **Kết quả**: `PASS` 100%.

### 2. API cURL Verification
- Thực hiện cURL request kiểm tra endpoint `/api/reconciliation/report/export_jobs?page=1&page_size=30`:
  ```bash
  curl -s -H "Authorization: Bearer dev-token" "http://localhost:8083/api/reconciliation/report/export_jobs?page=1&page_size=30" | jq .
  ```
  **Kết quả**: Các bản ghi thuộc phân đoạn `shadow_master` đã trả về đầy đủ metadata mong muốn:
  ```json
  "master_schema": "master_centrallized_export_service",
  "master_table": "export_jobs"
  ```
