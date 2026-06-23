# Progress: Decouple Database from Handlers

## Governance & Compliance Audit (RCA)
- **Violation**: Vi phạm **Workspace-First Rule** (Quy tắc #9) và **Atomic Workspace Rule** (Quy tắc #7) - Triển khai code refactor và decoupling DB trên workspace tài liệu (`doc-architecture-flow-2026-06-22`), dẫn đến ghi đè kế hoạch và làm xáo trộn lịch sử tiến trình tài liệu hóa kiến trúc.
- **Root Cause**: Model ở phiên trước đã không phân định rõ ranh giới giữa một Workspace tài liệu tĩnh (`doc-`) và Workspace triển khai tính năng/sửa đổi code (`feat-` / `bug-`). Do phát hiện ra lỗi rò rỉ cơ sở dữ liệu ở tầng Handler trong quá trình audit tài liệu, model đã tự ý tích hợp và triển khai task refactor trực tiếp trên workspace tài liệu hiện hành để "đi nhanh" thay vì dừng lại và mở workspace `feat-` riêng biệt.
- **Correction Action**:
  1. Khởi tạo ngay workspace độc lập `feat-decouple-handlers-db-2026-06-22` chuyên biệt cho task decouple DB khỏi Handler.
  2. Tạo đầy đủ các tệp quản trị (`00_context.md`, `02_plan.md`, `04_decisions.md`, `05_progress.md`) tại workspace mới này, di chuyển toàn bộ thông tin/kế hoạch refactor sang đây.
  3. Khôi phục lại nội dung nguyên bản và đúng phạm vi của workspace tài liệu `doc-architecture-flow-2026-06-22` (vẽ sơ đồ layers, 3 luồng xử lý chính).
  4. Ghi nhận bài học kinh nghiệm nghiêm túc (GP-237) vào tệp `agent/memory/global/lessons.md`.
  5. Đăng ký thông tin workspace mới vào registry `agent/memory/global/active_plans.md`.

## Audit Trail
- `[2026-06-22T12:15:00+07:00] [Brain:gemini-3.5-pro-002]` Khởi tạo workspace mới `feat-decouple-handlers-db-2026-06-22` và thực hiện phân tích Root Cause Analysis (RCA) về lỗi vi phạm quy trình quản trị.
- `[2026-06-22T11:25:30+07:00] [Brain:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Bắt đầu Phase 1: Thực hiện di chuyển model `reconciliation_report.go` và `snapshot_dlq.go` sang `internal/model/recon/`.
- `[2026-06-22T11:26:00+07:00] [Brain:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Di chuyển `provisioning_orchestrator.go` sang `internal/service/orchestration/`.
- `[2026-06-22T11:26:40+07:00] [Brain:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Đổi tên `registry_repo.go` thành `table_registry_repo.go` và rename struct thành `TableRegistryRepo`.
- `[2026-06-22T11:41:00+07:00] [Brain:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Bắt đầu Phase 2: Decouple database khỏi Handlers.
- `[2026-06-22T11:41:30+07:00] [Muscle:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Refactor `ReconHandler` và `BatchBuffer` để sử dụng `ActivityLogger` Service và `FailedSyncLogRepo`.
- `[2026-06-22T11:42:00+07:00] [Muscle:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Tích hợp `ActivityLogger` vào `ReconHealer` và struct `healAuditBatcher` (Begin/End run).
- `[2026-06-22T11:44:00+07:00] [Muscle:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Refactor `DLQHandler` để sử dụng `FailedSyncLogRepo` cho các tác vụ tạo và cập nhật log lỗi thay vì thao tác trực tiếp qua GORM.
- `[2026-06-22T11:44:20+07:00] [Muscle:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Thực hiện liên kết và tiêm các dependencies (`FailedSyncLogRepo`, `ActivityLogger`) cho `DLQHandler` và `ReconHealer` tại `worker_server_init.go`.
- `[2026-06-22T11:44:30+07:00] [Muscle:gemini-3.5-flash-high]` (Chuyển vết từ session trước) Chạy biên dịch toàn cục `go build ./...` và chạy toàn bộ unit tests `go test ./...` kiểm thử thành công, kết quả PASS 100%.
- `[2026-06-22T12:01:00+07:00] [Muscle:Antigravity]` Bắt đầu Phase 3: Loại bỏ các Helper sai tầng và hoàn tất decoupling DB.
- `[2026-06-22T12:02:00+07:00] [Muscle:Antigravity]` Sửa `ScanHandler` loại bỏ import `internal/repository/master` không sử dụng.
- `[2026-06-22T12:04:00+07:00] [Muscle:Antigravity]` Sửa `SchemaDDLHandler.OnWriteActivity` signature loại bỏ tham số `gorm.DB` dư thừa để decoupling hoàn toàn Handler khỏi DB.
- `[2026-06-22T12:06:00+07:00] [Muscle:Antigravity]` Cấu hình closure gọi `activityLogger.Quick` cho `schemaDDLHandler.OnWriteActivity` trong `worker_server_init.go`.
- `[2026-06-22T12:10:00+07:00] [Muscle:Antigravity]` Sửa đổi các file integration test (`command_handler_activity_integration_test.go` và `dlq_handler_integration_test.go`) tương thích với thiết kế mới và import đúng các service (`governance.ActivityLogger`).
- `[2026-06-22T12:12:00+07:00] [Muscle:Antigravity]` Xóa bỏ tệp helper cũ `internal/handler/orchestration/activity_logger.go` hoàn thành mục tiêu Phase 3.
- `[2026-06-22T12:15:00+07:00] [Muscle:Antigravity]` Chạy thành công `go test ./...` và kiểm tra tĩnh `go vet ./...` sạch sẽ 100% không còn lỗi sao chép lock hay lỗi biên dịch test.
- `[2026-06-22T13:15:00+07:00] [Brain:Antigravity]` Thực hiện audit đối chiếu tiến độ thực tế với workspace `02_plan.md` của cả hai workspace, xác nhận tất cả các đầu mục công việc và checklist đã hoàn thành 100%, đồng bộ hóa tài liệu kế hoạch thành công.
- `[2026-06-22T14:35:00+07:00] [Brain:Antigravity]` Phác thảo kế hoạch Phase 5 nhằm giải quyết 7 điểm vi phạm rò rỉ DB còn lại. Cập nhật implementation_plan.md và 02_plan.md, trình User phê duyệt.
- `[2026-06-22T14:40:00+07:00] [Muscle:Antigravity]` Triển khai hoàn thành Phase 5: Xây dựng `ConnectorResolver` service, nâng cấp các Repositories (`TransmuteScheduleRepo`, `MasterBindingRepo`, `MappingRuleV2Repo`, `ReconciliationReportRepo`, `TableRegistryRepo`), thêm các phương thức DDL/DML tương tác shadow table vào `SchemaAdapter`.
- `[2026-06-22T14:45:00+07:00] [Muscle:Antigravity]` Tạo `SourceRegistrationService` đóng gói logic đăng ký nguồn 5 bước (tự xử lý transaction) và refactor các Handlers sử dụng repositories/services mới tiêm vào.
- `[2026-06-22T14:50:00+07:00] [Muscle:Antigravity]` Cập nhật dependency injection tại `worker_server_init.go` và `admin_server_init.go`.
- `[2026-06-22T15:05:00+07:00] [Muscle:Antigravity]` Sửa lỗi compile cycle trong tests của `governance/masking_service_test.go` bằng cách mock `MetadataRegistry` thay cho dependency `source`.
- `[2026-06-22T15:10:00+07:00] [Muscle:Antigravity]` Sửa lỗi compile và test suite cho admin API registration handlers trong `server_test.go` và `registration_test.go`.
- `[2026-06-22T15:15:00+07:00] [Muscle:Antigravity]` Chạy thành công toàn bộ test suite `go test ./...` với tỷ lệ PASS 100% không gặp lỗi compile hay runtime panic.
- `[2026-06-22T15:20:00+07:00] [Brain:Antigravity]` Xác nhận hoàn thành toàn bộ Phase 5 của kế hoạch, cập nhật active_plans.md sang Done. Kết thúc phiên làm việc thành công.
- `[2026-06-22T15:30:00+07:00] [Brain:Antigravity]` Mở lại workspace, bắt đầu Phase 6 để khắc phục 4 điểm rò rỉ cơ sở dữ liệu thô phát hiện trong quá trình audit. Cập nhật implementation_plan.md và 02_plan.md.
- `[2026-06-22T15:35:00+07:00] [Muscle:Antigravity]` Bổ sung các phương thức helper DDL & DML mới (`DeleteRecords`, `CreateEmptyTable`, `AddPrimaryKeyColumn`, `CheckPrimaryKeyExists`, `AddPrimaryKeyConstraint`, `EnsureCDCColumnsInSchema`, `TableExists`) vào `SchemaAdapter` (`internal/service/shadow/schema_adapter.go`).
- `[2026-06-22T15:40:00+07:00] [Muscle:Antigravity]` Refactor `ReconHealer` (`internal/service/recon/recon_heal.go`) để tiêm `ConnectorResolver` thay vì `db *gorm.DB` thô. Sửa đổi `healAuditBatcher` lấy DB gián tiếp qua `ActivityLogger.GetDB()`.
- `[2026-06-22T15:45:00+07:00] [Muscle:Antigravity]` Refactor `SchemaDDLHandler` (`internal/handler/shadow/schema_ddl_handler.go`) để tiêm `DiscoverService` thay vì `shadowDB *gorm.DB` thô, chuyển tất cả các lệnh DDL/DML sang `SchemaAdapter` và uỷ thác đồng bộ rules v2.
- `[2026-06-22T15:48:00+07:00] [Muscle:Antigravity]` Xóa bỏ tệp dead code `internal/handler/shadow/schema_ddl_handler_schema.go`.
- `[2026-06-22T15:52:00+07:00] [Muscle:Antigravity]` Cập nhật cấu hình dependency injection tại `internal/server/worker_server_init.go` cho `schemaDDLHandler` và `reconHealerShared`.
- `[2026-06-22T15:55:00+07:00] [Muscle:Antigravity]` Chạy thành công kiểm thử tự động `go build ./...` và `go test ./...` PASS 100%. Thực hiện Security Gate hoàn thành kiểm tra an toàn.
- `[2026-06-22T16:00:00+07:00] [Brain:Antigravity]` Cập nhật walkthrough.md và task.md, xác nhận hoàn thành Phase 6 và toàn bộ chiến dịch Decouple Database from Handlers.



