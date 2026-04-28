# Implementation — Phase 6 Debezium-only Pruning

## API / Contract Audit Summary

Đã audit trước khi sửa:

- `QueueMonitoring` là page FE riêng dựa vào worker stats, nhưng không còn là tuyến điều hướng chính cần tách khỏi `SystemHealth`.
- `bridge` endpoint trong backend đã retire về semantics, dù route legacy vẫn còn mount.
- `airbyte-sync` chỉ còn là dấu vết legacy trong UI copy/options, không còn đúng mục tiêu hiện tại.

## FE Changes

### `src/App.tsx`

- Bỏ `QueueMonitoring` khỏi menu `Advanced`.
- Bỏ lazy route page thật.
- Giữ compatibility bookmark bằng redirect:
  - `/queue` -> `/system-health`

### `src/pages/ActivityManager.tsx`

- Xóa operation:
  - `bridge`
  - `airbyte-sync`
- Update operator guidance để nói rõ CMS hiện chạy Debezium-only.

### `src/pages/DataIntegrity.tsx`

- Bỏ `airbyte` / `both` khỏi màu Sync Engine.
- Đổi tooltip Sync Engine để nói theo current target: Debezium là luồng chuẩn.

### `src/pages/SystemHealth.tsx`

- Bỏ `Airbyte` khỏi danh sách infrastructure components first-class.

### `src/pages/MappingFieldsPage.tsx`

- Đổi `_source` description từ `airbyte/debezium` sang `debezium`.
- Đổi error copy fallback scan để không còn nhắc Airbyte.

### `src/pages/ActivityLog.tsx`

- Bỏ các operation legacy khỏi filter chính:
  - `bridge`
  - `cmd-bridge-airbyte`
  - `scan-airbyte-streams`
  - `auto-register-stream`
  - `bridge-sql`
  - `bridge-batch-pgx`
- Thêm note đầu page để nói rõ UI đã rút về Debezium-only.

## Kết quả

- Surface FE gọn hơn, ít khái niệm thừa hơn.
- Operator không còn bị đẩy về các luồng Airbyte/bridge đã retire.
