# Báo cáo thay đổi - Khắc phục transmute safety gate batchSize

## 1. Danh sách các file thay đổi và số dòng code
- **File logic**: `centralized-data-service/internal/service/master/transmuter.go` (Thay đổi khoảng 30 dòng code).
- **File test**: `centralized-data-service/internal/service/master/transmuter_orphan_test.go` (Thêm khoảng 120 dòng code cho test case `TestTransmuter_OrphanMasterChunking`).

## 2. Mô tả thay đổi (Overview)
- Loại bỏ safety gate kiểm tra cứng giới hạn kích thước mảng `onlySourceIDs` ở đầu hàm `Run` của `TransmuterModule`.
- Tự động chia mảng `onlySourceIDs` thành nhiều chunk có kích thước nhỏ hơn hoặc bằng `batchSize` (2000).
- Thực thi vòng lặp transmuter (gồm fetch shadow batch, check orphan, process batch, lưu checkpoint) lần lượt cho từng chunk.
- Tạo test case giả lập `batchSize = 2` bằng reflection/unsafe để đảm bảo tính chính xác của cơ chế chunking đối với các lô ID lớn.
- Chạy linter quy trình thành công, đảm bảo các tài liệu workspace được lưu vết vật lý đầy đủ.
