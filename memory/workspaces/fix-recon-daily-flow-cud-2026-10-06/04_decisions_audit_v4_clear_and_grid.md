# 04 — ADR Audit V4

## ADR-V4-1: Định danh pipeline được ghi tại writer, không suy ở reader
- **Quyết định:** Thêm key additive `shadow_schema` (log Shadow) và `master_binding_id` (log transmute) vào `details`; reader ưu tiên key khi có.
- **Lý do:** Reader không thể khôi phục định danh không tồn tại trong bản ghi. `shadow_binding_id` là key lý tưởng (reader đã hỗ trợ) nhưng `UpsertRecord` không mang binding id → phải plumbing qua routing registry (impact lớn). `schemaName` đã có sẵn trong scope.
- **Laya V4:** DISAGREE 74.65% (không kèm lý do). Phản biện: giải pháp thay thế (chỉ lọc ở reader) đã được chứng minh vô hiệu bằng code (L38-48); giữ quyết định, ghi nhận rủi ro: log legacy vẫn mơ hồ.

## ADR-V4-2: Clear = "xóa đúng cái UI thấy"
- Dùng chung `enrichmentFromClause`; scope bắt buộc; bỏ dòng mơ hồ; mốc RFC3339; bỏ from/to.
- **Laya V4:** DISAGREE 54.07% (gần ngẫu nhiên, confidence 0.005). Rủi ro đã biết: pipeline thiếu `shadow_schema` không Clear được qua UI; binding inactive không Clear được → retention dọn.

## ADR-V4-3: `DELETE /activity-log` qua `destructiveChain` — Laya AGREE 59.29%.
## ADR-V4-4: Dấu snapshot truyền qua trigger + group debounce theo (binding, trigger) tuần tự — Laya AGREE 59.64%.
## ADR-V4-5: Handler gọi repo trực tiếp qua `ActivityLogCleaner` (mirror `DeleteReport`) — Laya AGREE 62.85%.
## ADR-V4-6: Tổng hợp ngày phía server (P2) — bắt buộc cho tính đúng khi > 200 batch.
