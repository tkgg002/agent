# Plan: Decoupling Database Access from Handlers in centralized-data-service

## Goal
Thực hiện toàn bộ kế hoạch refactor kiến trúc theo báo cáo audit đã duyệt, chuẩn hóa domain folder, giải quyết rò rỉ thao tác DB tại Handler và dọn dẹp helper sai tầng.

## Checklist

### Phase 1: Chuẩn hóa Miền nghiệp vụ (Domain Alignment)
- [x] Di chuyển model `reconciliation_report.go` và `snapshot_dlq.go` sang `internal/model/recon/` và đổi package sang `recon`.
- [x] Di chuyển `provisioning_orchestrator.go` sang `internal/service/orchestration/` và đổi package sang `orchestration`.
- [x] Đổi tên `registry_repo.go` thành `table_registry_repo.go` và đổi tên struct thành `TableRegistryRepo`.
- [x] Cập nhật toàn bộ imports của model, service, repo mới di chuyển và biên dịch lại.

### Phase 2: Decouple database khỏi Handlers
- [x] Refactor `ScanHandler` để ủy thác quét field qua `ScanService`.
- [x] Refactor `DiscoverHandler` để sử dụng `TableRegistryRepo` và `SourceObjectRegistryRepo` thay vì gọi `h.DB` trực tiếp.
- [x] Refactor `ReconHandler` và `BatchBuffer` để sử dụng `ActivityLogger` Service và `FailedSyncLogRepo`.
- [x] Refactor `DLQHandler` để sử dụng `FailedSyncLogRepo` cho các tác vụ tạo và cập nhật log lỗi thay vì thao tác trực tiếp qua GORM.

### Phase 3: Loại bỏ các Helper sai tầng
- [x] Thay thế các lệnh gọi helper ghi activity cũ bằng `ActivityLogger` Service.
- [x] Xóa bỏ file helper cũ `internal/handler/orchestration/activity_logger.go`.

### Phase 4: Verification & Test
- [x] Chạy `go build ./...` để verify code build thành công.
- [x] Chạy `go test ./...` để verify tất cả các bài test vượt qua.

### Phase 5: Decouple Database access - Giai đoạn 2 (7 điểm vi phạm còn lại)
- [x] Xây dựng `ConnectorResolver` service để đóng gói phân giải connector mà không cần nhận `*gorm.DB` thô.
- [x] Nâng cấp các Repositories (`TransmuteScheduleRepo`, `MasterBindingRepo`, `MappingRuleV2Repo`, `ReconciliationReportRepo`, `TableRegistryRepo`) để che giấu các SQL query/command thô.
- [x] Thêm các phương thức DDL/DML tương tác shadow table vào `SchemaAdapter`.
- [x] Tạo `SourceRegistrationService` để đóng gói logic đăng ký nguồn 5 bước (tự xử lý transaction).
- [x] Refactor các Handlers (`ShadowBindHandler`, `SchemaDDLHandler`, `ScheduleEnableHandler`, `TransmuteHandler`, `RegisterHandler`, `SyncHandler`, các Recon handlers) sử dụng các repositories/services mới tiêm vào.
- [x] Cập nhật dependency injection tại `worker_server_init.go` và `admin_server_init.go`.
- [x] Chạy kiểm thử toàn bộ dự án (`go build` & `go test`).

### Phase 6: Khắc phục 4 điểm rò rỉ database thô (Audit Gap Fix)
- [x] Bổ sung các methods DDL & DML mới vào `SchemaAdapter`
- [x] Refactor `ReconHealer` loại bỏ trường DB thô và sử dụng `ConnectorResolver` + `SchemaAdapter`
- [x] Refactor `SchemaDDLHandler` loại bỏ `shadowDB` thô, dời logic sang `SchemaAdapter` và tích hợp `DiscoverService`
- [x] Xóa bỏ file helper DDL cũ `schema_ddl_handler_schema.go`
- [x] Cập nhật dependency injection tại `worker_server_init.go`
- [x] Verification & Test (Chạy build và test pass 100%)


