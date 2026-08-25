# 09_tasks_solution.md — Hồ sơ giải pháp kỹ thuật cụ thể

## 1. Backend Solutions (`cdc-cms-service`)

### 1.1 Read Model (`internal/app/queries/source/source_objects_read_models.go`)
```go
type SourceObjectListItem struct {
    ...
    SnapshotBatchSize *int `json:"snapshot_batch_size,omitempty"`
    SnapshotMaxRPS    *int `json:"snapshot_max_rps,omitempty"`
    ...
}
```

### 1.2 Read Repository (`internal/infra/persistence/source/source_object_read_repo_gorm.go`)
```go
// SELECT query trong ListSourceObjects:
so.snapshot_batch_size,
so.snapshot_max_rps,
so.is_active,
```

### 1.3 Command & Handler (`internal/app/commands/source/update_source_object_v2.go`)
```go
type UpdateSourceObjectV2Command struct {
    ...
    SnapshotBatchSize *int `json:"snapshot_batch_size,omitempty"`
    SnapshotMaxRPS    *int `json:"snapshot_max_rps,omitempty"`
    UpdatedBy         string `json:"updated_by,omitempty"`
}

var ErrSourceObjectInvalidMaxRPS = errors.New("invalid_snapshot_max_rps")

const (
    snapshotMaxRPSMin = 10
    snapshotMaxRPSMax = 100000
)

func (c UpdateSourceObjectV2Command) Validate() error {
    ...
    if c.SnapshotMaxRPS != nil {
        v := *c.SnapshotMaxRPS
        if v != 0 && (v < snapshotMaxRPSMin || v > snapshotMaxRPSMax) {
            return ErrSourceObjectInvalidMaxRPS
        }
    }
    return nil
}

func (h *UpdateSourceObjectV2Handler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
    ...
    if cmd.SnapshotMaxRPS != nil {
        if *cmd.SnapshotMaxRPS == 0 {
            updates["snapshot_max_rps"] = nil
        } else {
            updates["snapshot_max_rps"] = *cmd.SnapshotMaxRPS
        }
    }
    ...
}
```

### 1.4 API Handler (`internal/api/source/source_object_actions_handler.go`)
```go
var req struct {
    ...
    SnapshotBatchSize *int `json:"snapshot_batch_size"`
    SnapshotMaxRPS    *int `json:"snapshot_max_rps"`
}

cmd := source.UpdateSourceObjectV2Command{
    ...
    SnapshotBatchSize: req.SnapshotBatchSize,
    SnapshotMaxRPS:    req.SnapshotMaxRPS,
    UpdatedBy:         user,
}
```

## 2. Frontend Solutions (`cdc-cms-web`)

### 2.1 Types (`src/types/index.ts`)
```typescript
export interface SourceObjectRow {
  ...
  snapshot_batch_size?: number | null;
  snapshot_max_rps?: number | null;
  ...
}
```

### 2.2 TableRegistry (`src/pages/TableRegistry.tsx`)
```typescript
const V2_EXCLUSIVE_FIELDS = ['snapshot_batch_size', 'snapshot_max_rps', 'primary_key_field', 'primary_key_type'] as const;

// openEdit:
editForm.setFieldsValue({
  ...
  snapshot_max_rps: record.snapshot_max_rps ?? undefined,
});

// handleEdit:
if (payload.snapshot_max_rps == null) {
  if (editingRecord.snapshot_max_rps != null) {
    payload.snapshot_max_rps = 0;
  } else {
    delete payload.snapshot_max_rps;
  }
}

// Modal JSX:
<Form.Item
  name="snapshot_max_rps"
  label="Snapshot Max RPS (snapshot.v2)"
  tooltip="Giới hạn tốc độ đọc/ghi (records/giây) khi chạy snapshot.v2 cho source này để tránh nghẽn I/O đĩa database. Bỏ trống = không giới hạn. Ví dụ: 1000, 1500, 2000."
>
  <InputNumber
    min={10}
    max={100000}
    step={100}
    style={{ width: '100%' }}
    placeholder="Để trống = không giới hạn"
  />
</Form.Item>
```

## 3. Trace ID & Parent Trace Correlation Solutions

### 3.1 Cập nhật `claimProgress` khi Resume (`centralized-data-service/internal/handler/orchestration/snapshot_runner_state.go`)
```go
// Khi resume, cập nhật trace_id mới và xóa error_msg cũ
res := tx.Exec(`
    UPDATE cdc_system.snapshot_progress
    SET status = 'running',
        trace_id = ?,
        error_msg = NULL,
        updated_at = NOW()
    WHERE id = ? AND status IN ('paused', 'error')
`, p.TraceID, p.ProgressID)
```

### 3.2 Kế thừa / Liên kết Parent Trace ID khi Resume (`cdc-cms-service/internal/api/scheduler/snapshot_progress_handler.go`)
```go
// Khi dispatch resume, nạp trace_id gốc từ snapshot_progress để gán vào correlation header
var progress struct {
    SourceObjectID  int64   `gorm:"column:source_object_id"`
    ShadowBindingID *int64  `gorm:"column:shadow_binding_id"`
    TraceID         *string `gorm:"column:trace_id"`
}
// Đưa trace_id gốc vào payload và correlation context
payload := fmt.Sprintf(`{"source_object_id":%d,"shadow_binding_id":%d,"progress_id":%d,"trace_id":%q,"action":"resume","overwrite":false}`,
    progress.SourceObjectID, bindingID, id, origTraceID)
```

