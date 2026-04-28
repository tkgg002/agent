# Implementation — Phase 2 FE Nav Refactor

## Code Changes

### `src/App.tsx`

- Bỏ lazy import `CDCInternalRegistry` khỏi routing chính.
- Chuyển `Menu` sang cấu trúc nhóm:
  - `Setup`
  - `Operate`
  - `Advanced`
- Đổi label:
  - `Table Registry` -> `Source Objects`
  - `Debezium Command Center` -> `Sources & Connectors`
  - `Quản lý tác vụ` -> `Operations`
  - `Queue Monitor` -> `Queue Monitoring`
  - `Mapping Approval` -> `Schema Review`
- Thêm redirect `Route path="/cdc-internal"` sang `/registry`.
- Dùng `useLocation()` để giữ selected menu key theo pathname thật.

### `src/pages/SourceToMasterWizard.tsx`

- Đổi Step 2 và Step 3 sang vocabulary V2:
  - `source object`
  - `shadow binding`
  - `shadow_<source_db>.<target>`
- Đổi Step 9 sang `master binding + transform spec`.
- Vá compatibility với Ant Design hiện tại:
  - bỏ `Steps.Step`
  - dùng `items` prop để build pass.

### `src/pages/TableRegistry.tsx`

- Đổi title sang `Source Objects`.
- Sửa comment mô tả source metadata để không còn ám chỉ `cdc_internal.sources`.

### `src/pages/SourceConnectors.tsx`

- Đổi title sang `Sources & Connectors`.

### `src/pages/MasterRegistry.tsx`

- Đổi title `Master Table Registry` -> `Master Registry`.
- Đổi placeholder `source_shadow (cdc_internal table)` -> `shadow_<source_db>.<table>`.

### `src/pages/ActivityManager.tsx`

- Đổi title sang `Operations`.

## Kết quả

- Navigation chính đã bỏ page `CDCInternalRegistry`.
- Runtime UI chính không còn đẩy operator về semantics `cdc_internal`.
- FE build pass sau khi vá API `Steps` của Ant Design.
