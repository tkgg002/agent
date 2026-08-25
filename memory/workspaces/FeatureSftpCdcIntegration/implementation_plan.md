# Kế hoạch Tích hợp SFTP Source Connector vào hệ thống CDC

Kế hoạch này mô tả giải pháp kỹ thuật để lắng nghe và đồng bộ dữ liệu thay đổi (data changes) từ file đối soát final của `reconcile-service` (được đẩy lên SFTP) vào hệ thống CDC hiện tại thông qua việc tích hợp **SFTP Source Connector** trên Kafka Connect và tận dụng bộ máy xử lý của `cdc-worker`.

## User Review Required

> [!IMPORTANT]
> **Quy trình ghi file final của `reconcile-service`:** Để tránh lỗi đọc file dở dang (Partial Read) do SFTP Connector quét trúng khi file đang được ghi, `reconcile-service` bắt buộc phải upload dưới dạng file tạm (ví dụ: `*.csv.tmp`) và đổi tên (Rename) thành tệp tin chính thức (`*.csv`) ngay sau khi hoàn tất upload.
> **Phân quyền thư mục SFTP:** Kafka Connect SFTP Connector cần quyền đọc, ghi và xóa/di chuyển file trong thư mục SFTP để chuyển các file đã xử lý xong vào thư mục lưu trữ (`/processed`), tránh việc đọc lặp lại dữ liệu.

## Proposed Changes

Giải pháp được thực hiện thông qua việc bổ sung cấu hình và code tích hợp nhẹ trên cả 2 dịch vụ **`cdc-cms-service`** và **`cdc-worker`** (nằm trong `centralized-data-service`).

---

### 1. cdc-cms-service (Control Plane - Quản lý Connector)

Cấu hình để CMS cho phép tạo và quản lý SFTP Source Connector trực tiếp từ UI/API, tự động gọi sang Kafka Connect.

#### [MODIFY] [`connector_types.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/source/connector_types.go)
* Định nghĩa thêm loại nguồn dữ liệu mới là `sftp`.

#### [MODIFY] [`debezium_connector.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/source/debezium_connector.go)
* Bổ sung template sinh cấu hình JSON cụ thể cho **SFTP Source Connector** khi nhận lệnh `system-connector.create` với loại nguồn là `sftp`.
* Mapping các tham số từ form CMS (host, port, credentials, input.path, finished.path, target topic...) sang cấu hình JSON của Confluent SFTP Source Connector.

---

### 2. cdc-worker (Data Plane - Xử lý Event CDC)

Bổ sung Adapter để chuyển đổi dữ liệu phẳng từ SFTP Connector sang định dạng sự kiện CDC tiêu chuẩn và thực hiện ghi xuống Postgres Shadow Table.

#### [NEW] [`sftp_adapter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/sftp_adapter.go)
* Tạo một adapter chuyển đổi payload dạng phẳng (flat JSON) từ CSV của SFTP Connector thành struct `CDCEvent` chuẩn:
  * Đặt `Data.Op = "c"` (Create) hoặc `"u"` (Update).
  * Gán toàn bộ payload JSON phẳng vào trường `Data.After`.
  * Set `Source = "sftp-connector"`.

#### [MODIFY] [`event_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
* Tại hàm `HandleRaw`, bổ sung nhánh kiểm tra nguồn event. Nếu topic bắt đầu bằng `sftp.`, tự động chạy qua `sftp_adapter` để chuẩn hóa payload trước khi gọi `processEvent`.

#### [MODIFY] [`kafka_consumer.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)
* Bổ sung cơ chế tự động subcribe thêm Kafka topic dạng `sftp.*` được sinh ra bởi SFTP Source Connector dựa trên cấu hình config.

---

### 3. Cấu hình Database Metadata (cdc_system)

Thực hiện đăng ký Registry để CDC-worker tự động ánh xạ dữ liệu:
* **Bảng `cdc_system.source_object_registry`:** Đăng ký đối tượng nguồn `source_object_name = "reconcile_final"`, `primary_key_field = "transaction_id"` (hoặc khóa chính tương ứng).
* **Bảng `cdc_system.shadow_binding`:** Đăng ký liên kết ánh xạ sang Postgres target table `shadow_reconcile_final`.
* **Bảng `cdc_system.mapping_rule_v2`:** Đăng ký các rules biến đổi và cast kiểu dữ liệu cho từng cột của file final.

---

## Verification Plan

### Automated Tests
* Viết Unit Test cho `sftp_adapter.go` để đảm bảo chuyển đổi chính xác từ dữ liệu phẳng sang struct `CDCEvent`.
* Chạy test suite của `centralized-data-service` để đảm bảo không bị regression:
  ```bash
  go test -v ./internal/handler/shadow/...
  ```

### Manual Verification
1. Dùng Postman/CMS UI gửi lệnh tạo SFTP Connector, verify connector được khởi tạo thành công trên Kafka Connect và ở trạng thái `RUNNING`.
2. Tạo một file giả lập `reconcile_final_20260806.csv` đẩy lên SFTP server.
3. Kiểm tra xem file có tự động được di chuyển vào thư mục `/processed` hay không.
4. Kiểm tra Kafka topic `sftp.reconcile.final.events` nhận được message dạng JSON phẳng.
5. Kiểm tra log của `cdc-worker` xem đã nhận event, tự tạo/reconcile cột và ghi thành công dữ liệu vào bảng `shadow_reconcile_final` trong Postgres.
