# 00_context_default_collections — Bối cảnh

> **Workspace**: `feature-connector-default-collections-2026-05-25`
> **Phase**: `default_collections`
> **Date**: 2026-05-25
> **Owner Brain**: Antigravity (Chairman) — Plan only
> **Owner Muscle**: claude-sonnet-4-6 (default) — Sẽ thực thi SAU khi user approve
> **Governance ràng buộc**: CLAUDE.md §0, §1, §3, §7, §11, §12, §14

---

## 1. Bối cảnh nghiệp vụ

User đang dùng UI `cdc-cms-web` để tạo Source Connector kiểu MongoDB. Trên form có field `Collections` (free-text, optional). Behavior **mong đợi**:

> Khi user KHÔNG nhập gì vào field `Collections` → connector phải CDC **toàn bộ collections** của database đó (không filter).

Behavior **hiện tại** (đã audit) — xem `10_gap_analysis_default_collections.md`:

- FE: form field `Collections` không bắt buộc. Nếu để trống → `compactConfig` drop key `collection.include.list` khỏi payload gửi BE.
- BE: handler `system_connectors_handler.go` accept config map as-is, không inject default, không validate field này.
- Kafka Connect / Debezium: KHÔNG nhận key `collection.include.list` → áp dụng default → CDC all collections.

**Kết luận audit**: Pipeline đã hoạt động đúng yêu cầu user **về mặt runtime**.

## 2. Vấn đề thực tế còn lại

Mặc dù runtime đúng, **UX của user không rõ ràng**:

| Layer | Vấn đề UX | Hệ quả |
|---|---|---|
| Form input | Placeholder `users,orders,payments` gợi ý "phải nhập" → user tưởng bỏ trống là invalid hoặc connector sẽ không hoạt động | User nhập bừa hoặc bỏ qua connector |
| Tooltip / helper text | KHÔNG có text giải thích "để trống = CDC tất cả" | User không biết default behavior |
| Display sau khi tạo | Connector list view hiển thị empty / `(none)` cho field collections → user không phân biệt được "connector mới chưa cấu hình" vs "intentional CDC all" | Confusion, suspicion |
| Documentation | Project README / CMS handbook không note rõ default behavior | New onboarding miss |

## 3. Phạm vi scope phase này

| In scope | Out of scope |
|---|---|
| ✅ Cải thiện UX form Collections trong UI tạo/edit connector | ❌ Thay đổi BE handler logic |
| ✅ Cải thiện hiển thị field Collections trong connector list view | ❌ Thay đổi Debezium config defaults |
| ✅ Audit Debezium default behavior trên Mongo connector class | ❌ Migrate dữ liệu connector cũ |
| ✅ Smoke test: tạo connector empty Collections → verify CDC all | ❌ Thêm UI option "Select collections from picker" (phase sau) |
| ✅ Update CLAUDE-level / project doc nếu cần | ❌ Đổi schema DB |

## 4. Constraints (ràng buộc)

1. **§12 Brain Code Prohibition**: Brain TUYỆT ĐỐI KHÔNG sửa source code. Phase này chỉ tạo plan + tasks + solution demo + report. User approve mới Muscle thực thi.
2. **§7 Full Doc Set**: Phải có đủ bộ doc 00..10 với suffix `_default_collections`.
3. **§11 APPEND-ONLY** cho `05_progress.md`.
4. **§3 Verify before Done**: Mọi gate phải có evidence thực tế, không claim "Đã xong" suông.
5. **User directive**: "đảm bảo ko sửa code rồi hẵng chạy tiếp néh" → Phase Brain dừng ở Plan + Doc, KHÔNG đụng .tsx/.go/.sql.
6. **Core systems direction**: KHÔNG cheat database, KHÔNG hack config để pass. Mọi thay đổi phải qua đường chính thống (form schema, component, validator).

## 5. Files liên quan (đã audit, READ-only)

| Path | Mục đích | Note |
|---|---|---|
| `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx:966-969` | Form.Item Collections | Cần thêm `extra` hoặc `tooltip` |
| `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx:131-133` | `compactConfig` | KHÔNG đụng — đang đúng |
| `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx:160-166` | `buildConnectorConfig` | KHÔNG đụng — đang đúng |
| `data-hub/cdc-cms-service/internal/api/system_connectors_handler.go:168-171` | Create handler DTO | KHÔNG đụng — đang đúng |
| (potential) connector list view component | Hiển thị field collections | Cần audit thêm trong M1 |

## 6. Stakeholders

| Vai trò | Người | Vai phase này |
|---|---|---|
| Product / Ops | trainguyen | Quyết định UX wording cuối cùng |
| Brain | Antigravity | Plan + design + verify |
| Muscle | CC CLI | Thực thi sau approve |

## 7. References

- Lessons cross-reference: `agent/memory/global/lessons.md` (L-CDC-golden-rule — read-only Mongo, L-Path-B-pattern, L-cheat-DB-ALTER-in-report — không cheat).
- Tech stack: `agent/memory/global/tech_stack.md` — React + Antd, Go BE, Kafka Connect / Debezium 1.9.x.
- Project context: `agent/memory/global/project_context.md`.
