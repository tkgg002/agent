# Progress: Snapshot V2 Control Plane

| Bước | Mô tả | Trạng thái | Commit / File |
|------|-------|------------|---------------|
| 1 | Khởi tạo workspace `feature-snapshot-v2-control-plane-2026-05-25` | ✅ Done | `00_context.md`, `05_progress.md` |
| 2 | Research & Lập kế hoạch chi tiết cho các Module (Flow Control, Resiliency, Fail-Safe, LWW Guard) | ✅ Done | `implementation_plan.md`, `02_plan.md` |
| 3 | Thực thi (được gán cho Muscle) | ✅ Done | `runSnapshot` implementation |

## Timeline & Hành động
- **[2026-05-25] [Brain:Antigravity]**: Khởi tạo workspace theo quy tắc 7. Chuyển sang giai đoạn research `SnapshotRunner` handler và LWW Guard tại `upsert.go`. 
- **[2026-05-25] [Brain:Antigravity]**: Cập nhật bản kế hoạch thiết kế dựa trên phản biện của User. Đã chốt cơ chế NATS Event-Driven Pause, PostgreSQL DLQ Storage, chế độ Strict/Lenient theo Object, hàm `EstimatedDocumentCount()` và LWW Guard đầy đủ. Kế hoạch đã hoàn thành, sẵn sàng chuyển cho Muscle thực thi.
- **[2026-05-25] [Muscle:CC CLI]**: Thực thi triển khai DDL, GORM model, tích hợp LWW Guard và NATS Control Plane vào logic `runSnapshot`. Đã build và test thành công.
- **[2026-05-25] [Brain:Antigravity]**: Khắc phục lỗi thiếu sót execution scope (ghi lesson). Hoàn thiện API `/api/v1/snapshot-progress/:id/pause|resume` tại CMS Service và giao diện Snapshot Monitor tại CMS Web (Progress Bar, Total Rows, Pause/Resume Buttons). Build thành công cả hai repo.
- **[2026-05-25] [Muscle:CC CLI]**: Fix lỗi SQL 42P01 khi mapping GORM (sai tên bảng `cdc_source_objects` thành `source_object_registry` và các trường `source_database`, `source_object_name`). Đã xác nhận không còn lỗi crash.
- **[2026-05-25] [Brain:Antigravity]**: Phát hiện thiếu tính năng tuỳ chọn chạy đè (Overwrite) vs chạy tiếp (Resume) của quá trình snapshot. Đã lập kế hoạch bổ sung.
- **[2026-05-25] [Muscle:CC CLI]**: Thực thi bổ sung luồng Overwrite/Resume từ Frontend (truyền tham số qua Modal Dialog) -> Backend (Parse flag) -> Worker (Cập nhật logic cấp khoá `claimProgress` trong DB để reset hoặc giữ lại tiến trình). Đã build pass 3 service.
- **[2026-05-25] [Muscle:CC CLI]**: Fix UI issues: Add `scroll={{ x: 1400 }}` to Table in `SnapshotMonitor.tsx` to prevent layout break. Add Snapshot Monitor to Sidebar in `App.tsx`. Fix `ActivityLog` linking to Snapshot Monitor by using `source_object_id`. Add `source_object_id` parsing to `SnapshotMonitor.tsx` and CMS API.
- **[2026-05-25] [Muscle:CC CLI]**: Gom "Priority" và "Trạng thái table" vào Modal Sửa (Edit) thay vì hiển thị trực tiếp trên Table. Comment lại action "Tạo Field MĐ" theo yêu cầu.
- **[2026-05-25] [Muscle:CC CLI]**: Fix lỗi "Snapshot mất tác dụng": Cập nhật `claimProgress` để khi user chọn "Overwrite" thì bỏ qua logic Zombie-check (chặn các snapshot đang chạy dưới 10 phút), cho phép force-overwrite tiến trình bị kẹt.
