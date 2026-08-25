# Yêu cầu: Cô lập định tuyến (Route Isolation) cho các topic để tránh tranh chấp và cho phép chạy lại SFTP Snapshot

## Bối cảnh & Vấn đề
Khi một SFTP Source Connector mới được tạo, nó đẩy toàn bộ dữ liệu snapshot ban đầu lên Kafka topic (ví dụ: `cdc.sftplocal.testsftp13.reconcile_final`).
Do liên kết `Active Binding` chưa được kích hoạt, route cho nguồn này chưa có trong registry.
Tuy nhiên, trong `ResolveSourceRoutes`:
- Khi tìm kiếm route cho `sourceDB = "testsftp13"` và `sourceTable = "reconcile_final"`, hệ thống sử dụng các khóa tìm kiếm fallback, trong đó có khóa chung `reconcile_final` (không có prefix `sourceDB`).
- Khóa chung `reconcile_final` này khớp với route của một nguồn khác đang hoạt động (ví dụ: PostgreSQL `reconcile_final`).
- Kết quả là Worker ngầm hiểu sự kiện của SFTP là của PostgreSQL, thấy thiếu PK `transaction_id` nên bỏ qua (skip) và commit offset lên mức cao nhất.
- Khi người dùng kích hoạt `Active Binding`, bảng Shadow được tạo ra nhưng không nhận được dữ liệu snapshot do offset đã bị commit sớm.

## Yêu cầu sửa đổi
1. **Cô lập định tuyến (Route Isolation):** Sửa đổi hàm `ResolveSourceRoutes` trong `metadata_registry_service.go` để nếu khóa khớp là khóa fallback chung (`sourceTable`), ta phải lọc lại danh sách route và chỉ giữ lại những route thực sự khớp với `sourceDB` (schema name hoặc connection code). Điều này ngăn chặn triệt để việc tranh chấp định tuyến chéo giữa các database/connection khác nhau có cùng tên bảng.
2. **Cho phép chạy lại Snapshot dễ dàng:** Không lọc ẩn topic ở tầng discovery để giữ hệ thống tường minh và kiểm soát được. Nếu xảy ra trường hợp tiêu thụ sớm khi liên kết chưa hoạt động, Worker sẽ cảnh báo rõ ràng `event skipped: source not in registry cache`. Người dùng chỉ cần kích hoạt liên kết và nhấn nút **"Xóa Offset"** (Reset Offset) trên giao diện quản lý Connector để đẩy lại dữ liệu snapshot và Worker sẽ tự động đồng bộ lại từ đầu.
