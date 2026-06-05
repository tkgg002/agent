# Context: Thêm tab monitor tiến trình snapshot

- **Feature**: Thêm 1 tab trên UI của cdc-cms-web để giám sát tiến trình snapshot (snapshot_progress).
- **Trigger**: Từ trang `activity-log`, khi có sự kiện liên quan đến `snapshot.v2`, user có thể click qua tab monitor này để xem chi tiết tiến độ.
- **Goal**:
  1. Frontend: Tạo tab Monitor Snapshot. Link qua từ Activity Log.
  2. Backend: Đảm bảo có API (có pagination/filter) để get data từ `cdc_system.snapshot_progress`.
- **System**:
  - `cdc-cms-web` (React/Vite).
  - `cdc-cms-service` (Go) cung cấp API.
