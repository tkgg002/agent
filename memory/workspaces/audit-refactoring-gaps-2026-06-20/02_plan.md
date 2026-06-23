# Kế hoạch Audit logic toàn diện: centralized-data-service

Kế hoạch rà soát và đối chiếu logic nghiệp vụ của `centralized-data-service` giữa bản backup gốc (`data-hub-bf/centralized-data-service`) và bản hiện tại (`data-hub/centralized-data-service`).

## Giai đoạn 1: Đối chiếu trực tiếp logic nghiệp vụ của các cấu phần chính

### 1. Provisioning & Table Mapping
- Đối chiếu logic nhận lệnh, khởi tạo bảng shadow/master, tạo indexes, mapping cột dữ liệu.
- Rà soát file `provisioning_handler.go` và các helper liên quan so với file gốc trong bản backup.

### 2. Swap Table (DDL)
- Đối chiếu logic swap table vật lý (BEGIN + RENAME + COMMIT) giữa bản gốc và bản hiện tại.
- Đảm bảo tính nguyên tử (atomic) và xử lý lỗi khi swap table không bị sai lệch.

### 3. Batch Transform & Data transformation
- Đối chiếu logic transform dữ liệu từ shadow sang master (BatchTransform).
- Kiểm tra cách xử lý các kiểu dữ liệu đặc biệt, logic watermark và chunking dữ liệu.

### 4. Reconciliation & Lag monitoring
- Đối chiếu logic đo lag (Lag monitoring) và so khớp dữ liệu (Reconciliation).
- Đảm bảo các chỉ số lag được tính toán và lưu trữ đúng cấu trúc cũ.

## Giai đoạn 2: Khôi phục Logic Nghiệp vụ & Sửa đổi
1. Sửa đổi trực tiếp các file logic bị gãy để khôi phục 100% logic nghiệp vụ cũ, giữ nguyên cấu trúc thư mục mới.
2. Kiểm tra biên dịch dự án: `go build ./...`

## Giai đoạn 3: Kiểm thử và Xác nhận
1. Chạy toàn bộ test suite của `centralized-data-service`: `go test ./...`
