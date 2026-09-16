# 11_report_fix_recon_istimestamptz.md: Báo cáo Thay đổi Mã nguồn

## 1. Overview
- **Mục tiêu:** Khắc phục triệt để điểm đứt gãy trong `ReconJobWorker.HandleJobEvent` bằng cách **gán trực tiếp `event.ShadowSchema` và `event.ShadowTable` từ NATS/API Event Payload vào `entry`**, đảm bảo `entry.QualifiedTarget()` trả về đúng `shadow_schema.shadow_table` (ví dụ `"shadow_traitestctphs.trans_his"`).
- **Dự án tác động:** `centralized-data-service`.

## 2. Danh sách Tệp tin Thay đổi (File Changes Summary)

| STT | File Path | Tác động | Chi tiết thay đổi |
| :--- | :--- | :--- | :--- |
| 1 | `internal/service/recon/recon_job_worker.go` | Modify (+6 lines) | Gán trực tiếp `event.ShadowSchema` và `event.ShadowTable` vào `entry` trước khi gọi `ExecuteSegment`. |
| 2 | `internal/model/source/table_registry.go` | Modify (+3 lines) | Sửa `QualifiedTarget()` tránh lặp schema prefix khi `TargetTable` đã chứa dấu `.`. |

## 3. Tối giản Mã nguồn
- **KHÔNG** viết thêm hàm Khử trùng lặp `deduplicateStaleIDsPayload`.
- **KHÔNG** viết thêm Fallback Query xuyên schema.
- Giữ nguyên 100% logic mã nguồn gốc.
