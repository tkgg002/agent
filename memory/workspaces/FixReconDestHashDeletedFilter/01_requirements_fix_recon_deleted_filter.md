# Yêu cầu kỹ thuật - Sửa lỗi thiếu filter _deleted và lệch múi giờ trong đối soát

## 1. Hiện trạng & Vấn đề
* **Vấn đề 1: Thiếu filter _deleted**
  Các truy vấn trong `HashWindow` và `BucketHash` của `ReconDestAgent` không loại bỏ các dòng đã xóa mềm (`_deleted = true`). Bản sửa lỗi trước đã giải quyết phần này.
* **Vấn đề 2: Lệch múi giờ khi đối soát cột TIMESTAMP (without timezone) (Bug mới phát hiện)**
  Bảng `payment_bills` lưu trường thời gian `last_updated_at` dưới kiểu dữ liệu `TIMESTAMP` (không có múi giờ). Khi chạy đối soát:
  1. **Lệch dải query:** Trong `resolvePostgresTimeParams`, khi so sánh cột `TIMESTAMP` với giá trị `time.Time` dạng UTC do Go truyền vào, Postgres tự động cast cột `TIMESTAMP` sang `TIMESTAMPTZ` bằng cách giả định múi giờ local của DB server (e.g. +07:00). Điều này làm dải quét bị dịch chuyển 7 tiếng, khiến Postgres quét sai khoảng thời gian so với MongoDB.
  2. **Lệch giá trị XOR hash:** Khi pgx/GORM quét giá trị `TIMESTAMP` từ Postgres vào biến `time.Time`, nó mặc định gán múi giờ local của tiến trình Go (e.g., `+07:00`). Điều này khiến hàm `ts.UnixMilli()` bị trừ đi offset 7 tiếng, trong khi MongoDB lưu ở dạng UTC. Do đó, mã băm XOR của mọi record trong mọi cửa sổ đều bị lệch nhau hoàn toàn, dẫn đến việc 100% cửa sổ đối soát đều bị báo lệch giả (`false drift`) và nhảy vào `drift_drill_down`.

## 2. Yêu cầu (Definition of Done)
* **Xử lý lệch dải query:** Cập nhật `resolvePostgresTimeParams` để trả về chuỗi timestamp không múi giờ (e.g., `2006-01-02 15:04:05.000000`) đối với các cột kiểu `timestamp without time zone` để Postgres so sánh trực tiếp dạng wall-clock.
* **Xử lý lệch giá trị XOR hash:**
  * Cập nhật `parsePostgresTimestamp` để nếu giá trị đầu vào là `time.Time` hoặc `*time.Time` và có location khác `UTC`, nó sẽ chuyển đổi wall-clock time đó sang `UTC` trực tiếp.
  * Cập nhật hàm `HashWindow` của `ReconDestAgent` để đi qua `parsePostgresTimestamp` trước khi lấy `UnixMilli()`.
* Đảm bảo tất cả các unit/integration test trong `internal/service/recon` đều pass sạch sẽ.
