# Phân tích Tổng hợp & Root Cause CDC-System (2026-05-02)

Dựa trên việc đối chiếu 4 báo cáo review và thực tế codebase tại `@/Users/trainguyen/Documents/work/cdc-system`.

## 1. Khám phá chấn động: Sự phân mảnh DDL (The Schema Schism)
Đây là "tử huyệt" của hệ thống hiện tại.

| Thành phần | File Source | Convention PK | Metadata Columns |
|---|---|---|---|
| **Provisioning (V1)** | `schema_adapter.go` | `id` (TEXT) | 8 cột (`_raw_data`, `_source`, ...) |
| **SinkWorker (V2)** | `schema_manager.go` | `_gpay_id` (BIGINT) | 10 cột (thêm `_gpay_source_id`, `_source_ts`) |

**Kết quả:** Các bảng `addtest` được tạo bởi Provisioning nên mang schema V1. Trong khi đó, module Transmuter (V2) lại code cứng:
```go
// internal/service/transmuter.go
qt := fmt.Sprintf(`SELECT _gpay_id, ... WHERE _gpay_id > ?`)
```
=> Dẫn đến lỗi `column "_gpay_id" does not exist` (B6).

## 2. Giải mã các Blockers tồn đọng (Track E)

### 🔴 B4: SchemaValidator Drift
- **Hiện trạng:** `SchemaValidator` (trong `internal/service/schema_validator.go`) quá nghiêm ngặt. Khi Debezium đẩy message có thêm các cột meta hoặc cấu trúc thay đổi nhẹ, nó reject ngay lập tức thay vì auto-evolve.
- **Vấn đề:** Nó đang chặn đứng ingest của topic `orders` (lag=3).

### 🔴 B5: DLQ Binary/UTF-8 0x00
- **Hiện trạng:** Khi message bị reject và đẩy vào DLQ, cột `raw_json` trong `failed_sync_logs` là kiểu `TEXT`. Payload Avro chứa byte `0x00` (null byte) khiến PostgreSQL từ chối INSERT.
- **Hậu quả:** Loop redelivery vô tận, tốn CPU nhưng không ghi được log lỗi.

### 🔴 B6: Transmute Hardcode `_gpay_id`
- **Hiện trạng:** Đã xác minh ở mục 1. Codebase đang giả định mọi shadow table đều có `_gpay_id`.
- **Thực tế:** Các bảng Provisioned theo V2 bridge (addtest) lại dùng `id`.

### 🟡 B8: MariaDB Connector Plugin
- **Hiện trạng:** Container `gpay-kafka-connect` thiếu file `.jar` của MySQL/MariaDB.
- **Dấu hiệu:** Connector MariaDB không thể START, báo lỗi `Failed to find any class that implements Connector`.

---

## 3. Đánh giá độ tin cậy của các báo cáo trước

- `report_system_summary.md`: Tốt về mặt tĩnh (vẽ ra viễn cảnh), nhưng kém về mặt động (không thấy được sự xung đột schema).
- `report_v2_bridge_e2e_verification.md`: Tốt khi verify được control-plane, nhưng sai lầm khi nói "Master table chưa tồn tại" (thực tế đã tồn tại).
- `report_review_v2_bridge_20260501.md`: Rất tốt, đã chỉ ra đúng điểm "Master tables ĐÃ tồn tại" và sự biến mất của DLQ.

## 4. Kết luận
Hệ thống không "chết", nó chỉ đang bị **lệch pha (out of sync)** giữa các module mới (SinkWorker/V2) và các module cũ/adapter (Provisioning/V1).

**Skills used**: Codebase Archaeology, DB Schema Verification, Log Analysis, Architecture Pattern Matching.
