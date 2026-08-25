# Implementation Plan: Lazy Connector Creation Architecture (SFTP ONLY Scope)

Thiết kế lại luồng khởi tạo Kafka Connector: **CHỈ ÁP DỤNG DUY NHẤT CHO SFTP / FILE STREAM CONNECTORS**. Tất cả các Nguồn dữ liệu Database SQL / NoSQL khác (MongoDB, PostgreSQL, MySQL, Oracle, SQL Server...) **GIỮ NGUYÊN 100% LUỒNG TẠO CONNECTOR CŨ (Eager Creation)**.

> [!IMPORTANT]
> **Vị trí can thiệp chính xác trong mã nguồn:**
> - **Chiều Tạo Connection:** `cdc-cms-service/internal/app/commands/source/debezium_connector.go` (`CreateSystemConnectorHandler.Handle`).
> - **Chiều Click Snapshot SFTP:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go` (Hàm `runSnapshot`, nhánh `if isSFTP`).
> - **KHÔNG Can Thiệp:** Hủy bỏ hoàn toàn các thay đổi đối với `TriggerSnapshot` trong `cdc-cms-service`.
> - **Code Preservation:** Tất cả code cũ của SFTP (gọi Create trực tiếp khi đăng ký, luồng `resume`/`restart` cũ) đều được comment giữ lại (`// LEGACY ... - PRESERVED`).

---

## Proposed Changes & Full Code Demos

### 1. Core Backend CMS - Chiều Tạo Connection (`cdc-cms-service`)

#### [MODIFY] [debezium_connector.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go)
- Sửa `CreateSystemConnectorHandler.Handle`:
  - Phân nhánh `if isSFTP`:
    - **SFTP:** Comment out code cũ gọi Kafka Connect API, chỉ lưu Fingerprint vào DB `sourceRepo` với `Status = "configured"`.
    - **Database khác:** Chạy luồng Saga `http-create-connector` + `db-upsert-fingerprint` cũ 100% KHÔNG ĐỔI.

---

### 2. Core Worker CDS - Chiều Click Snapshot SFTP (`centralized-data-service`)

#### [MODIFY] [snapshot_runner_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go)
- Sửa hàm `runSnapshot` tại nhánh `if isSFTP`:
  1. Gọi `GET /connectors/{connectionCode}/status` trên Kafka Connect REST API.
  2. Nếu trả về lỗi hoặc chưa tồn tại (trạng thái Lazy `"configured"`):
     - Lấy `raw_config_sanitized` đã lưu từ bảng `cdc_system.sources`.
     - Gọi `POST /connectors` để **Dynamic Create SFTP Connector** lên Kafka Connect ngay tại mốc thời gian click Snapshot!
     - Update trạng thái `status = 'active'` trong DB.
  3. Nếu đã tồn tại: Tiếp tục chạy restart task/ingest dữ liệu như bình thường.
  4. Comment preserving các đoạn resume code cũ.

---

## Verification Plan

### Automated Tests
- Chạy unit tests cho `CreateSystemConnectorHandler` và `SnapshotRunner`:
  ```bash
  go test ./internal/app/commands/source/... -v
  ```

### Manual Verification
1. **Kiểm tra Nguồn Database (MongoDB / PostgreSQL):**
   - Đăng ký MongoDB connector từ UI -> Connector khởi tạo trên Kafka Connect ngay lập tức (Eager flow giữ nguyên 100%).
2. **Kiểm tra Nguồn SFTP:**
   - Đăng ký SFTP connector mới (`testsftp31`) -> Chưa xuất hiện trên Kafka Connect (`http://localhost:8084/connectors`).
   - Click Snapshot `testsftp31` -> Handler `snapshot_runner_handler.go` kiểm tra thấy 404, tự động POST `/connectors` khởi tạo `testsftp31` trên Kafka Connect và nạp dữ liệu từ file CSV lập tức.
