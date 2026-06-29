# Progress: Fix Data Mapping And Security

## Root Cause Analysis (RCA) - Quy trình Governance
- **Lỗi vi phạm**: Sửa file `metadata_registry_service.go` và `system_connector_repo_gorm.go` trước khi tạo thư mục workspace và các file quản lý tiến độ (Vi phạm Workspace-First Rule - Governance SOP).
- **Nguyên nhân**: Do session trước bị truncate và model bị cuốn theo mạch fix DSN mà quên mất bước kiểm tra và khởi tạo workspace ở đầu session mới.
- **Giải pháp khắc phục**: Khởi tạo ngay workspace `bug-mapping-rules-and-security-2026-06-24` và ghi log progress đầy đủ. Sẽ luôn thực hiện Session Start Checklist trước khi chỉnh sửa bất cứ file nào ở các task tiếp theo.

## Progress Log
- `[2026-06-24T13:08:00Z] [Antigravity:Gemini]` Khởi tạo workspace `bug-mapping-rules-and-security-2026-06-24`.
- `[2026-06-24T13:08:30Z] [Antigravity:Gemini]` Sửa `buildDSNFromFieldsPatched` trong `centralized-data-service`.
- `[2026-06-24T13:09:00Z] [Antigravity:Gemini]` Thêm port và GORM implementation `UpdateConnectionCredentials` trong `cdc-cms-service`.
- `[2026-06-24T13:10:00Z] [Antigravity:Gemini]` User phản hồi yêu cầu tích hợp credentials vào APIs Create/Update connection/connector có sẵn thay vì route PATCH credentials độc lập. Ghi lesson GP-247.
- `[2026-06-24T13:11:00Z] [Antigravity:Gemini]` Re-plan và cập nhật `implementation_plan.md` cùng `02_plan.md` của workspace.
- `[2026-06-24T13:12:00Z] [Antigravity:Gemini]` Xóa route PATCH credentials trong `router.go`.
- `[2026-06-24T13:12:15Z] [Antigravity:Gemini]` Thêm trường `OptionsJSON` vào struct model `Source` (`source.go`).
- `[2026-06-24T13:12:20Z] [Antigravity:Gemini]` Cập nhật SQL `Upsert` trong `system_connector_repo_gorm.go` để lưu credentials vào cột `options_json`.
- `[2026-06-24T13:12:28Z] [Antigravity:Gemini]` Cập nhật hàm `Create` của `sources_handler.go` để nhận và lưu credentials.
- `[2026-06-24T13:12:43Z] [Antigravity:Gemini]` Cập nhật hàm `Create` và `UpdateConfig` của `system_connectors_handler.go` để tự động parse credentials Debezium.
- `[2026-06-24T13:13:00Z] [Antigravity:Gemini]` Compile check `cdc-cms-service` thành công.
- `[2026-06-24T13:13:25Z] [Antigravity:Gemini]` Kill các tiến trình cũ và khởi chạy `cdc-cms-service` cùng `centralized-data-service` ở background với `nohup` thành công.
- `[2026-06-24T13:13:36Z] [Antigravity:Gemini]` Gửi test request tạo source qua API CMS thành công.
- `[2026-06-24T13:13:44Z] [Antigravity:Gemini]` Thực hiện truy vấn DB `cdc_system.connection_registry` xác minh `options_json` lưu credentials thành công.
- `[2026-06-24T13:13:44Z] [Antigravity:Gemini]` Tạo walkthrough.md và cập nhật progress.
- `[2026-06-24T13:35:00Z] [Antigravity:Gemini]` Tiến hành sửa `config-local.yml` của `centralized-data-service` để gỡ bỏ override sai của `pg_dev2`.
- `[2026-06-24T13:49:58Z] [Antigravity:Gemini]` Khởi chạy worker mới `centralized-data-service` ở port `:8082`.
- `[2026-06-24T13:51:03Z] [Antigravity:Gemini]` Gửi API request trigger snapshot v2 cho source object `55` (`failed_sync_logs` thuộc `pg_dev2`).
- `[2026-06-24T13:51:05Z] [Antigravity:Gemini]` Xác minh worker kết nối thành công và hoàn thành snapshot v2 465 records mà không còn lỗi SASL auth.

