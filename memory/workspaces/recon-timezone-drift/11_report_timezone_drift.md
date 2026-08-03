# Báo Cáo Triển Khai: Sửa Lỗi Timezone Drift Cho Recon Pipeline (`payment_bills`)

- **Ngày thực hiện:** 2026-07-21
- **Trạng thái:** PASS (Unit tests 100% PASS)

---

## 1. Tổng Quan Thay Đổi

Đã khắc phục triệt để vấn đề lệch timezone 7 tiếng giữa PostgreSQL (`TIMESTAMPTZ`) và MongoDB (`ISODate`) trong reconciliation pipeline bằng cơ chế **Adaptive Schema-Aware Parsing**.

---

## 2. Thống Kê Thay Đổi Code (Physical Evidence)

| File Đường Dẫn | Hành Động | Nội Dung Thay Đổi Chính |
| :--- | :--- | :--- |
| `internal/service/recon/recon_dest_agent.go` | MODIFY | Khởi tạo cache thread-safe (`colTypes map[string]bool` & `colTypesMu RWMutex`) trong `ReconDestAgent` |
| `internal/service/recon/recon_dest_query.go` | MODIFY | Thêm hàm `IsColTimestamptz` kiểm tra kiểu dữ liệu qua `information_schema.columns` có caching; cập nhật `ListIDTsInWindow` dùng adaptive parsing |
| `internal/service/recon/recon_query.go` | MODIFY | Thêm `parsePostgresTimestampWithLocationAndType` phân biệt `TIMESTAMPTZ` (giữ nguyên UTC) vs `TIMESTAMP` (xử lý offset) |
| `internal/service/recon/recon_dest_hash.go` | MODIFY | Cập nhật `HashWindow` gọi `IsColTimestamptz` trước khi parse timestamp dòng dữ liệu |
| `internal/service/recon/recon_dest_agent_test.go` | MODIFY | Thêm unit test `TestDestAgent_HashWindow_DomainTS_Timestamptz` & helper `expectIsColTimestamptzQuery` cho sqlmock |
| `internal/service/recon/recon_tier_a_test.go` | MODIFY | Cập nhật các test case Tier A để mock query `information_schema.columns` cho `HashWindow` |
| `internal/service/recon/recon_smoke_test.go` | MODIFY | Đồng bộ mock expectations cho các test case smoke reconcilation |

---

## 3. Kết Quả Kiểm Thử (Verification Result)

- **Lệnh chạy:** `go test -v ./internal/service/recon/...`
- **Kết quả:** `PASS ok centralized-data-service/internal/service/recon 0.699s`
- **Tất cả unit test (bao gồm test case mới cho TIMESTAMPTZ) đều trôi chảy và đạt 100% PASS.**
