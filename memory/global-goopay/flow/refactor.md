Đây là **LỘ TRÌNH THỰC THI CHI TIẾT (MASTER IMPLEMENTATION PLAN)** được tổng hợp từ toàn bộ các vấn đề đã thảo luận (Graceful Shutdown, Retry, Idempotency, Queue, Saga, UX).

Lộ trình này được thiết kế để áp dụng cho hệ thống ~60 services (Hybrid Node.js/Go) của bạn, đi từ sửa lỗi hạ tầng (Infrastructure) đến tái cấu trúc lõi (Core Re-architecture).

---

## 📅 GIAI ĐOẠN 0: RÀ SOÁT & CHUẨN BỊ (PREPARATION)

*Mục tiêu: Đảm bảo nền móng vững chắc trước khi thay đổi config.*

### 0.1. Kiểm tra Database Schema (Idempotency Audit)

* [ ] **Rà soát bảng Transaction/WalletLog:** Kiểm tra xem các bảng này đã có **UNIQUE INDEX** cho cột `request_id` (hoặc `reference_code`) chưa.
  * *Yêu cầu:* Nếu chưa có -> Tạo migration script đánh index ngay. Đây là "khiên chắn" bắt buộc để chống trừ tiền kép (Double Charging).
* [ ] **Chuẩn hóa Redis Key:** Review cách đặt key lock trong Redis (nếu có dùng distributed lock). Đảm bảo key có TTL (Time-To-Live) để tránh deadlock khi service crash.

### 0.2. Nâng cấp Shared Library (`package` repo)

* [ ] **Thêm Helper Graceful Shutdown:** Viết sẵn các hàm helper trong thư viện chung để các service con chỉ việc import và dùng, tránh sửa code lặp lại 50 lần.
* [ ] **Versioning:** Đóng băng version hiện tại, tạo version mới cho đợt nâng cấp này.

---

## 📅 GIAI ĐOẠN 1: ỔN ĐỊNH HẠ TẦNG (INFRASTRUCTURE STABILIZATION)

*Mục tiêu: Service tắt/bật mượt mà, không cắt kết nối "ngang xương". Xử lý lỗi 502/Connection Refused.*

### 1.1. Cấu hình Graceful Shutdown (Node.js/Moleculer)

*Áp dụng: 55 services Node.js*

* [ ] **Cập nhật `moleculer.config.ts`:**
  **TypeScript**

  ```
  tracking: {
      enabled: process.env.MOLECULER_TRACKING_ENABLED === 'true',
      shutdownTimeout: parseInt(process.env.MOLECULER_SHUTDOWN_TIMEOUT || '30000') // 30s mặc định
  }
  ```
* [ ] **Phân loại Timeout:**

  * Nhóm thường (User, Auth...): Timeout  **30s** .
  * Nhóm Connector (BIDV, Napas...): Timeout **60s** (Do bank phản hồi chậm).

### 1.2. Cấu hình Graceful Shutdown (Go Services)

*Áp dụng: 4 services Go (`disbursement`, `reconcile`...)*

* [ ] **Refactor `main.go`:**
  * Sử dụng `signal.Notify` lắng nghe `SIGTERM`.
  * Dùng `server.Shutdown(ctx)` thay vì để chương trình tự exit.
  * **Quan trọng:** Tách `Context` của DB Query ra khỏi `Context` của HTTP Request (Background Context) để query không bị kill khi HTTP connection đóng.

### 1.3. Cấu hình Kubernetes (Deployment YAML)

*Áp dụng: Toàn bộ 60 services*

* [ ] **Thêm `preStop` Hook:**
  **YAML**

  ```
  lifecycle:
    preStop:
      exec:
        command: ["/bin/sh", "-c", "sleep 10"] # "Câu giờ" cho Load Balancer cập nhật IP
  ```
* [ ] **Tăng `terminationGracePeriodSeconds`:**

  * Nhóm thường: **45s** ( > 30s timeout + 10s sleep).
  * Nhóm Connector:  **75s** .
* [ ] **Cấu hình `readinessProbe`:**

  * Trỏ vào API `/health/ready` (API này phải check connect DB/Redis/NATS).
  * Chỉ cho traffic vào khi Service thực sự sẵn sàng.

---

## 📅 GIAI ĐOẠN 2: LƯỚI AN TOÀN DỮ LIỆU (RESILIENCE & SAFETY NET)

*Mục tiêu: Nếu mạng đứt do restart, hệ thống tự thử lại. Nếu thử lại thất bại, có tool quét dọn.*

### 2.1. Cấu hình Retry Policy (Service-to-Service)

* [ ] **Moleculer Config:** Bật retry policy với exponential backoff.
  **JavaScript**

  ```
  retryPolicy: {
      enabled: true,
      retries: 3,
      delay: 500,
      maxDelay: 2000,
      factor: 2,
      check: (err) => err && err.retryable // Chỉ retry lỗi mạng/503
  }
  ```
* [ ] **Go Client:** Nếu Go gọi ngược lại Node, cấu hình HTTP Client có Retry (dùng thư viện như `go-retryablehttp`).

### 2.2. Implement Job "Quét rác" (Transaction Sweeper)

* [ ] **Tạo Service mới hoặc Module:** `reconcile-service` (Go) hoặc `scheduler-service`.
* [ ] **Logic Cronjob (Chạy 5-10 phút/lần):**
  1. Scan DB tìm giao dịch `PENDING` > 15 phút.
  2. Gọi API tra soát sang Bank/Partner.
  3. Update trạng thái cuối cùng (SUCCESS/FAILED) vào DB.
  4. Alert vào Slack/Telegram nếu giao dịch vẫn treo không xác định.

### 2.3. Admin Tooling (Cứu hộ thủ công)

* [ ] **CMS Portal:** Thêm module quản lý giao dịch treo.
* [ ] **Chức năng:**
  * `Check Status`: Gọi tra soát real-time.
  * `Force Success/Fail`: Cập nhật trạng thái bằng tay (có Audit Log lý do).

---

## 📅 GIAI ĐOẠN 3: TÁI CẤU TRÚC ASYNC & SAGA (CORE RE-ARCHITECTURE)

*Mục tiêu: Decoupling hệ thống. Chuyển đổi các luồng quan trọng (Payment -> Disbursement) sang Bất đồng bộ.*

### 3.1. Chuyển đổi sang NATS JetStream (Queue)

*Thay thế RPC `ctx.call` trực tiếp giữa Node và Go.*

* [ ] **Thiết lập Stream:** Tạo Stream `EVENTS_PAYMENT` trên NATS JetStream (Lưu file, Replicas 3).
* [ ] **Producer (Node.js - Payment Service):**
  * Thay vì gọi disbursement.transfer, hãy Publish event:
    payment.deducted (Payload: { tx_id, amount, bank_info }).
* [ ] **Consumer (Go - Disbursement Service):**
  * Viết Worker Subscribe `payment.deducted`.
  * **Logic:** Nhận message -> Chuyển tiền Bank -> Update DB.
  * **Manual ACK:** Chỉ gửi ACK cho NATS khi DB đã update xong. Nếu crash, message tự động được gửi lại cho Pod khác.

### 3.2. Implement Saga Pattern (Orchestrator)

*Xử lý hoàn tiền tự động khi lỗi.*

* [ ] **Saga Coordinator:** (Nên đặt tại `payment-service` hoặc tách service riêng).
* [ ] **Định nghĩa State Machine:**
  * Step 1: `Wallet Deduct` (Node) -> Done.
  * Step 2: `Bank Transfer` (Go) -> Failed?
  * Step 3 (Compensate): `Wallet Refund` (Node) -> Triggered.
* [ ] **Xử lý Rollback:**
  * Nếu Consumer (Go) báo lỗi (hoặc hết retry queue) -> Publish event `disbursement.failed`.
  * Coordinator nhận event -> Gọi action hoàn tiền vào ví user.

---

## 📅 GIAI ĐOẠN 4: TỐI ƯU TRẢI NGHIỆM & GIÁM SÁT (UX & OPS)

*Mục tiêu: Khách hàng không ức chế khi chờ đợi, Dev dễ debug.*

### 4.1. Nâng cấp Frontend (React/Mobile)

* [ ] **Xử lý Timeout 504:**
  * Catch lỗi HTTP 504 từ Gateway.
  * Hiển thị: *"Hệ thống đang xử lý, vui lòng kiểm tra lịch sử"* (Thay vì báo đỏ "Thất bại").
* [ ] **Cơ chế Polling/Socket:**
  * Sau khi submit, App lắng nghe Socket event `TX_COMPLETED`.
  * Fallback: Gọi API `/status` mỗi 5s để cập nhật trạng thái nếu Socket mất.

### 4.2. API Gateway (Async Response)

* [ ] **Update Endpoint Nạp/Rút:**
  * Trả về HTTP 202 (Accepted) ngay khi đẩy job vào Queue thành công.
  * Body: `{ status: "PROCESSING", request_id: "..." }`.

### 4.3. Observability (Giám sát)

* [ ] **Distributed Tracing (Jaeger):**
  * Tích hợp Moleculer Tracing.
  * Đảm bảo `request_id` được truyền xuyên suốt: Gateway -> Node -> NATS -> Go -> DB.
* [ ] **Alerting:**
  * Cảnh báo khi Queue backlog tăng cao (Go service xử lý chậm).
  * Cảnh báo khi tỷ lệ rollback (Saga compensation) > 1%.

---

## 🚦 TIÊU CHÍ NGHIỆM THU (SUCCESS METRICS)

| **Giai đoạn** | **Tiêu chí hoàn thành (Definition of Done)**                                                                                                    |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **GĐ 1**       | Restart 1 service bất kỳ (kể cả Go) không gây lỗi 502 cho Gateway. Log không báo lỗi "Socket Hang up".                                          |
| **GĐ 2**       | Tắt Service Disbursement (Go) 1 phút rồi bật lại, các request từ Payment (Node) tự động retry thành công, không mất đơn.                  |
| **GĐ 3**       | Rút dây mạng Service Go khi đang xử lý, sau khi bật lại, giao dịch trong Queue tiếp tục chạy. Nếu lỗi Bank, tiền ví tự hoàn về (Saga). |
| **GĐ 4**       | User không bao giờ thấy thông báo "Lỗi không xác định" khi server restart. Admin có tool để xử lý các ca lỗi hiếm gặp.                 |

### ĐỀ XUẤT THỨ TỰ TRIỂN KHAI CHO 60 SERVICES

1. **Nhóm Pilot:** `notification-service`, `promotion-service` (Ít rủi ro).
2. **Nhóm Gateway:** `admin-portal-gateway`, `merchant-portal-gateway`.
3. **Nhóm Core:** `user`, `auth`, `profile`.
4. **Nhóm Finance (Critical):** `wallet`, `payment`, `disbursement` (Go).
5. **Nhóm Connector:** `bidv`, `napas`... (Cần test timeout kỹ nhất).


# **KHÔNG CÓ giải pháp kỹ thuật nào (ở mức code/config) để buộc tất cả các service trong một chuỗi phân tán (Distributed Chain) tự động "nín thở" chờ nhau hoàn thành rồi mới tắt.**

Lý do: Kubernetes và Microservices sinh ra để hoạt động  **độc lập** . Service A không thể (và không nên) điều khiển việc sống/chết của Service B và C.

Tuy nhiên, có một **Giải pháp Hạ tầng (Infrastructure Solution)** để đạt được mục đích bạn muốn:  **"Để cho toàn bộ giao dịch cũ chạy xong hết rồi mới tắt luồng cũ"** . Đó là chiến lược  **BLUE/GREEN DEPLOYMENT** .

Dưới đây là phân tích các giải pháp từ "Nhà giàu" đến "Nhà nghèo":

---

### 1. Giải pháp "Đại gia": Blue/Green Deployment (Khuyên dùng nếu dư tài nguyên)

Đây là cách duy nhất để **Graceful toàn bộ hệ thống** mà không cần sửa một dòng code nào.

Cơ chế:

Thay vì thay thế dần dần từng Pod (Rolling Update), bạn dựng hẳn một hệ thống mới (Green) song song với hệ thống cũ (Blue).

1. **Hiện tại (Blue):** Version 1 đang chạy. User -> Gateway -> Service A -> Service B -> Bank.
2. **Deploy (Green):** Dựng Version 2 lên. Lúc này chưa có traffic nào vào Green.
3. **Switch Traffic:** Điều hướng ở Router/Gateway. Chuyển 100% traffic mới sang  **Green** .
4. **Draining (Quan trọng nhất):**
   * Lúc này, **Blue** không nhận khách mới, nhưng các giao dịch cũ vẫn đang chạy trong đó.
   * Giữ **Blue** sống thêm 5-10 phút (thay vì 30s như Graceful Shutdown thường).
   * Trong 5-10 phút đó, mọi transaction phức tạp nhất (qua A, B, C, Go, Bank) đều sẽ hoàn thành trọn vẹn.
5. **Shutdown:** Sau khi chắc chắn Blue đã "sạch" traffic -> Tắt toàn bộ cụm Blue.

Ưu điểm: Zero Downtime tuyệt đối. An toàn nhất cho giao dịch nhiều bước.

Nhược điểm: Tốn gấp đôi tài nguyên Server (RAM/CPU) trong lúc deploy.

---

### 2. Giải pháp "Tắt đèn": Maintenance Mode (Chế độ bảo trì)

Nếu không đủ tiền làm Blue/Green, và giao dịch quá quan trọng, hãy dùng cách này.

**Cơ chế:**

1. **Bước 1:** Bật cờ "Maintenance" trên API Gateway.
   * Request mới -> Trả về lỗi đẹp: "Hệ thống đang nâng cấp, vui lòng quay lại sau 2 phút".
2. **Bước 2:** Chờ 2-3 phút.
   * Lúc này hệ thống "tĩnh" lại. Các giao dịch dở dang sẽ chạy nốt. Không có giao dịch mới chui vào.
3. **Bước 3:** Bắt đầu Restart/Deploy các service bên dưới (A, B, C, Go...).
4. **Bước 4:** Tắt cờ Maintenance.

Ưu điểm: Đảm bảo 100% nhất quán dữ liệu. Không bao giờ lo đứt gánh giữa đường.

Nhược điểm: Có Downtime (Khách không giao dịch được trong vài phút).

---

### 3. Giải pháp "Kiến trúc sư": Chuyển đổi Synchronous -> Asynchronous (Đã bàn ở trên)

Đây là cách giải quyết vấn đề từ gốc rễ:  **Đừng để transaction phụ thuộc vào việc Service phải "Sống"** .

**Ví dụ:**

* **Cách cũ (Đồng bộ):** A gọi B. B phải sống thì A mới xong. Nếu B restart -> A chết -> Transaction chết. -> **Cần Graceful cả dây.**
* **Cách mới (Queue):** A ném tin nhắn vào Hòm thư. A xong việc. B restart thoải mái. B dậy thì mở hòm thư ra làm tiếp. -> **Không cần Graceful cả dây.**

=> Khi dùng Queue (Giai đoạn 3), khái niệm "Graceful toàn bộ Transaction" trở nên vô nghĩa, vì Transaction đã được cắt nhỏ ra và bền vững với việc restart.

---

### 4. Giải pháp "hack" (Dùng Distributed Lock - Không khuyến khích)

Một số người cố gắng làm điều này:

1. Khi Service A bắt đầu transaction, nó tạo 1 Key Redis: `lock:tx_123`.
2. Tất cả Service (A, B, C) trước khi shutdown đều check Redis: "Còn key lock nào không?".
3. Nếu còn -> Từ chối shutdown.

**Tại sao KHÔNG NÊN làm:**

* **Kubernetes không quan tâm:** K8s gửi lệnh tắt (`SIGTERM`). Nó chờ hết giờ (`terminationGracePeriodSeconds`) là nó chém (`SIGKILL`). Service của bạn có cố tình không tắt thì cũng bị K8s giết.
* **Deadlock:** Nếu Service A tạo lock xong bị Crash (OOM), cái lock đó treo vĩnh viễn -> Cả hệ thống không ai dám tắt -> Treo deployment.

### TỔNG KẾT & LỜI KHUYÊN

Bạn đang tìm kiếm sự hoàn hảo trong một hệ thống phân tán, điều đó rất rủi ro.

1. **Nếu bạn giàu tài nguyên:** Hãy dùng **Blue/Green Deployment** (hoặc Canary Deployment). Đây là cách các Big Tech làm để đảm bảo transaction không đứt.
2. **Nếu bạn tiết kiệm:** Hãy chấp nhận việc Transaction sẽ bị lỗi một tỷ lệ nhỏ khi deploy, và dùng **Giai đoạn 2 (Retry + Sweeper)** để dọn dẹp nó.
3. **Đích đến cuối cùng:** Hãy chuyển sang  **Queue/Saga (Giai đoạn 3)** . Lúc đó bạn sẽ ngủ ngon mà không cần quan tâm service nào đang restart.

**Lời khuyên cho bạn:** Đừng cố "Graceful toàn bộ". Hãy làm **Giai đoạn 1** (Graceful từng node) thật tốt + **Giai đoạn 2** (Safety Net) thật kỹ. Đó là điểm cân bằng tốt nhất giữa Chi phí và Hiệu quả.
