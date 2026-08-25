# 14_walkthrough.md — Hướng dẫn & Tổng kết triển khai

## 1. Tóm tắt kết quả triển khai (Implementation Summary)

Chúng ta đã hoàn thành xuất sắc việc khắc phục sự cố Snapshot và tối ưu hóa hệ thống:
1. **Tính năng điều tiết tốc độ `snapshot_max_rps` (Rate Limiting & Disk Protection)**:
   - Cho phép người dùng trực tiếp cấu hình giới hạn RPS trên Modal "Chỉnh sửa Source Object" của trang `TableRegistry`.
   - Backend `cdc-cms-service` tiếp nhận, validate chuẩn (`10 <= rps <= 100,000` hoặc `0` để clear về `NULL`), lưu vào `cdc_system.source_object_registry`.
   - Worker `centralized-data-service` tự động áp dụng `time.Sleep` giữa các batch, ngăn không cho PostgreSQL bị quá tải I/O 95% và Forced Checkpoint.
2. **Khắc phục sự cố Trace ID & Kế thừa Parent Trace khi Resume**:
   - Khi bấm Resume, CMS truyền lại `trace_id` gốc sang Worker.
   - Worker cập nhật `trace_id` mới nhất và xóa thông báo lỗi đỏ (`error_msg = NULL`) trong bảng `cdc_system.snapshot_progress`.
   - Giao diện `SnapshotMonitor` cập nhật ngay Trace ID hoạt động, cho phép click để copy và theo dõi trực tiếp trên SigNoz.

---

## 2. Danh sách 8 file mã nguồn đã chỉnh sửa

| Service | File | Nội dung thay đổi |
| :--- | :--- | :--- |
| `cdc-cms-service` | `internal/app/queries/source/source_objects_read_models.go` | Thêm field `SnapshotMaxRPS *int` vào `SourceObjectListItem`. |
| `cdc-cms-service` | `internal/infra/persistence/source/source_object_read_repo_gorm.go` | Thêm `so.snapshot_max_rps` vào câu SQL SELECT trong `ListEnriched`. |
| `cdc-cms-service` | `internal/app/commands/source/update_source_object_v2.go` | Thêm field vào struct, error `ErrSourceObjectInvalidMaxRPS`, validation `[10, 100000]`, và xử lý map `0` -> `NULL`. |
| `cdc-cms-service` | `internal/api/source/source_object_actions_handler.go` | Tiếp nhận `snapshot_max_rps` từ Fiber request body, ánh xạ lỗi sang HTTP 400. |
| `cdc-cms-service` | `internal/api/scheduler/snapshot_progress_handler.go` | Truy vấn `trace_id` gốc và truyền trong NATS payload khi Resume. |
| `cdc-cms-web` | `src/types/index.ts` | Thêm `snapshot_max_rps?: number | null;` vào `SourceObjectRow`. |
| `cdc-cms-web` | `src/pages/TableRegistry.tsx` | Thêm `'snapshot_max_rps'` vào `V2_EXCLUSIVE_FIELDS`, `openEdit`, `handleEdit`, và InputNumber UI Form. |
| `centralized-data-service` | `internal/handler/orchestration/snapshot_runner_state.go` | Cập nhật `trace_id = ?, error_msg = NULL` trong `claimProgress` khi Resume. |

---

## 3. Hướng dẫn vận hành cho Operator (Operational Runbook)

1. Mở giao diện CMS tại `http://localhost:5173/shadow` (hoặc `TableRegistry`).
2. Tìm dòng bảng `bank_requests` (thuộc database `banvietbank-connector-service` / `bvb-connector-service`).
3. Bấm nút **"Sửa Source"**:
   - Nhập ô **Snapshot Max RPS (snapshot.v2)** = `1500` (hoặc `2000`).
   - Bấm **OK / Lưu**.
4. Mở trang `http://localhost:5173/snapshot-monitor`:
   - Tìm tiến trình snapshot của `bank_requests` (đang ở trạng thái `error` hoặc `paused` tại mốc 5,125,000 / 12,614,888).
   - Bấm nút **"Resume"**.
5. **Kết quả kỳ vọng**:
   - Tiến trình chuyển sang trạng thái `running`.
   - Cột **Trace ID** cập nhật ngay mã trace mới nhất (có thể click để copy tra cứu trên SigNoz).
   - Worker tự động nạp tiếp từ dòng `5,125,001` tới `12,614,888` mà không làm nghẽn đĩa PostgreSQL (I/O duy trì ổn định < 35%).
