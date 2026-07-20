# Yêu cầu kỹ thuật: Sửa lỗi Timezone Drift trong Recon Pipeline

## 1. Bối cảnh
Trong hệ thống đối soát (`recon` service), việc so sánh dữ liệu giữa MongoDB (Source) và PostgreSQL (Shadow/Destination) được thực hiện bằng cách so sánh số lượng bản ghi và giá trị băm XOR vân tay bản ghi (`HashWindow` check).
Vân tay của bản ghi được tính toán bằng hàm `hashIDPlusTsMs` sử dụng ID và thời gian cập nhật của bản ghi quy đổi ra epoch milliseconds (`UnixMilli`).

## 2. Vấn đề phát hiện
Hàm `parsePostgresTimestamp` trong file `internal/service/recon/recon_query.go` khi nhận một giá trị kiểu `time.Time` từ PostgreSQL thông qua driver:
- Nếu location của `time.Time` không phải là `time.UTC` (ví dụ: Local timezone `Asia/Ho_Chi_Minh` +07:00), hàm này thực hiện hành vi "timezone shift":
  `time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), time.UTC)`
- Hành vi này chuyển đổi giờ local (ví dụ: `11:48:00 +07:00`) trực tiếp thành giờ UTC với các giá trị trường tương ứng (thành `11:48:00 UTC`), tức là làm tăng thời gian lên đúng 7 tiếng so với thời điểm vật lý thực tế (`04:48:00 UTC`).
- Kết quả là mốc `UnixMilli` tính toán bị sai lệch 7 tiếng, làm cho XOR Hash của Postgres Shadow bị tính toán sai hoàn toàn và dẫn đến báo cáo "drift" giả mạo (False Positive).

## 3. Mục tiêu & Yêu cầu
- Sửa hàm `parsePostgresTimestamp` trong file [recon_query.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_query.go) để chuyển đổi kiểu `time.Time` sang múi giờ UTC một cách chuẩn xác bằng cách gọi `.UTC()` thay vì shift timezone.
- Bổ sung kiểm thử tích hợp (Unit Test) trong [recon_postgres_source_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_postgres_source_test.go) để kiểm chứng trường hợp truyền `time.Time` có múi giờ Local thì hàm phải trả về đúng mốc giờ UTC tương đương về mặt vật lý.
- Đảm bảo tất cả các test case trong package `recon` tiếp tục hoạt động chính xác.
