# Thiết kế Kỹ thuật Chi tiết: Snapshot Monitor

## 1. Backend (cdc-cms-service)

### Model & CQRS Query
```go
// internal/app/queries/snapshot_progress_read_models.go
type SnapshotProgressRow struct {
	ID                     int64      `json:"id"`
	SourceObjectID         int64      `json:"source_object_id"`
	SourceDatabase         string     `json:"source_database"` // Từ cdc_source_objects
	SourceTable            string     `json:"source_table"`    // Từ cdc_source_objects
	Status                 string     `json:"status"`
	LastSeenID             *string    `json:"last_seen_id"`
	RowsProcessed          int64      `json:"rows_processed"`
	TraceID                *string    `json:"trace_id"`
	ErrorMsg               *string    `json:"error_msg"`
	StartedAt              time.Time  `json:"started_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	FinishedAt             *time.Time `json:"finished_at"`
	MongoClusterStartMs    *int64     `json:"mongo_cluster_time_start_ms"`
	MongoClusterCaptureMethod *string `json:"mongo_cluster_time_capture_method"`
}

type SnapshotProgressFilter struct {
	SourceDatabase string
	SourceTable    string
	Status         string
}
```

### Persistence (GORM)
```go
// internal/infra/persistence/snapshot_progress_read_repo_gorm.go
func (r *SnapshotProgressReadRepoGorm) ListSnapshotProgress(ctx context.Context, f queries.SnapshotProgressFilter, page, pageSize int) ([]queries.SnapshotProgressRow, int64, error) {
	var rows []queries.SnapshotProgressRow
	var total int64

	q := r.db.WithContext(ctx).Table("cdc_system.snapshot_progress sp").
		Select("sp.*, so.database as source_database, so.table as source_table").
		Joins("JOIN cdc_system.cdc_source_objects so ON so.id = sp.source_object_id")

	if f.SourceDatabase != "" {
		q = q.Where("so.database = ?", f.SourceDatabase)
	}
	if f.SourceTable != "" {
		q = q.Where("so.table = ?", f.SourceTable)
	}
	if f.Status != "" {
		q = q.Where("sp.status = ?", f.Status)
	}

	err := q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = q.Order("sp.started_at DESC").Limit(pageSize).Offset(offset).Scan(&rows).Error
	return rows, total, err
}
```

## 2. Frontend (cdc-cms-web)

### SnapshotMonitor.tsx
- Sử dụng hook `useLocation` để parse query param `source_database` và `source_table` làm filter mặc định.
- Render một Table chứa toàn bộ SnapshotProgressRow lấy từ `/api/snapshot-progress`.
- Component có tính năng Auto Refresh nếu có record đang ở trạng thái `running`.

### App.tsx & ActivityLog.tsx
- Trong `App.tsx`: `const SnapshotMonitor = lazy(() => import('./pages/SnapshotMonitor'));`
- Trong `ActivityLog.tsx`:
```tsx
if (r.operation === 'snapshot.v2' || r.operation === 'debezium-snapshot') {
    return (
        <Space direction="vertical" size={0}>
            <Tag color="purple">{r.operation}</Tag>
            <Link to={`/snapshot-monitor?source_database=${r.source_database}&source_table=${r.source_table}`}>
                <Button size="small" type="link">View Progress</Button>
            </Link>
        </Space>
    );
}
```
