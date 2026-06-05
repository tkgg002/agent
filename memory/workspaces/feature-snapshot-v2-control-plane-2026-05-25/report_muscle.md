## Execution Report
### Task: Implement Snapshot V2 Control Plane
### Phase: GĐ4 | Service Group: Worker
### Changes
| File | Change |
|------|--------|
| migrations/062_add_total_rows_to_progress.sql | Thêm cột total_rows vào bảng snapshot_progress |
| migrations/063_create_snapshot_dlq.sql | Tạo bảng snapshot_dlq |
| migrations/064_add_snapshot_rps_to_registry.sql | Thêm cấu hình snapshot_max_rps, error_mode |
| model/source_object_registry.go | Append GORM tags (SnapshotMaxRPS, SnapshotErrorMode) |
| model/snapshot_dlq.go | Tạo mới GORM Model cho DLQ |
| sinkworker/upsert.go | Sửa DO NOTHING thành DO UPDATE SET LWW Guard |
| handler/snapshot_runner_handler.go | NATS atomic.Bool Pause/Resume, MaxRPS Throttling, O(1) row_count, DLQ Bulk Insert |
| server/worker_server.go | Pass NATS Conn vào NewSnapshotRunner |

### Verification
- [x] Lint/Build: PASS
- [x] Unit Tests: PASS (go test handler)

### Memory Updated: ✅
