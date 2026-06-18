# Tasks Log for Workspace: feat-api-handlers-hexagonal-refactor-2026-06-16

Đây là bảng ghi nhận chi tiết nhiệm vụ và lệnh bàn giao (Delegate) cho Muscle (CC CLI) thực thi từng giai đoạn của kế hoạch tái cấu trúc API Handlers.

---

## Task: Refactor Shadow API Handlers (Phase 1)
- **Phase**: GĐ3 (Architecture - CQRS & Hexagonal)
- **Service Group**: Utilities / Business
- **Service(s)**: cdc-cms-service (internal/api/shadow/)
- **Mô tả**: Bóc tách các Fat Handlers trong thư mục `internal/api/shadow/` sang cấu trúc Hexagonal, định nghĩa các Ports, Application Queries/Commands mới và đăng ký chúng trong `internal/server/server.go`.
- **Trạng thái**: [x] COMPLETED

### [Context]
- **API Handlers cần bóc tách**:
  1. `mapping_preview_handler.go`: Đọc sample data và thực thi logic `gjson` trích xuất JSONPath trực tiếp.
  2. `mapping_rule_handler_commands.go` (Method `Reload`): Gọi trực tiếp `natsClient.PublishReload`.
  3. `mapping_rule_handler_batch.go` (Method `BatchUpdate`): Chứa logic lặp, dispatch các sub-commands và gọi `natsClient.PublishReload` trực tiếp.
- **Dependencies**: `internal/server/server.go` (Đăng ký handlers), `pkgs/natsconn` (NATS Client).

### [Definition of Done]
- [x] **Tạo Domain Port cho Preview**:
  - Tạo file `internal/domain/mapping/preview.go` định nghĩa struct `PreviewSample` và interface `ShadowPreviewRepository`.
- [x] **Tạo Application Query cho Preview**:
  - Tạo query handler `internal/app/queries/shadow/preview_mapping.go` chứa logic eval gjson và kiểm soát lỗi (ví dụ: `ErrBindingNotFound`).
- [x] **Tạo Infrastructure Adapter cho Preview**:
  - Tạo GORM repository `internal/infra/persistence/shadow_preview_repo_gorm.go` triển khai `ShadowPreviewRepository`.
- [x] **Bóc tách logic Preview trong API**:
  - Sửa `internal/api/shadow/mapping_preview_handler.go` để inject `*shadowQueries.PreviewMappingHandler` và gọi nó thông qua Handle.
- [x] **Bóc tách logic Reload trong API**:
  - Tạo Command `ReloadMappingCommand` trong `internal/app/commands/shadow/reload_mapping.go` (nhận arguments từ API và gọi `nats.PublishReload`).
  - Đăng ký Command này vào `cmdBus` trong `internal/server/server.go` with key `"shadow-mapping.reload"`.
  - Cập nhật handler `Reload` trong `internal/api/shadow/mapping_rule_handler_commands.go` để dispatch command này qua `h.bus.Execute`.
- [x] **Bóc tách logic BatchUpdate trong API**:
  - Tạo Command `BatchUpdateMappingRulesCommand` trong `internal/app/commands/shadow/batch_update_rules.go` chứa logic nghiệp vụ chạy vòng lặp, dispatch các sub-commands (status update, alter column, backfill) và gửi NATS reload.
  - Đăng ký Command này vào `cmdBus` trong `internal/server/server.go` với key `"shadow-mapping.batch-update"`.
  - Cập nhật handler `BatchUpdate` trong `internal/api/shadow/mapping_rule_handler_batch.go` để chỉ parse JSON payload và dispatch Command này qua `h.bus.Execute`.
- [x] **[QA Gate]**:
  - Biên dịch thành công dự án (`go build ./...`).
  - Chạy toàn bộ test suite thành công (`go test ./...`).
- [x] **Model Tracking**: Ghi nhận tiến trình vào `05_progress.md` với tag model.

