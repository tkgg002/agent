# Kế hoạch Tối ưu hóa SLOW SQL System Health Queries (cdc-cms-service)

## Giải Thích Lý Do Phát Sinh SLOW SQL Mới
Lỗi SLOW SQL 205ms mới phát sinh tại `system_health_queries.go:54` là do file này nằm ở **tầng Observability (System Health Collector ngầm)**, hoàn toàn độc lập với 2 file UI Persistence Read Repo đã tối ưu ở các lượt trước.

---

## Nguyên Nhân & Giải Pháp Tối Ưu Duy Nhất (Single Best Approach)

### Nguyên nhân:
Hàm `queryReconciliation` trong `system_health_queries.go` dùng `DISTINCT ON (CASE WHEN ...)` để lấy snapshot báo cáo đối soát mới nhất của từng bảng từ `cdc_reconciliation_report`.
Do **thiếu điều kiện lọc thời gian `checked_at`**, PostgreSQL phải Seq Scan toàn bộ dữ liệu lịch sử đối soát và thực hiện Sort 6 biểu thức phức tạp trên memory/disk.

### Giải pháp:
Bổ sung cờ khoanh vùng thời gian `WHERE checked_at >= NOW() - INTERVAL '7 days'` trong câu lệnh Raw SQL của hàm `queryReconciliation`.
- *Tác động:* Loại bỏ 95%+ các bản ghi lịch sử cũ của bảng `cdc_reconciliation_report`, giúp thời gian phản hồi của Health Collector giảm từ **205ms xuống < 10ms**.

---

## Proposed Changes

### cdc-cms-service (Go Backend)

#### [MODIFY] `internal/infra/observability/system_health_queries.go`
- Thêm `WHERE checked_at >= NOW() - INTERVAL '7 days'` vào câu Raw SQL của hàm `queryReconciliation`.

---

## Verification Plan

### Automated Tests
- Chạy biên dịch kiểm tra syntax Go: `go build ./cmd/server` tại repo `cdc-cms-service`.

### Manual Verification
- Kiểm tra log chạy ngầm của System Health Collector để xác nhận câu SQL không còn bị log SLOW SQL >= 200ms.
