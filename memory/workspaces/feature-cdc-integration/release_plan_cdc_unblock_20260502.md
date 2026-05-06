# Release Plan: CDC System Unblock & Schema Unification
**Version**: 1.0
**Date**: 2026-05-02
**Status**: DRAFT (Waiting for Approval)

## 1. Mục tiêu
Dứt điểm 3 blockers kỹ thuật (B4, B5, B6) và 1 blocker hạ tầng (B8) để thông luồng dữ liệu từ Source -> Shadow -> Master DW cho các sources `addtest`.

## 2. Các thay đổi đề xuất

### 🔹 Giai đoạn 1: Sửa lỗi Code-level (Muscle Execute)

#### [MODIFY] `internal/service/transmuter.go` (Fix B6)
- **Thay đổi**: Loại bỏ việc hardcode `_gpay_id`. Thay vào đó, đọc thông tin PK từ `shadow_binding` (đã có trong runtime context).
- **Lợi ích**: Cho phép Transmuter chạy được trên cả bảng shadow V1 (PK=`id`) và V2 (PK=`_gpay_id`).

#### [MODIFY] `internal/handler/kafka_consumer.go` (Fix B5)
- **Thay đổi**: Thực hiện Base64 encode trường `raw_json` trước khi ghi vào bảng `failed_sync_logs` nếu phát hiện payload chứa binary/Avro.
- **Lợi ích**: Ngăn chặn lỗi `invalid byte sequence for encoding "UTF8"` khiến DLQ bị kẹt.

#### [MODIFY] `internal/service/schema_validator.go` (Fix B4)
- **Thay đổi**: Chuyển chế độ từ `Strict` sang `Permissive-Additive`. Cho phép các cột mới xuất hiện trong message mà chưa có trong shadow table (ghi vào `_raw_data` và phát proposal thay vì reject message).
- **Lợi ích**: Xử lý được Schema Drift mà không làm dừng pipeline.

---

### 🔹 Giai đoạn 2: Khắc phục Hạ tầng & Dữ liệu

#### [INFRA] Cài đặt MariaDB Connector (Fix B8)
- **Hành động**: `docker exec` vào `gpay-kafka-connect` để download `debezium-connector-mysql` plugin.
- **Lệnh dự kiến**: `confluent-hub install debezium/debezium-connector-mysql:latest`.

#### [DATA] Đồng bộ Schema Shadow Tables
- **Hành động**: Chạy script ALTER TABLE để thêm các cột meta thiếu (`_gpay_id`, `_gpay_source_id`) vào 3 bảng shadow `addtest` hiện tại để chúng đạt chuẩn V2.
- **Script**: `ALTER TABLE shadow_src_local_pg_source.orders_addtest ADD COLUMN IF NOT EXISTS _gpay_id BIGINT ...`

---

## 3. Kế hoạch Verify
1. **Unit Test**: Chạy lại bộ test của `transmuter` và `kafka_consumer`.
2. **Integration Test**: Restart `cdc-worker`, quan sát stats `processed` tăng lên (lag về 0).
3. **E2E Test**: Kiểm tra bảng `dw_src_local_pg_source.orders_addtest` xem có dữ liệu mới từ source chưa.

## 4. Rủi ro & Giải pháp
- **Rủi ro**: ALTER TABLE trên bảng lớn gây lock.
- **Giải pháp**: Hiện tại các bảng `addtest` đang có 0 rows, thực hiện ALTER cực kỳ an toàn.

---
**Brain Approval Required**: Cần User confirm plan này để em delegate cho Muscle thực thi.

**Skills used**: Release Engineering, SQL Migration Design, Go Refactoring Strategy, Infrastructure Management.
