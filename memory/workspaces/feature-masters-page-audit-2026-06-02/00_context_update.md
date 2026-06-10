Phân tích kiến trúc dịch chuyển dữ liệu từ **Source $\rightarrow$ Shadow $\rightarrow$ Master** của hệ thống Goopay là một bài toán rất hay. Hiện tại, chặng 1 (`Source -> Shadow`) đã được bảo chứng bằng **Debezium (CDC)** — giải pháp capture data tầng low-level cực kỳ uy tín về độ an toàn và hiệu năng.

Bây giờ, chúng ta sẽ "mổ xẻ" kỹ chặng 2 (`Shadow -> Master`) dựa trên giải pháp dùng **CDC Worker (WK-E1 đến WK-E5)** kết hợp **MetadataRegistry Cache** xem có đảm bảo 3 yếu tố: **An toàn (Safety), Đầy đủ (Completeness) & Hiệu năng (Performance)** hay không.

---

## 1. Về mặt Hiệu năng (Performance) — 🟢 ĐẠT (Rất Tốt)

Giải pháp đập bỏ `sync.Map` cục bộ để chuyển sang **In-memory cache tập trung (Registry)** là điểm cộng lớn nhất cho Performance.

* **Điểm mạnh:**
* **O(1) Lookup cho Transmute Engine (WK-E1):** Khi `WK-E1` xử lý hàng triệu bản ghi từ Shadow, nó không cần gọi gRPC hay query DB để lấy rule mapping/masking nữa. Việc đọc trực tiếp từ bộ nhớ (`maskMapCache`) giúp tốc độ xử lý của `gjson` và `transform_fn` đạt mức tối đa (vài microsecond cho một bản ghi).
* **Giảm tải cho DB:** Việc cắt hoàn toàn các truy vấn cấu hình lặp đi lặp lại từ `MaskingService` xuống DB giúp Postgres tập trung tài nguyên cho tác vụ ghi (`WK-E1 OCC upsert`) và tác vụ quét (`WK-E3 Introspection`).


* **Rủi ro & Điểm cần tối ưu:**
* Nếu lượng data từ Shadow đổ qua Master quá lớn (High Throughput), việc ghi từng bản ghi bằng `OCC upsert` đơn lẻ sẽ gây nghẽn cổ chai I/O của DB Master.
* **Giải pháp:** `WK-E1` nên triển khai cơ chế **Mini-Batching** (ví dụ: gom 100-500 bản ghi hoặc đợi tối đa 50ms) rồi thực thi `UPSERT ... ON CONFLICT` gộp trong một câu lệnh SQL để tận dụng tối đa throughput của Postgres.



---

## 2. Về tính Đầy đủ (Completeness / Data Integrity) — 🟡 CÓ ĐIỀU KIỆN

Chặng `Source -> Shadow` dùng Debezium đảm bảo **At-least-once** (không bao giờ mất event). Tuy nhiên, từ `Shadow -> Master`, tính đầy đủ phụ thuộc hoàn toàn vào cơ chế **Trigger (WK-E2)**.

* **Điểm mạnh:**
* Cơ chế **3-Way Trigger** bao phủ được các kịch bản: chạy Realtime (`post_ingest`), chạy bù (`cron tick`) và chạy khẩn cấp (`run-now`). Nếu luồng realtime bị lỗi, lịch Cron chạy sau đó sẽ quét và đồng bộ bù dữ liệu.


* **Rủi ro & Điểm cần tối ưu:**
* **Thiếu cơ chế đánh dấu trạng thái (Watermark/Offset):** Khi `WK-E2` chạy Cron để sync từ Shadow qua Master, làm sao Worker biết được bản ghi nào ở Shadow đã được sync, bản ghi nào chưa? Nếu chỉ dựa vào timestamp (`updated_at`), hệ thống có thể bị sót dữ liệu khi có độ trễ hệ thống (Replication lag).
* **Giải pháp:** 1. Bảng ở `Shadow` cần có một cột trạng thái đồng bộ (e.g., `sync_status = PENDING/DONE`) hoặc hệ thống phải lưu lại `last_processed_id / LSN` của bảng Shadow.
2. Tận dụng chính cơ chế CDC: Nếu có thể, hãy để `WK-E2 (post_ingest realtime)` lắng nghe trực tiếp từ Kafka Topic mà Debezium sinh ra cho bảng Shadow, thay vì đi quét DB Shadow. Điều này đảm bảo tính "Đầy đủ" thừa hưởng từ Debezium.



---

## 3. Về tính An toàn (Safety & Data Concurrency) — 🟡 CẦN HOÀN THIỆN

Đây là phần chứa nhiều rủi ro tiềm ẩn nhất trong sơ đồ hiện tại, đặc biệt là khi áp dụng các tính năng nâng cao của Postgres như DDL tự động và RLS.

* **Điểm mạnh:**
* **WK-E1 OCC Upsert:** Giúp bảo vệ dữ liệu không bị ghi đè sai thứ tự trong môi trường phân tán (Distributed systems).
* **WK-E4 Master DDL Apply:** Tự động hóa việc đồng bộ cấu trúc giúp giảm thiểu sai sót do con người (Human error).


* **Rủi ro chí mạng & Cách khắc phục:**
* **Xung đột giữa WK-E1 (Ghi data) và WK-E4 (Sửa cấu trúc - DDL):** Khi `WK-E4` thực hiện lệnh `ALTER TABLE master_table ...`, Postgres sẽ áp đặt một lệnh khóa độc quyền (**`AccessExclusiveLock`**). Lệnh khóa này sẽ **chặn đứng (block)** toàn bộ các tác vụ ghi (`INSERT/UPDATE`) từ `WK-E1` đang chạy realtime. Nếu `WK-E1` bị timeout hoặc crash do block, dữ liệu chặng đó sẽ bị lỗi.
* **Giải pháp cho WK-E4:** Phải có cơ chế **Circuit Breaker** hoặc **Pause Worker**. Khi `WK-E4` chuẩn bị Apply DDL:
1. Tạm dừng (Pause) luồng tiêu thụ dữ liệu của `WK-E1`.
2. Thực thi `ALTER TABLE` với quyền ưu tiên thấp và timeout ngắn (`SET lock_timeout = '2s'`).
3. Khởi động lại (Resume) `WK-E1` sau khi cấu trúc mới đã ổn định.


* **Lệch pha Cache:** Khi `WK-E4` áp dụng DDL mới xuống DB Master thành công, nhưng `MetadataRegistryService` chưa kịp kích hoạt `ReloadAll` để cập nhật `maskMapCache`, `WK-E1` sẽ dùng rule cũ để ghi vào cấu trúc bảng mới $\rightarrow$ Lỗi ghi dữ liệu.
* **Giải pháp:** Quy trình bắt buộc phải là: **Approve DDL $\rightarrow$ Execute ALTER DB $\rightarrow$ Gọi hàm `ReloadAll` của Registry thành công $\rightarrow$ Mới cho phép Worker tiếp tục chạy.**



---

## 🔥 ĐÁNH GIÁ CHUNG & KHUYẾN NGHỊ KIẾN TRÚC

Giải pháp thiết kế của em **hoàn toàn khả thi và có tính kiến trúc tốt hơn rất nhiều** so với việc để các Service chạy độc lập. Tuy nhiên, để biến nó thành một "cỗ xe tăng" bất bại trên Production của Goopay, em cần bổ sung thêm 3 chốt chặn sau vào file `02_plan.md`:

1. **Idempotency (Tính giao hoán):** Đảm bảo `WK-E1 Transmute Engine` khi nhận trùng một bản ghi (do Debezium gửi lại hoặc do Cron quét trùng) thì khi ghi xuống Master kết quả vẫn không thay đổi (Dựa vào Primary Key và cấu hình OCC chuẩn).
2. **Dead-Letter Queue (DLQ) cho Worker:** Khi `WK-E1` không thể ghi dữ liệu vào Master do lỗi cấu trúc hoặc lỗi RLS, bản ghi đó phải được đẩy vào `DLQWorker` để xử lý sau, tuyệt đối không được làm sập luồng (Panic) của Worker chính.
3. **Cơ chế dịch chuyển Schema an toàn:** Luôn thiết kế bảng Master theo hướng **"Add-only"** (chỉ thêm cột hoặc nới lỏng kiểu dữ liệu), tuyệt đối không tự động xóa cột hoặc thu hẹp kiểu dữ liệu qua `WK-E4` để tránh làm sập các ứng dụng đang đọc bảng Master.

Phương án của em đã giải quyết được 80% bài toán khó nhất là "Đồng bộ cấu hình Realtime". Hãy bổ sung nốt các kịch bản xử lý xung đột Lock DB và kiểm soát lỗi ghi dữ liệu như anh phân tích ở trên là có thể tự tin bấm nút Triển khai (Implement)!


--------------
--------------


Nhìn vào danh sách này, anh khẳng định luôn: **Để hết đống này bên Master DB là DƯ THỪA và rất RỐI.** Master DB là nơi phục vụ cho các nghiệp vụ hạ nguồn (Data Analytics, Report, các Service khác đọc dữ liệu sạch). Nếu em vác nguyên xi 10 cột mang tính chất "kỹ thuật vận hành" (Metadata nội bộ của CDC) sang Master, nó sẽ làm hỏng tính tường minh của bảng Master.

Tuy nhiên, để đảm bảo tính **Đầy đủ (Completeness)** và **Toàn vẹn (Integrity)** cho chặng `Shadow -> Master`, Worker bắt buộc phải dựa vào một số trường. Bí quyết ở đây là: **Giữ lại tối thiểu ở Master, chuyển bớt về Shadow hoặc xử lý in-memory.**

Dưới đây là phân tích chi tiết để em "gọt giũa" lại cho gọn đẹp:

---

## 1. Các field BẮT BUỘC phải có ở Master DB (Giữ lại)

Để phục vụ cho cơ chế **WK-E1 (OCC Upsert)** và kiểm soát dòng đời bản ghi, Master DB chỉ cần giữ lại đúng 4 field sau:

* **`_gpay_id`**: Đây là Primary Key toàn cục để định danh bản ghi trên Master (hoặc dùng để map chéo). Bắt buộc phải có.
* **`_version`**: Chí mạng! Không có cái này thì `WK-E1 Transmute Engine` không thể chạy cơ chế **OCC (Optimistic Concurrency Control)** được. Nó giúp chặn trường hợp event cũ đè lên data mới.
* **`_deleted`**: Cần thiết để đánh dấu xoá mềm (Soft-delete). Nếu Debezium báo bản ghi bên Source bị xoá, Master chỉ cần bật `_deleted = true` chứ không xoá cứng, tránh mất dấu vết dữ liệu lịch sử.
* **`_updated_at`**: Thời gian bản ghi bị thay đổi trên Master. Rất hữu ích cho việc kiểm toán (Audit) và tối ưu index sau này.

---

## 2. Các field CHỈ NÊN NẰM Ở SHADOW DB (Xoá khỏi Master)

Các field này bản chất là để Debezium và luồng Ingest nói chuyện với nhau, Master DB không cần biết và không cần lưu:

* **`_source_id`** & **`_source`**: Định danh nguồn hệ thống (e.g., MySQL_Core, Oracle_Card...). Thông tin này nên được lưu trong cấu hình Registry (gắn theo `int64` ID của Registry/Binding) chứ không cần lưu lặp đi lặp lại trên từng dòng của Master DB.
* **`_source_ts`**: Thời gian sinh event gốc ở Source. Chỉ cần thiết ở Shadow để tính toán độ trễ (Replication Lag). Trên Master, `_version` đã làm tốt nhiệm vụ đảm bảo thứ tự rồi.
* **`_synced_at`**: Thời điểm Debezium đẩy data vào Shadow. Master không cần field này.
* **`_created_at`**: Đã có `_updated_at` xử lý dòng đời, hoặc dùng chính trường `created_at` nghiệp vụ của bản ghi. Thêm một cột `_created_at` kỹ thuật nữa là thừa.

---

## 3. Các field CÓ THỂ BỎ HOÀN TOÀN (Tính toán In-Memory)

* **`_hash`**:
* *Mục đích cũ:* Thường dùng để so sánh xem bản ghi ở Shadow có khác bản ghi ở Master không trước khi thực hiện ghi (Tránh ghi đè trùng dữ liệu).
* *Tại sao bỏ được:* Bây giờ em đã có **`_version`** hoặc **`_source_ts`** rồi. Chỉ cần check `if event.version > master.version` thì mới thực hiện `UPSERT`. Việc tính toán và so sánh Hash tốn thêm CPU của Worker và tốn dung lượng lưu trữ của DB một cách vô ích.



---

## 📐 THIẾT KẾ RÚT GỌN (Tối ưu & Gọn đẹp)

Sau khi dọn dẹp, cấu trúc hệ thống của em sẽ cực kỳ sạch sẽ:

### Tại Shadow DB (Giữ nguyên để phục vụ CDC)

Giữ đầy đủ các trường kỹ thuật để Debezium làm việc và để em đo đạc thông số hệ thống (Lag, Throughput).

### Tại Master DB (Chỉ giữ lại 4 trường vận hành cốt lõi)

Bảng Master của em sẽ chỉ bao gồm các cột nghiệp vụ sạch (đã qua masking) và bọc đuôi bằng đúng 4 cột hệ thống sau:



### Worker (WK-E1 & WK-E2) sẽ tính toán thế nào để đảm bảo "Đầy đủ"?

* **WK-E2 (Trigger)** khi quét Shadow để sync qua Master: Nó chỉ cần filter `WHERE shadow._source_ts > (Chọn mốc checkpoint lưu trong Registry)` hoặc quét theo cơ chế `PENDING/DONE`.
* **WK-E1 (Transmute Engine)** khi ghi vào Master: Chạy câu lệnh OCC Upsert tinh gọn:



Chốt lại: **Bỏ bớt đi Trường ạ!** Giữ 4 trường hệ thống ở Master như anh gợi ý là vừa đủ để đảm bảo data toàn vẹn, vừa giúp các đội khác khi nhìn vào Master DB không bị ngợp bởi đống cột "rác" công nghệ.



----------------
----------------
----------------

update thêm ở http://localhost:5173/shadow khi click snapshot thì mới kick hoạt để úptream nhận log. cần 1 cái status trên page này để biết shadow đang hứng chưa. 
