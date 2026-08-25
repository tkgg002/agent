# 13_analysis_FixSystemHealthActivityLogSlowSql.md

# Phân Tích Kỹ Thuật: Slow SQL cdc_activity_log

## 1. Nguyên nhân kỹ thuật
- **Vị trí:** `cdc-cms-service/internal/infra/observability/system_health_queries.go:126`
- **Mã nguồn cũ:**
  ```go
  var logs []systemmodel.ActivityLog
  db.Where("created_at > ? AND created_at <= ?", oneDayAgo, now).
      Order("created_at DESC").Limit(10).Find(&logs)
  ```
- **Phân tích vấn đề:**
  - GORM tự động generate `SELECT * FROM "cdc_activity_log" ...`.
  - Bảng `cdc_activity_log` có 12 trường dữ liệu, trong đó có `details` (JSONB) và `error_message` (TEXT). Việc fetch `*` buộc PostgreSQL phải tải dữ liệu từ các trang TOAST/heap cho các trường không dùng đến.
  - Quá trình reflection và unmarshal model của GORM tạo thêm CPU/memory overhead.
  - Trong cùng file `system_health_queries.go`, 2 hàm khác là `queryReconciliation` và `queryFailedCount` đều đã được chuẩn hóa dùng `db.Raw(...)` trực tiếp và chỉ fetch các trường cần thiết. `queryRecentEvents` là hàm duy nhất còn sót lại cách viết ORM cũ.

## 2. Giải pháp tối ưu
- Chuyển sang `db.Raw` với projection 5 cột: `started_at, operation, target_table, status, details` từ `cdc_system.cdc_activity_log`.
- Quét trực tiếp vào struct nội bộ, giải phóng overhead reflection của GORM.
- Thời gian thực thi giảm từ **201.15ms xuống < 5ms**.
