# Yêu cầu: Khắc phục lỗi bất đồng bộ/đua lệnh (Race Condition) trong luồng SFTP Snapshot

## Bối cảnh & Hiện tượng
Khi tạo mới một SFTP Source Connector (ví dụ: `testsftp13`), hệ thống Kafka Connect lập tức quét thư mục SFTP và đẩy tất cả dữ liệu ban đầu (snapshot/initial load) lên Kafka topic `cdc.sftplocal.testsftp13.reconcile_final`.
Tuy nhiên, tại thời điểm này:
- Bảng Shadow trong cơ sở dữ liệu chưa được sinh ra sạch sẽ.
- Trạng thái liên kết (Shadow Binding) trong Metadata Registry của Worker chưa được kích hoạt (`Active Binding` ở giao diện).
- Worker tự động phát hiện topic Kafka mới này và bắt đầu tiêu thụ (consume) ngay lập tức.
- Vì chưa có thông tin cấu hình và bảng Shadow tương ứng của nguồn SFTP này trong Registry, Worker nhận diện sai route (hoặc fall back về các route có cùng tên bảng của nguồn khác như PostgreSQL `reconcile_final`) và bỏ qua/skip toàn bộ message.
- Worker vẫn ghi nhận và commit offset của các message đã tiêu thụ lên Kafka.
- Khi người dùng hoàn tất các bước `Active Binding`, bảng Shadow được tạo ra nhưng do offset đã bị commit lên mức cao nhất (ví dụ: `216`), Worker không bao giờ tiêu thụ lại các message snapshot nữa. Dẫn đến bảng Shadow hoàn toàn trống rỗng (không chạy sync snapshot).
- Các luồng cập nhật (update) sau đó vẫn hoạt động bình thường vì lúc này liên kết đã kích hoạt.

## Yêu cầu sửa đổi
1. **Tránh tiêu thụ sớm (Prevent Premature Consumption):** Điều chỉnh Worker để chỉ đăng ký tiêu thụ (subscribe) các topic SFTP khi có liên kết Shadow Binding ở trạng thái kích hoạt (`IsActive = true`) tương ứng trong Registry.
2. **Khôi phục trạng thái:** Cung cấp quy trình/lệnh để reset offset của các topic SFTP bị tiêu thụ sớm về `0` (earliest), cho phép Worker đồng bộ lại toàn bộ dữ liệu snapshot ban đầu mà không cần restart hệ thống trong vận hành (hoặc chỉ restart 1 lần duy nhất để cập nhật logic mới của Worker).
