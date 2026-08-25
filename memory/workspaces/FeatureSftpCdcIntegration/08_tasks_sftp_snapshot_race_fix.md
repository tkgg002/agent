# Danh sách Task: SFTP Snapshot Race Condition Fix

- [ ] Sửa đổi `discoverTopics` trong `topic_helper.go` để lọc bỏ các topic SFTP chưa có shadow binding hoạt động.
- [ ] Chạy unit test backend của shadow package để xác minh không ảnh hưởng tới logic cũ.
- [ ] Dừng worker daemon để cho phép thay đổi offset.
- [ ] Chạy lệnh reset offset của topic `cdc.sftplocal.testsftp13.reconcile_final` về `0` (earliest).
- [ ] Khởi chạy lại worker daemon.
- [ ] Kiểm tra bảng shadow và log của worker xem dữ liệu đã được đồng bộ chuẩn xác chưa.
