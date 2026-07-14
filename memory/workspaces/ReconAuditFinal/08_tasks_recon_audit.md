# Danh sách các Task chi tiết cho việc Audit Reconciliation

- [x] Phân tích code mới `internal/handler/recon` và so sánh với `recon_bk`.
- [x] Xác định luồng đi (flow) của 3 trường hợp: Lookback, Full Search (Full Diff) và Deep Check.
- [x] Làm rõ cơ chế validation 30 ngày và việc loại trừ lẫn nhau (mutually exclusive) của 3 tuỳ chọn trên UI.
- [x] Tạo tài liệu phân tích chi tiết `13_analysis_recon_audit.md`.
- [x] Đề xuất kế hoạch sửa test compile error trong handler test và decommission recon_bk.
- [x] Sửa đổi và cập nhật `recon_heal_v4_test.go` sử dụng `HealHandler` và hỗ trợ tương thích ngược.
- [x] Di chuyển `scan_array_path_test.go` và `scan_handler_test.go` sang gói `internal/handler/scan/`.
- [x] Chạy unit tests và verify pass 100%.
- [x] Xóa bỏ hoàn toàn thư mục legacy `internal/handler/recon_bk/`.
- [x] Cập nhật tài liệu tiến độ và tạo walkthrough báo cáo.
- [x] Chuyển lưu toàn bộ mã nguồn legacy từ lịch sử Git sang thư mục `docs/recon_legacy/` dưới dạng các file `.go.bak` để tránh xung đột build.



