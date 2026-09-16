# 08_tasks_target_connection_mgmt.md - Checklist triển khai Target Connection Management Suite

## Phase 1: CMS Backend (`cdc-cms-service`)
- [x] Task 1.1: Bổ sung DTO request/response trong `internal/api/dto/master_connection_dto.go` (Create, Update, TestConnection).
- [x] Task 1.2: Thêm các method `CreateMasterConnection`, `UpdateMasterConnection`, `DeactivateMasterConnection`, `CheckMasterConnectionUsage` trong `internal/app/ports/repository.go` và cài đặt trong `internal/infra/persistence/master/master_repo_gorm.go`.
- [x] Task 1.3: Cài đặt logic Test Connection (mở kết nối pgx/gorm kiểm tra ping trong 5s) và CRUD handlers trong `internal/api/master/master_registry_handler_connections.go`.
- [x] Task 1.4: Đăng ký các route mới trong `internal/router/router.go`:
  - `POST /v1/master-connections/test`
  - `POST /v1/master-connections`
  - `PUT /v1/master-connections/:id`
  - `DELETE /v1/master-connections/:id`
- [x] Task 1.5: Build `go build ./cmd/server` và verify không có compiler error.

## Phase 2: CMS Web Frontend (`cdc-cms-web`)
- [x] Task 2.1: Tạo component modal/drawer quản lý kết nối đích:
  - Xem danh sách connections kèm tag trạng thái, host, database.
  - Form thêm/sửa connection: mã kết nối, tên hiển thị, host, port, database, schema, username, password, sslmode.
  - Nút `Test Connection` gọi `/api/v1/master-connections/test` hiển thị kết quả trực tiếp (latency / error message).
- [x] Task 2.2: Tích hợp vào `MasterRegistry.tsx`:
  - Thêm nút `[Quản Lý Kết Nối Đích]` trên thanh công cụ của Master Registry.
  - Thêm nút `[+ Thêm kết nối]` (quick add) cạnh dropdown chọn connection trong Modal Tạo Master Table.
- [x] Task 2.3: Build `npm run build` verify TypeScript và Vite compilation sạch sẽ 100%.

## Phase 3: Verification & Governance
- [x] Task 3.1: Kiểm thử toàn trình (End-to-End verification).
- [x] Task 3.2: Append audit log vào `05_progress.md`.
