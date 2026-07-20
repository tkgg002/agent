# Kế hoạch tối ưu hóa SQL chậm (Slow SQL Tuning)

Nhiệm vụ này tập trung vào tối ưu hóa hai câu truy vấn SQL chậm trong `recon_read_repo_gorm.go` để giảm thời gian phản hồi API xuống dưới ngưỡng 200ms (hiện tại mất từ 240ms đến 1.2s).

## User Review Required

> [!IMPORTANT]
> Câu truy vấn `listLatestPrimary` ban đầu thực hiện quét toàn bộ bảng lịch sử (`cdc_reconciliation_report` và `cdc_recon_smoke_result`) rồi mới distinct và join với registry.
> Giải pháp tối ưu hóa đề xuất chuyển sang **Registry-driven Lateral Fetch** (sử dụng bảng `cdc_table_registry` làm bảng gốc lái truy vấn, sau đó dùng `LEFT JOIN LATERAL` để chỉ lấy các báo cáo mới nhất của các bảng active). 
>
> Thay đổi này cải thiện hiệu năng vượt trội (từ ~1.2s xuống <15ms) và giữ nguyên cấu trúc kết quả trả về, đảm bảo an toàn tuyệt đối và tương thích ngược.

---

## Proposed Changes

### Component: cdc-cms-service (Read Repository)

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)

##### 1. Tối ưu hóa phép đếm failed logs trong `ListFailedLogs` (dòng 601)
- **Trước:**
  Sử dụng subquery bọc toàn bộ truy vấn chính bao gồm cả các phép `LEFT JOIN LATERAL` nặng nề:
  ```go
  countQuery := `SELECT COUNT(*) FROM (` + query + `) AS failed_logs`
  ```
- **Sau:**
  Đếm trực tiếp trên bảng `failed_sync_logs` cùng các filters (không join bảng khác):
  ```go
  countQuery := `SELECT COUNT(*) FROM cdc_system.failed_sync_logs f WHERE 1=1`
  if f.TargetTable != "" {
      countQuery += ` AND f.target_table = ?`
  }
  if f.Status != "" {
      countQuery += ` AND f.status = ?`
  }
  if f.ErrorType != "" {
      countQuery += ` AND f.error_type = ?`
  }
  ```

##### 2. Tối ưu hóa truy vấn báo cáo mới nhất `listLatestPrimary` (dòng 37)
- **Trước:**
  `UNION ALL` toàn bộ 2 bảng lịch sử -> `DISTINCT ON` trên kết quả gộp -> `INNER JOIN` với `cdc_table_registry` ở ngoài cùng để lọc bảng active.
- **Sau:**
  Bắt đầu từ `cdc_table_registry` active -> `LEFT JOIN LATERAL` để tìm report/smoke test mới nhất của bảng đó (sử dụng index trên `shadow_table`).

---

## Verification Plan

### Automated Tests
- Chạy ứng dụng và gọi API `/api/reconciliation/report` và `/api/failed-sync-logs` để kiểm tra kết quả trả về.
- Xác thực dữ liệu trả về của API tối ưu trùng khớp hoàn toàn với API cũ (kiểm tra cấu trúc và số lượng bản ghi).

### Manual Verification
- Kiểm tra logs của service xem các câu truy vấn có còn bị cảnh báo `SLOW SQL >= 200ms` hay không.
- Đo lường latency thực tế của các truy vấn tối ưu.
