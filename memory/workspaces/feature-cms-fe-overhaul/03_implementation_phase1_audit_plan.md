# Implementation — Phase 1 CMS-FE Audit & Reform Plan

## Route hiện tại

Nguồn: [App.tsx](/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/App.tsx)

### Public

- `/login`

### Protected

- `/`
- `/schema-changes`
- `/registry`
- `/cdc-internal`
- `/masters`
- `/schema-proposals`
- `/schedules`
- `/registry/:id/mappings`
- `/sources`
- `/queue`
- `/activity-log`
- `/activity-manager`
- `/data-integrity`
- `/system-health`
- `/source-to-master`

## Kết quả audit nhanh theo nhóm

### 1. Keep — lõi vận hành V2

- `SourceConnectors`
- `SourceToMasterWizard`
- `MasterRegistry`
- `SchemaProposals`
- `TransmuteSchedules`
- `SystemHealth`
- `DataIntegrity`
- `ActivityLog`

### 2. Merge / Rename — còn giá trị nhưng naming/flow đang legacy

- `TableRegistry`
  - nên đổi nghĩa sang `Source Objects / Shadow Bindings`
- `MappingFieldsPage`
  - nên gắn vào flow master binding / mapping rule V2 thay vì treo dưới `registry/:id`
- `SchemaChanges`
  - legacy approval flow cho `mapping_rules`
  - nên hợp nhất dần vào `SchemaProposals` hoặc `Mapping Rules`
- `ActivityManager`
  - còn giá trị vận hành, nhưng nên gộp vào `Operations / Schedules`

### 3. Remove — trái với target architecture hoặc trùng nhiệm vụ

- `CDCInternalRegistry`
  - trái trực tiếp với V2 vì `cdc_internal` đã bị loại
- `Dashboard`
  - hiện tại chủ yếu phản ánh registry legacy, nên nếu giữ phải viết lại bằng V2 metrics
- `QueueMonitoring`
  - có thể gộp vào `SystemHealth`

## Phát hiện quan trọng

1. Menu hiện tại trộn lẫn:
   - V1 registry
   - `cdc_internal` v1.25
   - V2 control-plane
2. Có page gọi API legacy `/api/registry`, `/api/mapping-rules`, `/api/schema-changes`
3. Có page gọi API V2 `/api/v1/...`
4. UX hiện tại tổ chức theo “module kỹ thuật” hơn là “operator flow”

## Thứ tự UX đề xuất mới

### Nhóm 1 — Setup / Onboard

1. `Sources`
2. `Source → Master Wizard`
3. `Source Objects & Shadow`
4. `Masters`
5. `Mapping Rules`

### Nhóm 2 — Operate

6. `Schedules`
7. `System Health`
8. `Data Integrity`
9. `Activity Log`

### Nhóm 3 — Advanced / Admin

10. `Schema Proposals`
11. `Operations`

## Page đề xuất sau cải tổ

### Giữ tên gần như nguyên

- `Sources`
- `Wizard`
- `Masters`
- `Schedules`
- `System Health`
- `Data Integrity`
- `Activity Log`

### Đổi tên / tái cấu trúc

- `TableRegistry` -> `Source Objects`
- `MappingFieldsPage` -> `Mapping Rules`
- `ActivityManager` -> `Operations`
- `Dashboard` -> `Overview` hoặc bỏ hẳn phase đầu

### Bỏ

- `CDCInternalRegistry`
- `QueueMonitoring` (gộp)
- `SchemaChanges` (sau khi hợp nhất xong)
