# 01_requirements.md — Yêu cầu chi tiết (Specs)

## 1. Yêu cầu chức năng (Functional Requirements)
- **REQ-01: Hiển thị trường Snapshot Max RPS trên CMS UI**
  - Modal *"Chỉnh sửa Source Object"* trên trang `TableRegistry.tsx` phải có thêm input field `Snapshot Max RPS (snapshot.v2)`.
  - Placeholder: `Để trống = không giới hạn tốc độ`.
  - Tooltip giải thích rõ tác dụng: điều tiết tốc độ đọc/ghi để chống tràn I/O đĩa.
  - Hỗ trợ nhập giá trị từ 10 đến 100,000 records/giây (hoặc để trống để clear về NULL).

- **REQ-02: Đồng bộ API Backend (`cdc-cms-service`)**
  - Endpoint `GET /api/v1/source-objects` phải trả về trường `snapshot_max_rps` từ bảng `cdc_system.source_object_registry`.
  - Endpoint `PATCH /api/v1/source-objects/:id` phải tiếp nhận payload `snapshot_max_rps`, validate hợp lệ (0 = clear to NULL, hoặc `10 <= v <= 100000`), và cập nhật vào database.

- **REQ-03: Thực thi tại Worker (`centralized-data-service`)**
  - Worker `SnapshotRunner` đã có logic tính `time.Sleep` dựa trên `SnapshotMaxRPS`.
  - Khi source object có `snapshot_max_rps > 0`, worker tự động hãm phanh sau mỗi batch.

## 2. Yêu cầu phi chức năng (Non-Functional Requirements)
- **NFR-01: Zero Data Loss & Resumable**
  - Tiến trình snapshot của `bank_requests` (12.6M rows) phải tiếp tục nạp từ `5,125,001` tới `12,614,888` dựa trên checkpoint `last_seen_id = '69e999af803579b1447f9140'`.
- **NFR-02: Ổn định I/O đĩa (Disk I/O Protection)**
  - Tốc độ ghi trung bình không vượt quá ngưỡng cấu hình, giữ Disk I/O của PostgreSQL dưới 40%.
