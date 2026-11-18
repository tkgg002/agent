---
tier: T2
mode: standard
suffix: scan_field_type
workspace: audit-scan-field-embeddata-type-2026-11-16
created_at: 2026-11-16T15:09:00+07:00
---
# 01 Requirements — Scan Field suy sai kiểu `embedData` (object) thành `TEXT`

## 1. Yêu cầu User
- Field Mongo `embedData = { promotionInfo: { campaignCode: 'blackfriday' } }` (object) nhưng "Scan Fields" sinh mapping rule `data_type = TEXT`.
- Tìm đúng luồng scan đang chạy, chỉ ra vì sao ra TEXT (file:line + data thật), đề xuất 1 fix (Propose-Only).

## 2. Ràng buộc (từ GEMINI.md + lessons)
- Propose-Only: không sửa source trước APPROVE.
- Không sửa data trực tiếp trong DB (chỉ SELECT read-only).
- Không hardcode tên field/bảng (`embedData`) vào engine scan — fix phải tổng quát.
- Không đụng phạm vi workspace `fix-shadow-sync-fields-alter-type-2026-11-16` (ALTER TEXT→JSONB do agent khác xử lý).
