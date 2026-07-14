# Audit Log / Tiến độ - Quản lý Index qua CMS UI (Index Manager)

- [2026-07-08T16:42:00Z] [Agent:Gemini] Khởi tạo workspace feat-index-manager-ui-2026-07-08. Đã tạo file requirements và progress.
- [2026-07-08T16:43:00Z] [Agent:Gemini] Bắt đầu thiết kế chi tiết solution cho Worker, CMS Service và Frontend.
- [2026-07-08T16:45:00Z] [Agent:Gemini] (MUSCLE) Tiến hành tạo file service/governance/index_manager.go và handler/governance/index_handler.go trên worker.
- [2026-07-08T16:46:00Z] [Agent:Gemini] (MUSCLE) Hoàn thành test-suite và compile worker thành công. Bắt đầu implement các endpoint trong cdc-cms-service.
- [2026-07-08T16:47:00Z] [Agent:Gemini] (MUSCLE) Đã implement xong IntrospectionHandler và SetupRoutes trên CMS Service, compile thành công. Bắt đầu tạo component TableIndexManager trên Frontend.
- [2026-07-08T16:48:00Z] [Agent:Gemini] (MUSCLE) Đã viết xong TableIndexManager.tsx, tích hợp vào MappingFieldsPage và MasterMappingFieldsPage. Frontend build thành công. Toàn bộ index-manager feature đã sẵn sàng.
- [2026-07-08T16:56:00Z] [Agent:Gemini] (MUSCLE) Phát hiện lỗi validation WHERE clause chặn nhầm system field `_deleted = true` (chứa substring `DELETE`). Đã khắc phục bằng cách sử dụng Regex với ranh giới từ `\bDELETE\b`. Bổ sung unit test TestIndexManager_UnsafeWhere và pass 100%.
- [2026-07-08T17:06:00Z] [Agent:Gemini] (MUSCLE) Điều chỉnh logic đề xuất index ở frontend: Chỉ khuyến nghị các index đặc thù của CDC shadow (`_deleted`, `_source_ts`, `_source_id`) khi `plane === 'shadow'`. Bảng Master sẽ không nhận các khuyến nghị này.
- [2026-07-08T17:17:00Z] [Agent:Gemini] (MUSCLE) Thay đổi UI SourceConnectors: thay thế cột URL / Host bằng Connector Name / Topic. Loại bỏ hoàn toàn server_address/URL để tránh lỗi hiển thị `<hidden_or_invalid_url>` và đảm bảo an toàn. Xóa hàm maskAddress không còn sử dụng. Frontend build thành công.
- [2026-07-08T17:50:00Z] [Agent:Gemini] (MUSCLE) Cập nhật ReconPipelineGrid.tsx: Thay thế hiển thị `source_host` bằng `source_connection_code` để loại bỏ hoàn toàn việc hiển thị `<hidden_or_invalid_url>` trong màn hình Data Integrity. Frontend build thành công.
