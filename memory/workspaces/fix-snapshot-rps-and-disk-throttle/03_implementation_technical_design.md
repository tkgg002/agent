# 03_implementation_technical_design.md — Thiết kế kỹ thuật chi tiết

## 1. Kiến trúc luồng dữ liệu (Data Flow Architecture)

```mermaid
sequenceDiagram
    autonumber
    actor Operator as Operator (Browser)
    participant Web as cdc-cms-web (TableRegistry.tsx)
    participant CMS as cdc-cms-service (API & Command Bus)
    participant DB as Postgres (cdc_system)
    participant CDS as centralized-data-service (SnapshotRunner)
    participant ShadowDB as Postgres (cdc_shadow)

    Operator->>Web: Nhập Snapshot Max RPS = 1500 -> Bấm Lưu
    Web->>CMS: PATCH /api/v1/source-objects/:id { snapshot_max_rps: 1500 }
    CMS->>DB: UPDATE cdc_system.source_object_registry SET snapshot_max_rps = 1500
    Operator->>Web: Bấm Resume Snapshot
    Web->>CMS: POST /api/v1/snapshot-progress/:id/resume
    CMS->>CDS: NATS dispatch cdc.cmd.snapshot.v2 (resume)
    CDS->>DB: Đọc source_object_registry (SnapshotMaxRPS = 1500)
    loop Từng batch (batchSize = 3000-5000)
        CDS->>ShadowDB: Flush batch vào Shadow Table
        CDS->>CDS: time.Sleep(expectedDuration - elapsed)
        Note over CDS,ShadowDB: Disk I/O được xả, duy trì < 35%
        CDS->>DB: Checkpoint last_seen_id + rows_processed
    end
```

## 2. Chi tiết sửa đổi (Detailed Code Changes)

### 2.1 Backend (`cdc-cms-service`)
- `internal/app/queries/source/source_objects_read_models.go`:
  - `SnapshotMaxRPS *int json:"snapshot_max_rps,omitempty"`
- `internal/infra/persistence/source/source_object_read_repo_gorm.go`:
  - Thêm `so.snapshot_max_rps,` vào câu SQL SELECT trong `ListSourceObjects`.
- `internal/app/commands/source/update_source_object_v2.go`:
  - `SnapshotMaxRPS *int json:"snapshot_max_rps,omitempty"`
  - `Validate()`: Kiểm tra `v == 0` (clear) hoặc `10 <= v <= 100000`.
  - `Handle()`: Map `snapshot_max_rps` vào `updates` map.
- `internal/api/source/source_object_actions_handler.go`:
  - Thêm `SnapshotMaxRPS *int` vào request DTO và gán vào Command.

### 2.2 Frontend (`cdc-cms-web`)
- `src/types/index.ts`:
  - Thêm `snapshot_max_rps?: number | null;` vào `SourceObjectRow`.
- `src/pages/TableRegistry.tsx`:
  - `V2_EXCLUSIVE_FIELDS`: thêm `'snapshot_max_rps'`.
  - `openEdit`: nạp `snapshot_max_rps: record.snapshot_max_rps ?? undefined`.
  - `handleEdit`: xử lý 0 = clear về NULL.
  - Modal Form: thêm InputNumber field cho `snapshot_max_rps`.
