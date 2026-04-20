ở mức độ "over" (chuyên sâu và bao quát mọi ngóc ngách), chúng ta cần nhìn vào những góc khuất mà các hệ thống CDC thường bỏ qua. Dưới đây là phân tích các kịch bản "thảm họa" tiềm ẩn và kiến trúc **Core/Agent** hoàn chỉnh để xử lý triệt để.

---

## 1. Deep Dive: Những Case "Tử Thần" Thường Bị Bỏ Sót

Ngoài việc Server die hay Kafka die, đây là những trường hợp thực tế gây lệch dữ liệu âm thầm:

### A. Case "Oplog Time Travel" (Clock Skew)
* **Kịch bản:** Cluster MongoDB có sự lệch thời gian giữa các node hoặc Debezium bị lệch clock so với DB.
* **Hậu quả:** Debezium có thể bỏ qua một số sự kiện vì nghĩ rằng chúng thuộc về "tương lai" hoặc đã quá cũ so với cursor hiện tại.
* **Kết quả:** Data bị hổng một vài giây, cực kỳ khó phát hiện bằng mắt thường.

### B. Case "The Silent Truncation" (Data Type Mismatch)
* **Kịch bản:** MongoDB là schema-less. Một field trong Mongo đột ngột chứa string cực dài hoặc một Object phức tạp mà cột Postgres (ví dụ `VARCHAR(255)`) không chứa nổi.
* **Hậu quả:** Worker bị lỗi tại bản ghi đó. Nếu không có **DLQ (Dead Letter Queue)**, Worker có thể skip bản ghi đó để chạy tiếp hoặc đứng hình (crash loop).
* **Kết quả:** Thiếu đúng bản ghi đó ở Postgres.

### C. Case "Ghost Schema Update" (In-place updates)
* **Kịch bản:** Bạn dùng `db.collection.updateMany({}, { $set: { new_field: 1 } })` trên triệu bản ghi MongoDB.
* **Hậu quả:** Oplog bùng nổ (Spike). Debezium không xử lý kịp, dẫn đến **Consumer Lag** cực lớn. Trong lúc đó, Kafka có thể chạm ngưỡng `retention.ms` hoặc `retention.bytes` và tự động xóa bớt dữ liệu cũ trước khi Worker kịp đọc.
* **Kết quả:** Mất sạch một dải dữ liệu lớn do Kafka dọn dẹp (Cleanup policy).

### D. Case "Race Condition" giữa Recon và CDC
* **Kịch bản:** Hệ thống Recon đang "heal" (ghi đè) một bản ghi cũ, đúng lúc đó CDC lại đẩy về một bản ghi mới nhất của cùng ID đó.
* **Hậu quả:** Nếu không có cơ chế so sánh version/timestamp, bản ghi "cũ" từ Recon có thể ghi đè bản ghi "mới" từ CDC.
* **Kết quả:** Dữ liệu ở Postgres bị lùi về quá khứ (Stale data).

---

## 2. Kiến trúc Core / Agent cho Reconciliation (Advanced)

Để giải quyết đống "thảm họa" trên, hệ thống cần được thiết kế theo mô hình Master-Worker (Core/Agent).

### 2.1. Phân vai
* **Recon-Agent (The Prober):** Triển khai dưới dạng Sidecar hoặc Service chạy gần Database.
    * **Source Agent (Mongo):** Quét Oplog và Data theo chunk (ví dụ: 10,000 IDs mỗi lần). Tính toán **Merkle Tree Hash** cho từng chunk.
    * **Dest Agent (Postgres):** Làm tương tự với Postgres.
* **Recon-Core (The Brain):** * So sánh Hash từ 2 Agent gửi về. 
    * Nếu Hash lệch, yêu cầu Agent gửi chi tiết danh sách ID trong chunk đó.
    * Điều phối việc "Heal" dữ liệu.



### 2.2. Cơ chế "Heal" dữ liệu thông minh
Thay vì chỉ `Insert` lại, Core sẽ thực hiện:
1.  **Lấy Version:** Query `ts` (timestamp) của record từ Mongo và `last_updated` từ Postgres.
2.  **So sánh Version:** Chỉ `UPSERT` nếu dữ liệu từ Mongo có timestamp **lớn hơn** dữ liệu đang có ở Postgres.
3.  **Audit Log:** Ghi lại mọi hành động "Heal" để tracking xem tại sao dữ liệu bị lệch (Do schema? Do downtime?).

---

## 3. Plan xử lý lệch dữ liệu hiện tại (Atomic Action Plan)

Vì bạn đang bị lệch dữ liệu (P0), hãy thực hiện theo thứ tự này:

### Bước 1: Scan & Detect (Agent Task)
* Dùng Agent quét theo dải ID hoặc `created_at`. 
* **Priority:** Quét các bản ghi được tạo/cập nhật trong khoảng thời gian nghi ngờ hệ thống die trước.
* Xuất ra file/table: `mismatched_ids_report`.

### Bước 2: Deep Re-sync (Core Task)
* Core đọc từ `mismatched_ids_report`.
* Gọi lệnh `ad-hoc snapshot` cho Debezium (nếu dùng Debezium 1.9+) bằng cách ghi signal vào bảng `debezium_signal`.
* *Lợi ích:* Debezium tự quét lại đúng những bản ghi đó và đẩy qua Kafka, đảm bảo mọi logic Transformer/Worker đều được tái sử dụng.

### Bước 3: Đấu nối DLQ & Alerting
* Cấu hình Worker: Mọi bản ghi lỗi (do Schema, Type...) phải bắn vào bảng `worker_dlq`.
* Xây dựng CMS xem `worker_dlq`: Admin có thể sửa mapping hoặc tăng size cột Postgres trực tiếp rồi nhấn "Retry".

### Bước 4: Chặn đứng Schema Drift
* Cài đặt **Schema Registry** (Confluent hoặc Apicurio).
* Worker sẽ check version schema của message. Nếu message có version lạ -> Stop và báo động thay vì xử lý sai.

---

## 4. Bảng Tổng kết "Full-stack" Resilience

| Thành phần | Case lỗi | Giải pháp "Over" |
| :--- | :--- | :--- |
| **Source DB** | Oplog Overload | Monitor Oplog Size & Retention; Tăng I/O trong lúc migration. |
| **Debezium** | Offset Inconsistency | Snapshot Signal Table (Re-sync không cần restart). |
| **Kafka** | Log Deletion (Retention) | Cấu hình `cleanup.policy=compact` thay vì `delete` cho các topic quan trọng. |
| **Worker** | Silent Crash / Logic Error | **Idempotent Write** + **DLQ** + **Prometheus Metrics**. |
| **Destination** | Schema Incompatibility | **Schema Evolution** (Auto-alter table) + **Validation Agent**. |
| **Toàn hệ thống** | Lệch dữ liệu tổng thể | **Core/Agent Reconciliation** với **Merkle Tree Hash**. |