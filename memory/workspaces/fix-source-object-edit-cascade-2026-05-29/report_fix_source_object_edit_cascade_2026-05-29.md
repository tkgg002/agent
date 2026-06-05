# Report — Fix Source Object edit "nhảy chéo binding"

- **Workspace**: `fix-source-object-edit-cascade-2026-05-29`
- **Date**: 2026-05-29
- **Severity**: P1 — UX bug, không lỗi runtime, nhưng gây nhầm lẫn cascade is_active.
- **Service**: `cdc-cms-web` (FE)
- **Trigger**: User báo "sửa 1 cái thì cái còn lại cũng nhảy theo" trên trang Source Object cho entity `wallet-capsets`.

---

## 1. Field + table trả lời cho user

| Layer | Mapping |
|---|---|
| FE Form | `Form.Item name="is_active"` (TableRegistry.tsx:1090) |
| FE label | "Kích hoạt debezium Sync" |
| API contract | `PATCH /api/v1/source-objects/:id` body `{is_active: bool}` |
| BE Go cmd | `IsActive *bool` (update_source_object_v2.go:21) |
| **DB table chính** | `cdc_system.source_object_registry.is_active` |
| **DB cascade** | `cdc_system.shadow_binding.is_active` (cùng `source_object_id`) |
| DB legacy bridge | `cdc_system.cdc_table_registry.is_active` |

## 2. Root cause bug "nhảy chéo"

1. Edit Modal mở per-binding row (`shadow_binding_id` cụ thể).
2. Submit payload 7 field (is_active, priority, notes, timestamp_field, primary_key_field, primary_key_type, snapshot_batch_size — `TableRegistry.tsx:550-558`).
3. `updateEntry` (L421-479) check `togglesBindingOnly = restKeys.length === 1 && restKeys[0] === 'is_active'` → false (7 key) → route vào V2 endpoint `PATCH /source-objects/:sourceObjectId`.
4. BE handler `update_source_object_v2.go:148-161`:
```go
if cmd.IsActive != nil {
    h.db.Table("cdc_system.shadow_binding").
        Where("source_object_id = ?", cmd.ID).   // cascade ALL bindings
        Updates(shadowUpdates)
}
```
→ 1 cú flip Switch ở Modal → tất cả binding cùng source flip theo.

**Đánh giá**: cascade BE là đúng semantic source-level. Vấn đề ở UI: hiển thị Edit per-binding tạo ấn tượng "edit từng binding" trong khi thực ra là edit source.

## 3. Fix (user direction): FE-only

Approach: giữ semantic source-level + làm UI rõ ràng.

| # | Patch | Vị trí | Tác dụng |
|---|-------|--------|----------|
| 1 | useMemo `firstRowIndexBySource` | sau state actionLoadingId | Tracking row đầu tiên mỗi source_object_id |
| 2 | Edit conditional render | column Thao tác | Edit button chỉ hiện 1 lần/source |
| 3 | Snapshot + Manage Masters `disabled` | column Thao tác | Off khi source.is_active = false + Tooltip hint |
| 4 | Quét field `disabled` + effectiveActive | AsyncRowActions | Off khi binding.is_active = false + Tooltip hint + Tag |

## 4. Files modified

| # | File | LOC delta | Loại |
|---|------|-----------|------|
| 1 | `cdc-cms-web/src/pages/TableRegistry.tsx` | +52 / -25 (NET +27) | FE patch |

## 5. Verify evidence

| Item | Result |
|---|---|
| `cd cdc-cms-web && npm run build` | PASS — `built in 742ms` |
| TypeScript check (`tsc -b`) | Silent, no error |
| Bundle `TableRegistry-*.js` | 24.38 kB / gzip 8.07 kB (healthy) |
| Manual smoke (user duty) | ⏳ pending — user verify trên `npm run dev` |

## 6. UX hành vi mới

- Source `wallet-capsets` có N binding → table render N row. Cột "Thao tác" chỉ row đầu có nút "Sửa". Còn lại N-1 row có Snapshot + Manage Masters.
- User click "Sửa" (row đầu) → modal mở → flip "Kích hoạt debezium Sync" → BE cascade is_active xuống mọi binding của source — đúng intent: "tắt toàn bộ source".
- User click Switch "Trạng thái table" inline (mỗi row) → route `PATCH /shadow-bindings/:id` per binding — chỉ flip 1 binding (giữ nguyên hành vi cũ đã đúng).
- Khi source OFF → Snapshot + Manage Masters disabled trên mọi row của source đó, Tooltip giải thích.
- Khi binding OFF → Quét field disabled trên row đó, Tag "Binding chưa active".

## 7. Out of scope

- KHÔNG đụng BE handler — cascade vẫn đúng semantic.
- KHÔNG migration / DB schema.
- KHÔNG đụng inline Switch behavior — đã đúng từ trước.
- Tab "Shadow Bindings" — không liên quan bug này.

## 8. Lessons-candidate

- "UX cascade visibility": khi BE field cascade lên N row, FE phải render entry-point edit chỉ 1 lần/scope cha — tránh user nghĩ edit per-row. Đăng ký lesson sau khi user confirm fix work.
