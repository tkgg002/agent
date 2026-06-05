# Requirements: Snapshot Monitor Tab

## Mục tiêu (Goal)
- Giám sát tiến độ Snapshot V2 từ hệ thống bằng một màn hình/tab trực quan trên `cdc-cms-web`.
- Kết nối liền mạch từ màn hình Activity Log (khi có sự kiện `snapshot.v2`).

## Phạm vi (Scope)
- **Frontend (`cdc-cms-web`)**:
  - Tab/Route mới: `/snapshot-monitor`.
  - Hỗ trợ querystring `?source_database=...&source_table=...` để tự động filter.
  - Các cột hiển thị: ID, Source DB, Source Table, Status, Rows Processed, Trace ID, Started At, Finished At, Error Msg, Cluster Time.
  - Từ màn hình Activity Log, click vào link của một event `snapshot.v2` sẽ nhảy sang `/snapshot-monitor`.
- **Backend (`cdc-cms-service`)**:
  - Read Model: `SnapshotProgressRow`.
  - Filter Model: Filter theo Database, Table, Status.
  - Endpoint `GET /api/snapshot-progress` trả về danh sách Snapshot Progress với đầy đủ thông tin (JOIN với bảng `cdc_source_objects` để lấy tên DB/Table).

## Không bao gồm (Out of scope)
- Không có chức năng Cancel/Pause snapshot (đó là write operation, sẽ xử lý ở phase khác nếu cần).
