# 12_implementation_plan_FixSystemHealthActivityLogSlowSql.md

# Kế Hoạch Triển Khai: Khắc Phục Slow SQL tại `system_health_queries.go:126`

## 1. Mục tiêu
Tối ưu hóa hàm `queryRecentEvents` trong `internal/infra/observability/system_health_queries.go` để loại bỏ log cảnh báo `[SLOW SQL >= 200ms]`, đưa latency xuống dưới 10ms mà không làm thay đổi contract output JSON của System Health Snapshot.

## 2. Các Bước Triển Khai
1. **Bước 1 (Refactor SQL & Struct Mapping):**
   - Thay `db.Where(...).Order(...).Limit(...).Find(&logs)` bằng `db.Raw(...)`.
   - Chọn tường minh 5 trường: `started_at, operation, target_table, status, details`.
   - Quét vào struct nội bộ `rows []struct { ... }`.
   - Bổ sung error handling debug log `c.logger.Debug("query recent events", zap.Error(err))`.
2. **Bước 2 (Kiểm định tính đúng đắn):**
   - Kiểm tra kiểu dữ liệu `details` (json.RawMessage) đảm bảo serialization sang map result string(l.Details) không bị thay đổi định dạng.
   - Đảm bảo `time` nhận giá trị `l.StartedAt`.
3. **Bước 3 (Đóng gói & Báo cáo):**
   - Cập nhật tiến độ `05_progress_*.md`.
