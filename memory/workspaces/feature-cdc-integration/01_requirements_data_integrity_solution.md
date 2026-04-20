# Tài liệu Đặc tả: Đảm bảo Toàn vẹn Dữ liệu CDC (Data Integrity)

> **Ngày:** 16-04-2026  
> **Trạng thái:** Urgent (P0) - Đang có lệch dữ liệu  
> **Hệ thống:** MongoDB (Source) → Debezium → Kafka → Worker (Go) → Postgres (Dest)

---

## 1. Phân tích kịch bản sự cố và Cơ chế khôi phục

### 1.1. Khi CDC Worker (Consumer) bị die
Khi Worker dừng, Kafka vẫn tiếp tục nhận và lưu trữ message từ Debezium. 

* **Dữ liệu (Records):** Không bị mất. Khi Worker restart, nó sẽ đọc từ `committed offset` cuối cùng.
    * *Trường hợp Record bị "Miss":* Chỉ xảy ra nếu Worker đã commit offset lên Kafka nhưng logic ghi vào Postgres bị fail (do code lỗi, network timeout) mà không có cơ chế Retry/DLQ.
* **Thay đổi Schema:** Nếu trong lúc Worker die, source DB thêm field mới.
    * *Kết quả:* Message mới trong Kafka sẽ có cấu trúc mới. Khi Worker chạy lại, nếu code chưa cập nhật để map field mới này, Worker có thể bị crash (Panic) hoặc bỏ qua field đó, dẫn đến lệch dữ liệu giữa 2 bên.

### 1.2. Khi Debezium hoặc Kafka bị die
Đây là sự cố tầng hạ tầng (Infrastructure).

* **Cơ chế Oplog Retention:** MongoDB lưu các thay đổi trong Oplog. Debezium lưu vị trí đã đọc (offset) vào một topic riêng của Kafka hoặc file local.
* **Khi chạy lại:** Debezium sẽ tra cứu offset cũ và yêu cầu MongoDB gửi tiếp các bản ghi Oplog từ mốc đó.
* **Khi nào Record bị "Miss" thực sự?** * Nếu thời gian Debezium/Kafka die **dài hơn** thời gian lưu trữ của MongoDB Oplog (ví dụ: Oplog cấu hình lưu 24h nhưng hệ thống die 48h). 
    * *Hậu quả:* Debezium không tìm thấy điểm bắt đầu cũ (Oplog bị ghi đè). Hệ thống sẽ rơi vào trạng thái lỗi nghiêm trọng, buộc phải thực hiện **Re-snapshot** (quét lại toàn bộ bảng).
* **Schema Change:** Debezium có cơ chế "Schema History Topic". Nó sẽ cố gắng tái cấu trúc lại lịch sử thay đổi để gửi message đúng format.

---

## 2. Giải pháp Reconciliation (Core & Agent)

Để giải quyết tình trạng lệch dữ liệu hiện tại và ngăn ngừa trong tương lai, hệ thống cần triển khai theo mô hình **Reconciliation Service**.

### 2.1. Kiến trúc thành phần

1.  **Recon Core (Orchestrator):** * Quản lý cấu hình đối soát (bảng nào, tần suất nào).
    * Nhận báo cáo từ các Agent, so sánh kết quả và ra quyết định "Heal" (sửa lỗi).
    * Cung cấp API cho CMS Dashboard.
2.  **Recon Agent (Scanner):**
    * **Source Agent:** Chạy gần MongoDB. Quét dữ liệu theo batch, tính toán checksum/hash hoặc lấy danh sách ID.
    * **Dest Agent:** Chạy gần Postgres. Thực hiện truy vấn tương ứng để đối chiếu.
    * *Lợi ích:* Giảm tải truyền tải dữ liệu thô qua network, chỉ truyền kết quả hash/count.

### 2.2. Chiến lược đối soát (Tiered Approach)

| Cấp độ | Phương pháp | Cách xử lý khi lệch (Action) |
| :--- | :--- | :--- |
| **Tier 1 (Fast)** | **Count Check:** So sánh `countDocuments()` và `SELECT COUNT(*)`. | Gửi Alert tới CMS, trigger Tier 2. |
| **Tier 2 (Medium)** | **ID Set/Boundary Check:** Kiểm tra theo từng dải ID (ví dụ 10.000 record một lần). | Tìm ra chính xác dải ID nào đang thiếu bản ghi. |
| **Tier 3 (Deep)** | **Field Hash:** Tính MD5/SHA cho toàn bộ các field quan trọng của bản ghi. | Phát hiện các bản ghi đủ số lượng nhưng sai lệch nội dung (Stale data). |

---

## 3. Plan xử lý lệch dữ liệu (Action Plan)

Hiện tại hệ thống đang bị lệch, cần thực hiện ngay các bước sau:

### Bước 1: Giám sát và Cô lập (Monitoring)
* Kiểm tra `Consumer Lag` trên Kafka để xem Worker có đang xử lý kịp không.
* Kiểm tra Error Log của Worker để tìm các bản ghi bị lỗi Schema không insert được.

### Bước 2: Triển khai Recon Agent tạm thời
* Chạy script lấy toàn bộ `_id` từ MongoDB và `id` từ Postgres của bảng đang lệch.
* Dùng phép trừ tập hợp để tìm ra danh sách các ID bị thiếu (Missing IDs).

### Bước 3: Cơ chế Tự chữa lành (Auto-healing)
* **Manual Trigger:** Từ CMS, cung cấp nút "Re-sync IDs". Khi nhấn, Core sẽ gửi danh sách ID thiếu cho một **Repair Worker**.
* **Repair Worker:** Sẽ trực tiếp truy vấn MongoDB bằng ID đó và thực hiện `UPSERT` vào Postgres, không đi qua luồng Kafka để tránh làm chậm stream hiện tại.

### Bước 4: Dashboard báo cáo (CMS UI)
* **Màn hình tổng quan:** Hiển thị danh sách các bảng, trạng thái (Matched/Drifted), số lượng lệch.
* **Màn hình chi tiết:** Xem danh sách các record bị lỗi, lý do lỗi (Schema mismatch, Timeout...).
* **Công cụ điều khiển:** Reset offset cho Debezium, Trigger snapshot cho 1 bảng cụ thể.

---

## 4. Đặc tả kỹ thuật cho Worker (Core Master)

Để đảm bảo không lặp lại tình trạng này, Worker cần được bổ sung:

1.  **Idempotency:** Luôn luôn dùng `ON CONFLICT (id) DO UPDATE` để có thể replay dữ liệu bất cứ lúc nào mà không sợ trùng lặp.
2.  **Dead Letter Queue (DLQ):** Khi một record lỗi schema, không được để Worker crash. Hãy ghi record đó vào một bảng `failed_sync_logs` trong Postgres kèm theo nội dung JSON lỗi.
3.  **Observability:** Tích hợp Prometheus để theo dõi số lượng `success_sync_total` và `failed_sync_total`.