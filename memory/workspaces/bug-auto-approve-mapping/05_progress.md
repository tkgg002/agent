# 05_progress — Audit Log (Append Only)

- [2026-08-27 10:00] [Brain:Gemini-3.7-Flash] Khởi tạo phân tích nguyên nhân gốc rễ bug auto-approve chéo.
- [2026-08-27 10:05] [Muscle:Gemini-3.7-Flash] Phát hiện bug #1 trong `source_repo_gorm.go:359-364`: UPDATE mapping_rule_v2 chỉ WHERE theo `source_object_id`, thiếu `shadow_binding_id`.
- [2026-08-27 10:10] [Muscle:Gemini-3.7-Flash] Phát hiện bug #2 trong `MappingFieldsPage.tsx:232`: ternary copy-paste `rule.is_active ? 'approved' : 'approved'`.
- [2026-08-27 10:30] [Muscle:Gemini-3.7-Flash] Thử nghiệm sửa `source_repo_gorm.go` lần 1: query `shadow_binding.legacy_registry_id` (SAI do column không tồn tại).
- [2026-08-27 11:00] [Muscle:Gemini-3.7-Flash] Thử nghiệm sửa `source_repo_gorm.go` lần 2: query `so.legacy_registry_id` (SAI do column không tồn tại trên source_object_registry).
- [2026-08-27 11:20] [Muscle:Gemini-3.7-Flash] Thử nghiệm sửa `source_repo_gorm.go` lần 3: query `source_locator_json->>'legacy_registry_id'`.
- [2026-08-27 13:18] [Brain:Gemini-3.7-Flash] Thực hiện QC phản biện toàn diện:
  - Phân tích quan hệ 1-N giữa `source_object_registry` và `shadow_binding`.
  - Phát hiện `source_locator_json` bị ghi đè khi có nhiều table registry trỏ cùng source table (`payment_bills` vs `payment_bills_1`).
  - Tái cấu trúc query `shadow_binding` sang dạng chuẩn mực: `WHERE source_object_id = ? AND shadow_table = ? AND is_active = true`.
  - Bổ sung cập nhật `status` đồng thời trong optimistic update của frontend `MappingFieldsPage.tsx`.
- [2026-08-27 13:20] [Muscle:Gemini-3.7-Flash] Build verify `cdc-cms-service` (PASS) và `cdc-cms-web` (PASS).
- [2026-08-27 13:22] [QA:Gemini-3.7-Flash] Xuất báo cáo QC gắt gao vào `audit_report_20260827_final.md`.
