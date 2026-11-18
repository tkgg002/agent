---
tier: T2
mode: standard
suffix: scan_nested_id
workspace: fix-scan-nested-id-skip-2026-11-16
created_at: 2026-11-16T15:13:30+07:00
---
# 01 Requirements — Scan field flatten object `merchantInfo` mất sub-field `id`

## 1. Yêu cầu User
- Flatten `merchantInfo = {id, merchantCode, name, email, address, type}` → kết quả scan có đủ field trừ `id`.
- Tìm root cause (file:line), đề xuất 1 fix (Propose-Only, chờ APPROVE).

## 2. Technical Constraints
- G2 Propose-Only; không sửa source trước APPROVE.
- Không làm lộ cột hệ thống child-explode (`_parent_source_id`, `_array_index`) và 11 reserved column.
- Top-level skip `id`/`_id` (PK) giữ nguyên — chỉ áp dụng nested.

## 3. Bổ sung (2026-11-16T15:21 — User)
- F2: `cdc_system.mapping_rule_master.source_field` chuyển sang `TEXT` (giá trị Subset/Value(JSON)/Type-Presence dài vượt 255).
