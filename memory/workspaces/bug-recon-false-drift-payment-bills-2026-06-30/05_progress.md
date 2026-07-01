# Progress: Fix False Drift on Recon payment_bills

## Governance Audit / Kiểm toán Quản trị
- **Root Cause Analysis (RCA) on Governance Violation / Phân tích nguyên nhân gốc rễ vi phạm Quản trị**:
  - **Sự kiện vi phạm**: Khi bắt đầu task, thư mục workspace chỉ được khởi tạo 3 file (`00_context.md`, `02_plan.md`, `05_progress.md`) thay vì khởi tạo đầy đủ 8 file cấu trúc chuẩn V3 Workspace Standard.
  - **Nguyên nhân gốc rễ**: Model ưu tiên tập trung sửa lỗi logic nghiệp vụ của code nhanh chóng, dẫn đến việc thiếu sót bước khởi tạo đầy đủ khung tài liệu chuẩn của workspace như đã quy định trong `conventions.md`.
  - **Biện pháp khắc phục**: Khởi tạo ngay lập tức các file còn thiếu (`01_requirements.md`, `03_implementation.md`, `04_decisions.md`, `06_validation.md`, `07_lessons.md`, `08_tasks.md`, `report_recon_false_drift.md`) để đưa workspace về trạng thái tuân thủ 100% quy trình quản trị dự án.

## Progress Log
| [Timestamp] | [Agent/Model] | [Trạng thái: TODO/DOING/DONE] | [Thời gian thực hiện] | [Hành động] |
| :--- | :--- | :--- | :--- | :--- |
| `2026-06-30T03:10:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 5m | Khởi tạo workspace `bug-recon-false-drift-payment-bills-2026-06-30` và xác định Root Cause. |
| `2026-06-30T03:15:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 10m | Cập nhật signatures và query/hash logic trong `recon_dest_query.go` và `recon_dest_hash.go` để hỗ trợ dynamic timestamp field. |
| `2026-06-30T03:17:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 5m | Cập nhật logic gọi destAgent ở Tier 1 (`recon_tier_a.go`) để truyền `resolvedTS` từ registry config. |
| `2026-06-30T03:18:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 5m | Cập nhật logic đối soát Tier 2 (`recon_tier_b.go`) truyền `_source_ts` để giữ nguyên so sánh CDC stream time. |
| `2026-06-30T03:19:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 10m | Sửa đổi legacy wrapper `recon_dest_legacy.go` và tạo `recon_dest_agent_test.go` với các unit test case cho dynamic timestamp. |
| `2026-06-30T03:20:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 5m | Chạy compile `go build` và pass thành công 100% unit tests. |
| `2026-06-30T20:25:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 15m | Tiến hành audit quản trị dự án, bổ sung đầy đủ các tài liệu workspace còn thiếu (`01_requirements.md`, `03_implementation.md`, `04_decisions.md`, `06_validation.md`, `07_lessons.md`, `08_tasks.md`, `report_recon_false_drift.md`). |
| `2026-06-30T20:30:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 15m | Nâng cấp MaxWindowTs hỗ trợ dynamic timestampField, cập nhật các cuộc gọi liên quan và pass 100% unit tests. |
| `2026-06-30T20:40:00Z` | `[Antigravity:Gemini-3.5-Flash-High-Unverified]` | DONE | 10m | Tiến hành audit chéo code và tài liệu `06_validation.md`. Phát hiện thiếu 4 test cases mặc định. Đã viết bổ sung `TestDestAgent_BucketCounts_Default`, `TestDestAgent_ListIDTsInWindow_Default`, `TestDestAgent_HashWindow_Default`, và `TestDestAgent_BucketHash_Default` vào `recon_dest_agent_test.go` và chạy pass 100%. |

## Root Cause Analysis (Technical)
- **Hiện tượng**: Bảng `payment_bills` báo chênh lệch ảo 1.410 bản ghi mặc dù thực tế dữ liệu giữa Mongo và Postgres hoàn toàn khớp nhau.
- **Root Cause**: Destination Agent của đối soát trước đây fix cứng sử dụng metadata CDC timestamp `_source_ts` để lọc và hash, trong khi Mongo Source sử dụng domain timestamp `lastUpdatedAt`. Khi Debezium chạy snapshot/backfill lại gần đây, cột `_source_ts` bị cập nhật thành ngày mới, trong khi domain timestamp vẫn ở ngày cũ. Việc lệch hệ quy chiếu lọc cửa sổ này gây ra drift ảo.
- **Bài học & Biện pháp khắc phục**: Đối soát chéo giữa hai hệ cơ sở dữ liệu khác nhau (Mongo vs Postgres) bắt buộc phải sử dụng cùng một hệ quy chiếu cột thời gian (domain timestamp). Chỉ đối soát Tier 2 (Shadow vs Master - cùng là Postgres và cùng CDC stream) mới nên sử dụng `_source_ts`.
