# 00_context.md — Phạm vi & Bối cảnh

## 1. Bối cảnh sự cố (Incident Context)
- **Hệ thống**: Data Hub CDC (Centralized Data Service, CDC CMS Service, CDC CMS Web).
- **Hiện tượng**: Tiến trình Snapshot bảng dữ liệu lớn `bank_requests` (`banvietbank-connector-service` / `bvb-connector-service`) với 12,614,888 bản ghi bị dừng đột ngột tại mốc **5,125,000 bản ghi (40.63%)**.
- **Lỗi hiển thị trên CMS**: `Heartbeat timeout: progress was stuck in running state for too long (worker stopped)`.
- **Traces Log**: `dial tcp 10.200.185.20:5432: connect: connection refused` tới Shadow Database Postgres (`cdc_shadow`).
- **Báo cáo hạ tầng DevOps**: Máy chủ PostgreSQL bị **Disk I/O 95% - 100%** do tốc độ ghi dồn dập và bão hòa WAL / Checkpoint, dẫn tới dịch vụ PostgreSQL bị crash/restart.

## 2. Phạm vi giải quyết (Scope)
1. **Phân tích nguyên nhân gốc rễ (Root Cause)**:
   - Cơ chế kiểm tra Stale Snapshot Progress của CMS gây ra thông báo Heartbeat Timeout.
   - Nguyên nhân PostgreSQL sập do Disk I/O quá tải khi chạy unthrottled snapshot.
2. **Kích hoạt tính năng Điều tiết tốc độ (Rate Limiting / Throttling)**:
   - Đưa cấu hình `snapshot_max_rps` lên toàn bộ hệ sinh thái Data Hub: Database (`cdc_system.source_object_registry`), Backend (`cdc-cms-service`), và Giao diện Quản trị (`cdc-cms-web`).
3. **Đảm bảo tính liên tục & toàn vẹn dữ liệu**:
   - Resume tiến trình từ vị trí `5,125,000` (con trỏ `last_seen_id = '69e999af803579b1447f9140'`).
