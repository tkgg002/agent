# Danh sách Task chi tiết - Quản lý Index qua CMS UI (Index Manager)

## Task: Xây dựng Index Manager UI & Backend
- **Phase**: GĐ0
- **Service Group**: Core System
- **Service(s)**: centralized-data-service, cdc-cms-service, cdc-cms-web
- **Trạng thái**: [x] DONE

### [Definition of Done]
- [x] Task 1: [Worker] Triển khai 3 NATS Handlers (`cdc.cmd.introspect-indexes`, `cdc.cmd.create-index`, `cdc.cmd.drop-index`) trong `centralized-data-service`.
- [x] Task 2: [CMS API] Triển khai 3 REST API proxy endpoints trong `cdc-cms-service` (`introspection_handler.go` và `router.go`).
- [x] Task 3: [FE] Xây dựng component `TableIndexManager.tsx` trong `cdc-cms-web` và tích hợp vào 2 trang mapping.
- [x] **[QA Gate]**: Chạy test suite của dự án, biên dịch build và kiểm tra UI hoạt động đúng.
- [x] **[Security Gate]**: Chạy security-agent (hoặc tự rà soát) để đảm bảo không rò rỉ secret hoặc SQL Injection.
- [x] Model Tracking: Ghi nhận task vào `05_progress_index_manager.md`.
