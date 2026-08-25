# 12_implementation_plan_sftp_scan_fix.md

# Kế hoạch Tự động Khởi tạo Kafka Topic cho SFTP Connector

## 1. Tóm tắt Vấn đề
Khi người dùng tạo Connector SFTP (`kafka-connect-fs`), Connector được đăng ký thành công trên Kafka Connect REST API. Tuy nhiên:
- Plugin `kafka-connect-fs` không tự khởi tạo Kafka Topic (`cdc.sftplocal.reconcile.final`) trên Broker cho đến khi nó đọc xong file đầu tiên.
- Vì Topic chưa tồn tại trên Kafka Broker, CDS worker chưa thể lắng nghe event để tự động tạo Shadow Table và ghi dữ liệu mẫu -> Luồng Quét Field báo lỗi `0 columns`.

## 2. Phương án Xử lý Tối ưu (The Single Best Approach)

### Bước 1: Auto-Create Kafka Topic tại CMS Backend (`cdc-cms-service`)
Trong [`system_connectors_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/system_connectors_handler.go):
- Khi tạo mới Connector loại SFTP (`sourceType == "sftp"`), CMS Backend sẽ tự động gọi helper tạo Topic `cdc.sftplocal.<collection>` trên Kafka Broker (với 1 partition, replication factor = 1).
- Việc này đảm bảo Topic luôn sẵn sàng trên Kafka Broker ngay từ khoảnh khắc Connector được tạo thành công!

### Bước 2: Cấp quyền đọc file SFTP trong Docker Local (`sftp-host`)
- Đảm bảo permission thư mục SFTP local `./docker/data/reconcile_final/` cho phép user `gp-reconcile-admin` đọc & ghi file CSV.

## 3. Kế hoạch Kiểm thử (Verification Plan)
1. Build `cdc-cms-service` server (`go build ./cmd/server`).
2. Kiểm tra khi tạo Connector SFTP mới -> Kafka Topic `cdc.sftplocal.reconcile.final` xuất hiện ngay lập tức trên Kafka Broker (verify qua `kafka-topics --list`).
3. Quét field -> Khởi tạo schema thành công 100%!
