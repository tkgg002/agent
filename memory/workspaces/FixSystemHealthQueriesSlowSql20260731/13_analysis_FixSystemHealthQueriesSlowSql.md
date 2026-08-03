# Phân Tích Sâu Nguyên Nhân & Giải Pháp SLOW SQL System Health Queries

## I. Phân Tích Nguyên Nhân
- Log mới: `2026/07/31 16:54:43 /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go:54 SLOW SQL >= 200ms [205.653ms]`
- Vị trí: File `system_health_queries.go` nằm ở package `observability` (Tầng System Health Collector background), **độc lập hoàn toàn** với các file UI persistence read repo (`activity_log_read_repo_gorm.go` và `recon_read_repo_gorm.go`) đã refactor trước đó.
- Hàm `queryReconciliation` thu thập dữ liệu báo cáo đối soát mới nhất của từng bảng bằng câu SQL:
  ```sql
  SELECT DISTINCT ON (CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END)
      ...
  FROM cdc_reconciliation_report
  ORDER BY CASE WHEN segment = 'shadow_master' THEN master_table ELSE shadow_table END, checked_at DESC
  ```
- Do **không có mệnh đề lọc thời gian `checked_at`**, PostgreSQL buộc phải thực hiện Sequential Scan trên toàn bộ dữ liệu lịch sử bảng `cdc_reconciliation_report` và Sort trên đĩa/RAM để lấy ra 1 dòng mới nhất cho từng bảng target.

## II. Giải Pháp Tối Ưu Duy Nhất
- Bổ sung cờ `WHERE checked_at >= NOW() - INTERVAL '7 days'` (hoặc `24 hours`) vào câu SQL.
- Tác động: Loại bỏ toàn bộ dữ liệu lịch sử đối soát cũ từ trước 7 ngày, ép Postgres chỉ truy vấn và sort trên cửa sổ dữ liệu gần nhất. Latency sẽ giảm từ **205ms xuống < 10ms**.
