# Workspace Context: feat-decouple-handlers-db-2026-06-22

## Overview
Workspace này chịu trách nhiệm thực thi chiến dịch refactoring nhằm tách biệt (decouple) toàn bộ các thao tác trực tiếp với cơ sở dữ liệu (GORM, raw SQL) ra khỏi tầng Handler trong hệ thống `centralized-data-service`, chuyển giao trách nhiệm truy xuất dữ liệu cho Repository Layer và nghiệp vụ cho Service Layer.

## Scope
- **Phase 1: Chuẩn hóa Miền nghiệp vụ (Domain Alignment)**
  - Di chuyển model `reconciliation_report.go` và `snapshot_dlq.go` về đúng miền `internal/model/recon/`.
  - Di chuyển `provisioning_orchestrator.go` về đúng miền `internal/service/orchestration/`.
  - Đổi tên `registry_repo.go` thành `table_registry_repo.go` để khớp 1-1 với model.
- **Phase 2: Decouple database khỏi Handlers**
  - Refactor `ScanHandler` để sử dụng `ScanService`.
  - Refactor `DiscoverHandler` để sử dụng `TableRegistryRepo` và `SourceObjectRegistryRepo` thay vì gọi trực tiếp `h.DB`.
  - Refactor `ReconHandler`, `BatchBuffer`, `ReconHealer` và `DLQHandler` để sử dụng `ActivityLogger` và `FailedSyncLogRepo`.
- **Phase 3: Loại bỏ các Helper sai tầng**
  - Xóa bỏ helper ghi log cũ `internal/handler/orchestration/activity_logger.go` và đồng bộ qua Service Logger.
- **Phase 4: Verification & Test**
  - Biên dịch dự án và chạy unit test suite để đảm bảo không lỗi hồi quy.

## Technical Context
- Dự án: `centralized-data-service`
- Ngôn ngữ: Go
- Framework: GORM, NATS
