# 00_context — fix-source-object-edit-cascade-2026-05-29

## Trigger
User báo bug FE trang `/shadow` (TableRegistry): khi edit field "Kích hoạt debezium Sync" cho 1 row (binding của `wallet-capsets`), các binding khác cùng source cũng nhảy theo (toggle là true/false giống nhau).

## Bối cảnh
- Service: `cdc-cms-web` (FE Antd) + `cdc-cms-service` (BE Go)
- DB metadata: `cdc_system`
- Liên hệ workspace cũ: `cleanup-gpay-cols-2026-05-28` (rename `_gpay_*` → `source_id`/`_deleted`) — KHÔNG conflict; task này về UI semantic, không đụng cột.

## Field mapping (verified)
| Layer | Tên field |
|---|---|
| FE form | `is_active` (Form.Item name) |
| FE label | "Kích hoạt debezium Sync" (TableRegistry.tsx:1090) |
| API | `PATCH /api/v1/source-objects/:id` body `{is_active: bool}` |
| BE struct | `IsActive *bool` (update_source_object_v2.go:21) |
| DB table chính | `cdc_system.source_object_registry.is_active` |
| DB cascade | `cdc_system.shadow_binding.is_active` (update theo `source_object_id = ?`) |

## Root cause bug "nhảy theo"
1. Modal Edit (per binding row) submit payload đa-field (7 field).
2. `updateEntry` (TableRegistry.tsx:421-479) check `togglesBindingOnly = restKeys.length === 1 && restKeys[0] === 'is_active'` → false → route vào `PATCH /source-objects/:sourceObjectId` (V2).
3. BE handler `update_source_object_v2.go:148-161` cascade `is_active` lên TẤT CẢ `shadow_binding WHERE source_object_id = ?`.
4. → 1 cú flip ở Modal → mọi binding cùng source flip theo.

## User direction
- Move Edit ra ngoài (1 lần/source, KHÔNG mỗi binding 1 lần).
- Rule 1: "Trạng thái table" chưa bật → Quét field disabled.
- Rule 2: "Kích hoạt debezium Sync" (source) chưa bật → Snapshot + Manage Masters disabled.
