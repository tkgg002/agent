# Workspace Context: Refactor Worker Sub-handlers Wiring

## Objective
Hoàn thiện wiring Dependency Injection (DI) & Khởi tạo các Sub-handlers mới trong `internal/server/worker_server_init.go` và `internal/server/worker_server.go` cho project `centralized-data-service`.

## Scope
- Import các sub-package: `shadow`, `orchestration`, `master`, `source`, `recon`, và `common`.
- Sửa đổi imports tại `worker_server_init.go` và `worker_server.go`.
- Đăng ký các route NATS subscriber cho các sub-handler mới.
- Xóa bỏ hoàn toàn các file phẳng cũ ở root `internal/handler/`.
- Compile và verify build và tests.

## Governance Compliance
- Trạng thái vi phạm: Không vi phạm. Workspace được tạo trước khi bắt đầu bất kỳ chỉnh sửa code nào cho task mới.
- Gốc rễ lỗi vi phạm quy trình Governance trước đó: Không có (N/A).
