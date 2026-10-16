# Báo Cáo Nghiệm Thu: Sửa (Edit), Xoá (Delete), Danh Mục Tác Vụ và Tái Cấu Trúc Cascading Selector 5 Tầng trên Activity Manager

---

## 1. TỔNG QUAN THAY ĐỔI
Theo yêu cầu chấn chỉnh nghiêm khắc từ User và phê duyệt `APPROVE`, Role **Muscle (Chief Engineer)** đã hoàn tất triển khai toàn diện trên cả Backend CMS và Frontend Web:
1. **Bổ sung tính năng Sửa (Edit) lịch trình:** Modal Edit cho phép chỉnh sửa toàn bộ các thông số (`params`, `interval_minutes`, `run_at_time`, `target_table`, `notes`, `is_enabled`), hỗ trợ cả `source_count`, `confluence`, `reconcile`, `bridge`...
2. **Bổ sung tính năng Xoá (Delete) lịch trình:** Popconfirm xác nhận an toàn, gọi API `DELETE /api/worker-schedule/:id` và ghi audit log `cdc_system.activity_logs`.
3. **Chuẩn hóa Danh mục Tác vụ (`ALL_OPERATIONS`):**
   - Loại bỏ: `drop-gin-index` và `create-default-columns`.
   - Bổ sung: `reconcile` (Đối soát dữ liệu) và `bridge` (Đồng bộ cầu nối).
4. **Tái cấu trúc triệt để Bộ chọn Cascading Selector 5 tầng tự động theo huyết thống metadata:**
   - Tầng 1: `layer_type` (`source`, `shadow`, `master`).
   - Tầng 2: `connection_code` lọc theo `layer_type`; tự động gán & khóa `engine_type`; tự động gán `object_type` (`mongodb` -> `collection`, SQL -> `table`, `sftp` -> `file_batch`).
   - Tầng 3: `container_namespace` dạng `<Select showSearch>` lấy dữ liệu thực tế từ `source_db`, `shadow_schema`, `master_schema` (loại bỏ ô Input tự gõ lung tung).
   - Tầng 4: `object_name` dạng `<Select showSearch>` lọc danh sách table / collection theo đúng Namespace.
   - Tầng 5: `date_field` tự động điền `timestamp_field` từ metadata của đối tượng đã chọn.

---

## 2. DANH SÁCH TẬP TIN ĐÃ THAY ĐỔI & DÒNG CODE

| Repository | File Path | Thao tác | Số dòng thay đổi | Mô tả tóm tắt |
| :--- | :--- | :---: | :---: | :--- |
| `cdc-cms-service` | `internal/app/ports/repository.go` | Edit | +1 | Thêm `Delete(ctx context.Context, id int64) error` vào `WorkerScheduleRepo` |
| `cdc-cms-service` | `internal/infra/persistence/scheduler/worker_schedule_repo_gorm.go` | Edit | +11 | Hiện thực `Delete(ctx, id)` bằng GORM |
| `cdc-cms-service` | `internal/app/commands/scheduler/update_schedule.go` | Edit | +7 | Bổ sung `TargetTable *string` vào `UpdateScheduleCommand` |
| `cdc-cms-service` | `internal/app/commands/scheduler/delete_worker_schedule.go` | Create | +78 | CQRS Command `DeleteWorkerScheduleCommand` & Handler |
| `cdc-cms-service` | `internal/app/commands/scheduler/worker_schedule_commands_test.go` | Create | +30 | Unit test cho validation Delete & Update Command |
| `cdc-cms-service` | `internal/api/scheduler/schedule_handler.go` | Edit | +44 | Bổ sung `TargetTable` vào Update & thêm handler `Delete(c *fiber.Ctx)` |
| `cdc-cms-service` | `internal/server/server.go` | Edit | +1 | Đăng ký `worker-schedule.delete` vào Command Bus |
| `cdc-cms-service` | `internal/router/router.go` | Edit | +1 | Đăng ký route `dual("DELETE", admin, "/worker-schedule/:id", ...)` |
| `cdc-cms-web` | `src/pages/ActivityManager.tsx` | Rewrite | ~680 | Chuẩn hóa `ALL_OPERATIONS`, thêm nút Sửa & Xoá, thêm Modal Edit, tách component `CascadingSourceSelector` 5 tầng |

---

## 3. KIỂM ĐỊNH CHẤT LƯỢNG (QUALITY GATES G1–G8)
- **G1 (Truy vết Yêu cầu):** 100% yêu cầu được đáp ứng không sót mục nào.
- **G3 (Build & Test thật):**
  - Backend `cdc-cms-service`: `go build ./cmd/server` Exit 0.
  - Backend Unit tests: `go test -v ./internal/app/commands/scheduler/...` và `go test -v ./internal/api/scheduler/...` PASS 100%.
  - Frontend `cdc-cms-web`: `npm run build` (`tsc -b && vite build`) Exit 0.
  - Core Worker `centralized-data-service`: `go build ./cmd/worker` Exit 0.
- **G4 (Edge Cases & Safety):**
  - Xoá lịch trình có bọc `Popconfirm` xác nhận của Quản trị viên.
  - Sửa token Confluence: Nếu người dùng để trống ô token trong modal sửa thì hệ thống tự động giữ nguyên token cũ đã mã hóa, không bị ghi đè chuỗi rỗng.
- **G8 (Bằng chứng vật lý trong Workspace):** Đầy đủ 5 tài liệu trong workspace `fix-recon-daily-flow-cud-2026-10-06`.
