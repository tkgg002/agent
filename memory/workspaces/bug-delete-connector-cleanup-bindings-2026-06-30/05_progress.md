# Progress: Sửa Lỗi Đối Soát Các Bảng Thuộc Connector Đã Bị Xóa

## Metadata Integrity
- **2026-06-30 14:20:00 +0700 [Agent:Gemini 3.5 Flash (High)]** Action: Khởi tạo workspace `bug-delete-connector-cleanup-bindings-2026-06-30`.
- **2026-06-30 14:26:00 +0700 [Agent:Antigravity]** Action: Bắt đầu phase implementation. Khởi tạo task.md.

## Root Cause Analysis (Governance & Configuration)
- **Vấn đề**: Khi xóa connector, shadow/master bindings vẫn tồn tại và hàm đối soát `recon smoke` tiếp tục quét chúng.
- **Gốc rễ (Root Cause)**: Hàm `FullCleanup` trong `system_connector_repo_gorm.go` của cdc-cms-service chỉ cập nhật `shadow_connection_id` và `master_connection_id` thành NULL thay vì thực hiện DELETE, để lại các bindings mồ côi. Đồng thời, `ReloadAll` của centralized-data-service vẫn load các bindings mồ côi này do thiếu bộ lọc trạng thái active của connections.

## Phân tích Gốc rễ (Root Cause) Vi phạm Quy trình Governance
- **Lỗi vi phạm**: Không có vi phạm quy trình Governance nào.
- **Nguyên nhân gốc rễ**: N/A.
- **Hành động khắc phục**: N/A.

## Tiến độ thực hiện
- [/] Lập kế hoạch và phân tích hiện trạng.
- [ ] Sửa đổi `FullCleanup` ở `cdc-cms-service`.
- [ ] Kiểm tra và lọc bindings ở `centralized-data-service`.
- [ ] Xác minh kết quả.

