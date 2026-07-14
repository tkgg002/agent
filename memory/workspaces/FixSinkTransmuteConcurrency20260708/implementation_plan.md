# Tối ưu hóa Kiến trúc Concurrency & Batching cho Sink và Transmute Worker dưới tải cao (5000 msg/s)

Tài liệu này đề xuất kế hoạch triển khai nâng cấp hệ thống CDC, giải quyết triệt để **12 lỗ hổng phân tán tiềm ẩn (issues)** được phát hiện từ quá trình Red Teaming tại tầng ghi Shadow DB (Sink Worker) và Master DB (Transmute Worker) khi vận hành ở quy mô tải thực tế (200 tables + burst 5000 events/giây).

---

## 12 Lỗ hổng & Giải pháp Gia cố (Red Teaming Hardening)

### 1. Issue 1: Lỗi "Tự sát chùm" tại Sink Worker do `errgroup.WithContext`
*   **Vấn đề:** Nếu 1 bảng gặp lỗi SQL làm hủy context chung, `errgroup` sẽ tự động cancel các bảng chạy song song còn lại, dẫn đến rollback hàng loạt các bảng bình thường.
*   **Giải pháp:** Sử dụng `errgroup.Group` thuần (không WithContext). Các goroutine con luôn trả về `nil` và thu thập lỗi độc lập thông qua khóa Mutex.

### 2. Issue 2: Cơn ác mộng Fallback tuần tự khi DB sập kết nối / Mạng lỗi
*   **Vấn đề:** Khi DB chết, Worker lùi về chạy tuần tự 500 lần ghi đơn, gây nghẽn luồng và cạn kiệt CPU/RAM do timeout dồn dập.
*   **Giải pháp:** Phân loại lỗi bằng hàm `isTransientError(err)`. Nếu lỗi mạng/DB kết nối, lập tức gọi `Nak()` toàn bộ lô để JetStream gửi lại sau (Fail-Fast). Chỉ fallback khi lỗi dữ liệu (Poison Pill).

### 3. Issue 3: NATS Redelivery Race Condition do quá hạn AckWait
*   **Vấn đề:** Quá trình xử lý tuần tự lỗi Poison Pill kéo dài vượt quá `AckWait` (10s), NATS Server giao lại lô tin nhắn cho worker thứ hai gây tranh chấp khóa (`Row Lock Contention`).
*   **Giải pháp:** Trong luồng chạy fallback tuần tự/đệ quy, định kỳ gọi `msg.InProgress()` để thông báo NATS kéo dài thời hạn AckWait.

### 4. Issue 4: Rủi ro tràn RAM (OOM) tại TableDebouncer
*   **Vấn đề:** NATS Pull Fetch tin nhắn nhanh hơn tốc độ DB tiêu thụ, làm phình to đệm RAM gây OOM.
*   **Giải pháp:** Tích hợp Backpressure vào hàm `Add()`: Nếu kích thước hàng chờ của một bảng vượt quá `maxSize * 2`, tạm giải phóng lock Mutex và sleep ngắn để hãm tốc độ Fetch của NATS Pull.

### 5. Issue 5: Chiến lược chống Batch Dilution có thể phản tác dụng
*   **Vấn đề:** Tải 5000 msg/s chia cho 200 bảng -> trung bình chỉ 25 msg/s một bảng. Cấu hình batch 5000 và timeout 200ms vô tình tạo độ trễ 200ms giả lập mà lô ghi thu được vẫn rất bé (chỉ ~5 dòng).
*   **Giải pháp:** Cấu hình batch size vừa phải (`500 - 1000`), timeout thấp (`50ms`), tối ưu bằng **Multi-row INSERT (Upsert)** cho các lô nhỏ để giảm tối đa kết nối DB.

### 6. Issue 6: Trùng lặp dữ liệu do thiếu tính Idempotency khi Late ACK
*   **Vấn đề:** Worker bị crash sau khi DB Commit nhưng trước khi kịp gọi `Ack()`. NATS sẽ redeliver tin nhắn gây trùng lặp bản ghi nếu thao tác không tự nhiên idempotent.
*   **Giải pháp:** Bắt buộc áp dụng cú pháp **Upsert Idempotent** (`INSERT ... ON CONFLICT (primary_key) DO UPDATE SET ...`) để đảm bảo tính nhất quán của Master DB.

### 7. Issue 7: Trễ tích lũy cho bảng thưa sự kiện do Debounce Timeout cố định
*   **Vấn đề:** Đệm debounce cố định 1s sẽ giữ chân một sự kiện đơn lẻ của một bảng thưa trong suốt 1s đó.
*   **Giải pháp:** Chuyển sang cơ chế **Idle Debounce** (Flush after idle X ms, hoặc tổng thời gian từ sự kiện đầu tiên đạt ngưỡng tối đa thì flush).

### 8. Issue 8: Fallback tuần tự Poison Pill quá chậm ($O(N)$)
*   **Vấn đề:** Gặp poison pill trong mẻ 500 records phải chạy 500 câu lệnh đơn gây sập hiệu năng.
*   **Giải pháp:** Thay thế bằng thuật toán **Chia để trị (Binary Search Split)** độ phức tạp $O(\log N)$ (khoảng 18 truy vấn cho mẻ 500 bản ghi), giảm 96.4% số lượng roundtrips DB.

### 9. Issue 9: Trôi AckWait do tin nhắn bị kẹt ở hàng chờ Semaphore
*   **Vấn đề:** Gọi Fetch tin nhắn ồ ạt nhưng bị nghẽn ở bước tranh chấp Semaphore, trôi AckWait khiến NATS gửi lại bản sao trùng lặp.
*   **Giải pháp:** Hãm phanh đầu vào NATS bằng cấu hình `MaxAckPending` kết hợp điều phối Fetch tương thích với slot Semaphore trống.

### 10. Issue 10: Cạn kiệt Postgres Connections khi scale ngang
*   **Vấn đề:** Nhiều instances nhân bản làm tăng đột biến kết nối đồng thời, vượt quá `max_connections` của Postgres.
*   **Giải pháp:** Giới hạn pool kết nối theo công thức và khuyến nghị triển khai **PgBouncer** ở chế độ Transaction Pooling cho Master DB.

### 11. Issue 11: Mất thứ tự sự kiện (Event Ordering)
*   **Vấn đề:** Xử lý song song nhiều lô có thể làm đảo lộn thứ tự ghi (ví dụ INSERT chạy sau UPDATE).
*   **Giải pháp:** Hashing partition theo khóa chính trên NATS Stream để đưa vào cùng một consumer FIFO duy nhất; kết hợp điều kiện version check/timestamp `WHERE EXCLUDED.updated_at > master.updated_at` trong Upsert SQL.

### 12. Issue 12: Cấu hình AckWait và MaxDeliver chưa tối ưu
*   **Vấn đề:** AckWait quá ngắn (30s) khi DB chậm làm bùng phát tin nhắn trùng lặp.
*   **Giải pháp:** Nâng cấu hình Stream Consumer: `AckWait = 60s`, `MaxDeliver = 5`.

---

## Proposed Changes

### 1. Component: Shadow Sink Layer (`internal/handler/shadow`)
#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
*   Song song hóa Flush 20 bảng bằng `errgroup.Group` thuần (luôn return `nil` trong hàm con để tránh hủy chùm).
*   Giảm batch size xuống 500 - 1000, timeout 50ms và tối ưu bằng Multi-row Upsert.

### 2. Component: Master Transmute Layer (`internal/handler/master`)
#### [MODIFY] [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
*   Tích hợp Concurrency Semaphore.
#### [NEW] [debounce.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/debounce.go)
*   Xây dựng bộ đệm `TableDebouncer` hỗ trợ: Idle Debounce, Backpressure hãm phanh RAM, chia để trị Binary Search Split cô lập Poison Pill, kiểm tra `isTransientError`, và báo cáo `InProgress` về NATS.
#### [MODIFY] [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go)
*   Nâng cấp NATS sang JetStream Pull Subscription với Manual ACK và `MaxAckPending` cấu hình hợp lý.

### 3. Component: System Configurations
#### [MODIFY] [config.yaml](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/configs/config.yaml)
*   Cấu hình tham số debounce timeout, max size và concurrency limit.

---

## Verification Plan

### Automated Tests
*   **TestBatchBufferParallelFlush:** Giả lập độ trễ DB và xác minh flush song song giảm latency đáng kể.
*   **TestPoisonPillBinarySearch:** Giả lập 1 bản ghi lỗi trong mẻ 500 bản ghi, verify thuật toán chia để trị tìm và cô lập đúng bản ghi đó trong vòng dưới 20 truy vấn DB.

### Manual Verification
*   Giả lập tải burst 5000 msg/s qua Kafka, giám sát kết nối Postgres và Consumer Lag trên SigNoz/Prometheus.
