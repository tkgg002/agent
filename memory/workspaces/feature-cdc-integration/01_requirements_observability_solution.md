quản lý một "vũ trụ" gồm **105 items** mà không có một đài quan sát (Observability) tập trung thì chẳng khác nào đi trong sương mù mà không có đèn pha. 

Dưới đây là phương án triển khai chi tiết cho các yêu cầu **Observability P0** của bạn, được thiết kế để tương thích hoàn toàn với mô hình **Core/Agent** mà chúng ta đã thống nhất.

---

## R1: System Health Page — "The Single Source of Truth"

Trang này không chỉ hiển thị "Sống/Chết" (Liveness) mà phải hiển thị "Sức khỏe" (Readiness & Performance).

### Thành phần Dashboard (Suggested Layout)
* **Infrastructure Heartbeat:** Status của Kafka, NATS, Postgres, MongoDB, Redis.
* **CDC Pipeline Status:**
    * **Debezium Connector:** Chế độ (Running/Paused/Failed), Task count.
    * **Worker Metrics:** Consumer Lag (số lượng message đang chờ), Throughput (msg/sec).
* **Reconciliation Status (The "Core" View):**
    * Bảng tổng hợp: `Table Name | Source Count | Dest Count | Drift (%) | Last Recon`.
* **E2E Latency (R5):** Biểu đồ line chart theo thời gian thực.

---

## R2 & R3: Activity Log — Batch-based & Command Logging

Để tránh "Log Spam" (ghi quá nhiều làm nghẽn hệ thống), chúng ta sẽ áp dụng cơ chế **Aggregation tại Agent**.

### Luồng xử lý:
1.  **Kafka Consumer (Worker):** Thay vì log từng record, Worker sẽ tích lũy kết quả của một `batch` (ví dụ: 100 messages hoặc mỗi 5 giây).
    * *Entry:* `[2026-04-16 10:00] Topic: loyalty_points | Processed: 100 | Success: 98 | Failed: 2 (Link to DLQ) | Duration: 450ms`.
2.  **NATS Command:** Mọi lệnh từ Core gửi xuống Agent (ví dụ: `START_RECON`, `SYNC_RETRY`) phải được `publishResult` kèm theo metadata để hiển thị trên Activity Log.

---

## R4: Debezium & Kafka Connect Health

Chúng ta sẽ sử dụng một **Health-Check Agent** (hoặc tích hợp vào Core) để poll dữ liệu từ Kafka Connect REST API.

* **Endpoint:** `GET /connectors/{connector-name}/status`
* **Logic:**
    * Nếu `state == FAILED` → Bắn Alert qua Telegram/Slack ngay lập tức.
    * Nếu `state == PAUSED` → Hiển thị cảnh báo vàng trên Dashboard.
    * Tự động bóc tách `trace` lỗi từ API để hiển thị trực tiếp lên UI, giúp dev không cần ssh vào container xem log.

---

## R5: E2E Latency Measurement — "The Golden Signal"

Đây là chỉ số quan trọng nhất để biết hệ thống CDC có đang chạy "Near-realtime" hay không.

### Cách tính:
$$Latency = T_{postgres\_insert} - T_{kafka\_message\_timestamp}$$

* **T1 (Source):** Debezium tự động chèn timestamp lúc event xảy ra trong MongoDB vào metadata của Kafka message.
* **T2 (Dest):** Worker lấy `time.Now()` ngay khi thực hiện `UPSERT` thành công vào Postgres.
* **Metric:** Đẩy giá trị $(T2 - T1)$ vào Prometheus/SigNoz dưới dạng **Histogram** để xem được Percentiles ($P95, P99$).

---

## R6: Worker Log Persistence (SigNoz Integration)

Vì hệ thống của bạn đang hướng tới sự chuyên nghiệp, mình đề xuất luồng **OpenTelemetry (OTel)**:

1.  **Logger:** Sử dụng `uber-go/zap` trong Go Worker.
2.  **Exporter:** Cấu hình OTel zap core để gửi log trực tiếp qua gRPC tới **SigNoz OTEL Collector**.
3.  **Context:** Đảm bảo mỗi log đều kèm theo `trace_id` để khi check một bản ghi bị lỗi, bạn có thể trace ngược lại từ lúc nó ở Kafka cho đến khi vào Postgres.

---

## Tổng kết Mô hình Observability (Core/Agent)



| Layer | Responsibility | Tooling |
| :--- | :--- | :--- |
| **Agent (Worker)** | Thu thập Metrics, Batch Activity Log, Latency T1/T2. | Zap, OTel SDK |
| **Core (Master)** | Tổng hợp Recon status, Poll Kafka Connect API, Quản lý Metadata. | Go Core Service |
| **Storage** | Lưu trữ Log và Metrics lâu dài. | SigNoz (ClickHouse), Prometheus |
| **Presentation** | UI Page duy nhất cho User. | React/Next.js (FE page) |