---
tier: T2
suffix: recon_heal_delete_id
workspace: audit-recon-heal-delete-id-impact-2026-10-23
created_at: 2026-10-23T12:00:35+07:00
---

# 01 Requirements: Audit Ảnh Hưởng Của Thay Đổi Delete / `_source_id` / `_gpay_id` Tới Luồng Recon & Heal (Segment A & B, 3 Loại Recon 2h / 7d / 30d)

## 1. Bối cảnh & Triệu chứng thực tế từ User
User cung cấp log kết quả chạy Recon thực tế:
```text
7494	Shadow → Master	reconB	11:51 23/09/2026 - 11:51 23/10/2026	4,536	4,539	1534	0	1537	11:56:35 23/10/2026
```
Và yêu cầu:
> "mày audit lại coi, bữa mày sửa cái vụ delete này kia có update cái source_id hay gpay_id gì đó, nó đang ảnh hưởng tới luồng recon / heal của A và B, và 3 loại recon (2h, 7 ngày, 30 ngày ) của tao."

Phân tích sơ bộ con số từ dòng log `id=7494`:
- Chiều đối soát: `Shadow → Master` (`reconB`)
- Cửa sổ thời gian: `11:51 23/09/2026 - 11:51 23/10/2026` (Recon 30 ngày)
- Số lượng bản ghi Shadow (`source_count`): `4,536`
- Số lượng bản ghi Master (`target_count`): `4,539` (chênh lệch tổng số lượng chỉ 3 bản ghi)
- Nhưng kết quả so khớp tập hợp lại báo:
  - `missing_count`: `1,534`
  - `mismatch_count`: `0`
  - `orphan_count` (hoặc `extra_count`): `1,537`
  - Nhận xét: `4,536 - 1,534 = 3,002` bản ghi khớp, còn `1,534` bản ghi ở Shadow và `1,537` bản ghi ở Master bị lệch định danh so khớp (`1,537 - 1,534 = 3 = 4,539 - 4,536`)!

## 2. Phạm vi Audit & Giải pháp (Scope)
1. Kiểm tra bản ghi `id=7494` trong DB runtime thật (`recon_history` / `cdc_activity_log` / bảng liên quan) để xác định bảng nào đang bị lệch (`1534` missing vs `1537` orphan) và cấu trúc `_gpay_id` / `_source_id` / PK thực tế trên Source, Shadow, Master.
2. Rà soát toàn bộ các commit vừa sửa liên quan đến `delete`, `_source_id`, `_gpay_id`, `conflictTarget` (`30b1b00`, `08c05a8`, `43e5da7` trong `centralized-data-service` và `069c456` trong `cdc-cms-service`).
3. Audit toàn diện ảnh hưởng lên:
   - **Segment A (`Source → Shadow`)**: Recon & Self-Heal.
   - **Segment B (`Shadow → Master`)**: Recon & Self-Heal (PostgreSQL Master & MongoDB Master).
   - **3 loại Recon theo cửa sổ thời gian**:
     - Recon `2h` (Recent/Short window)
     - Recon `7 ngày` (`7d` Medium window)
     - Recon `30 ngày` (`30d` Deep/Full window)
4. Thiết kế **1 phương án tối ưu duy nhất** (`09_tasks_solution_recon_heal_delete_id.md`), kiểm định qua Decision Gate `/laya_router` (`DG1/DG2` + `DG3`), và trình User `APPROVE` (tuân thủ nghiêm ngặt Propose-Only trước khi sửa code).

## 3. Yêu cầu Bổ sung từ User: Chống Cheat (Zero-Cheat / Config-Driven) & Audit Toàn Bộ Diff + 3 Repositories Từng Dòng
User phản bác gay gắt các kiểu sửa "cheat" (đoán mò thứ tự cột fallback và hardcode danh sách `IN`):
> "resolveBusinessPKForMaster: Phân giải khóa chính nghiệp vụ theo thứ tự resolveConflictTarget → binding.SourcePK → "_id" → "id" → "_source_id" cho cả deleteFromMasterBySourceAndGpayIDs và inferLastGpayIDFromMaster => thứ tự con mẹ mày, là sao, tao đã nói là phải detêch ở Spec (JSON) * {"pk":"_id"}, mày bị đần độn à. ListMasterTargetsByShadowIdentity: Chấp nhận @shadow_connection_key IN ('', 'default', 'default_shadow') để tương thích ngược 100%. => là sao nữa. phải có chỗ để lấy thông tin ra chứ in cái đéo gì. mày audit lại hết toàn bộ diff cho tao, data là phải chính xác, thiếu cái gì thì phải lên giải pháp để cònig chứ làm cheat như mày là thua rồi. xong mày đi audit toàn bộ 3 repo từng dòng luôn, để tìm ra chỗ nào cheat, mày làm cho mệt chết moẹ mày luôn đi. chán"

Các yêu cầu bắt buộc:
1. **Tuyệt đối cấm đoán mò khóa chính (`resolveBusinessPKForMaster`, `_id -> id -> _source_id`):**
   - Khóa chính nghiệp vụ của bảng Master bắt buộc và duy nhất phải đọc từ `Spec (JSON) *` (`cdc_system.master_binding.transform_spec`, ví dụ `{"pk": "_id"}` hoặc `{"pk": "transaction_id"}`) thông qua `resolveConflictTarget`.
   - Nếu ở tầng cấu hình (`cdc-cms-service` / `cdc-cms-web`) có chỗ nào cho phép tạo/sửa `master_binding` mà thiếu `"pk"` trong `transform_spec` (`{}`), phải lên giải pháp validate và tự động điền từ `source_object_registry.primary_key_field` ngay tại CMS/UI thay vì đoán mò ở runtime worker.
2. **Tuyệt đối cấm hardcode `IN ('', 'default', 'default_shadow')` hoặc hardcode `"shadow_connection_key": "default"` / `"default_shadow"`:**
   - Phải truy xuất `shadow_connection_key` (`cr.connection_code`), `shadow_schema` (`sb.shadow_schema`), và `shadow_binding_code` (`sb.binding_code`) trực tiếp từ metadata DB (`cdc_system.shadow_binding` JOIN `cdc_system.connection_registry`) tại mọi điểm bắn `cdc.cmd.transmute-shadow`, và `ListMasterTargetsByShadowIdentity` phải đối khớp chính xác với `scr.connection_code` thật.
3. **Audit từng dòng trên toàn bộ `git diff` hiện tại và toàn bộ 3 repositories (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`):**
   - Liệt kê đầy đủ mọi vị trí đang "cheat" (fallback đoán mò PK, hardcode connection key / schema, trộn lẫn PK Master vs Shadow, bỏ qua validate `transform_spec.pk`) kèm `file:line`, đoạn code thực tế, hậu quả và giải pháp chuẩn hóa theo Config/Metadata.
