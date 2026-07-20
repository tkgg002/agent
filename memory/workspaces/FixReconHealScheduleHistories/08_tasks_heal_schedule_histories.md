# Danh sách các task chi tiết: Sửa lỗi click Heal bảng schedule_histories

- [x] Task 1: Thiết kế giải pháp phân giải tên bảng trong `resolveTargetTableConfig` để hỗ trợ trích xuất tên bảng thuần túy.
- [x] Task 2: Cập nhật code trong `centralized-data-service/internal/handler/recon/recon_base_handler.go`.
- [x] Task 3: Chạy test integration hiện tại của recon/heal để xác minh logic.
- [x] Task 4: Cập nhật hàm `loadMaster` trong `transmuter.go` để hỗ trợ bóc tách schema prefix cho Segment B (`SegmentShadowMaster`) và chạy unit test package `master`.
- [x] Task 5: Chạy thủ công lệnh Heal cho report ID 42 để kiểm tra kết quả thực tế trên DB.
