# Solution — Phase 1 CMS-FE Audit & Reform Plan

## Kết luận chính

### Phải bỏ đầu tiên

- `CDCInternalRegistry`

### Nên giữ làm lõi V2

- `SourceConnectors`
- `SourceToMasterWizard`
- `MasterRegistry`
- `SchemaProposals`
- `TransmuteSchedules`
- `SystemHealth`
- `DataIntegrity`
- `ActivityLog`

### Nên gộp / đổi tên

- `TableRegistry` -> `Source Objects`
- `MappingFieldsPage` -> `Mapping Rules`
- `ActivityManager` -> `Operations`
- `QueueMonitoring` -> gộp vào `SystemHealth`
- `SchemaChanges` -> hợp nhất dần vào flow proposals/mapping

## Đề xuất phase refactor FE

### Phase A — Remove obvious legacy

1. bỏ `CDCInternalRegistry`
2. bỏ link/menu `cdc-internal`
3. sửa text/label còn nhắc `cdc_internal`

### Phase B — Reframe navigation

1. nhóm `Setup`
2. nhóm `Operate`
3. nhóm `Advanced`

### Phase C — Refactor data model screens

1. `TableRegistry` -> V2 source object / shadow binding
2. `MappingFieldsPage` -> mapping rule V2
3. `Masters` -> binding-aware UI

### Phase D — Consolidate ops screens

1. gộp `QueueMonitoring` vào `SystemHealth`
2. gộp `ActivityManager` vào `Schedules/Operations`
3. quyết định giữ hay bỏ `Dashboard`
