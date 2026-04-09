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
```typescript
tracking: {
    enabled: process.env.MOLECULER_TRACKING_ENABLED === 'true',
    shutdownTimeout: parseInt(process.env.MOLECULER_SHUTDOWN_TIMEOUT || '30000') // 30s mặc định
}

```


* [ ] **Phân loại Timeout:**
* Nhóm thường (User, Auth...): Timeout **30s**.
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
```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"] # "Câu giờ" cho Load Balancer cập nhật IP

```


* [ ] **Tăng `terminationGracePeriodSeconds`:**
* Nhóm thường: **45s** ( > 30s timeout + 10s sleep).
* Nhóm Connector: **75s**.


* [ ] **Cấu hình `readinessProbe`:**
* Trỏ vào API `/health/ready` (API này phải check connect DB/Redis/NATS).
* Chỉ cho traffic vào khi Service thực sự sẵn sàng.



---

## 📅 GIAI ĐOẠN 2: LƯỚI AN TOÀN DỮ LIỆU (RESILIENCE & SAFETY NET)

*Mục tiêu: Nếu mạng đứt do restart, hệ thống tự thử lại. Nếu thử lại thất bại, có tool quét dọn.*

### 2.1. Cấu hình Retry Policy (Service-to-Service)

* [ ] **Moleculer Config:** Bật retry policy với exponential backoff.
```javascript
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
* Thay vì gọi `disbursement.transfer`, hãy Publish event:
`payment.deducted` (Payload: `{ tx_id, amount, bank_info }`).


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

| Giai đoạn | Tiêu chí hoàn thành (Definition of Done) |
| --- | --- |
| **GĐ 1** | Restart 1 service bất kỳ (kể cả Go) không gây lỗi 502 cho Gateway. Log không báo lỗi "Socket Hang up". |
| **GĐ 2** | Tắt Service Disbursement (Go) 1 phút rồi bật lại, các request từ Payment (Node) tự động retry thành công, không mất đơn. |
| **GĐ 3** | Rút dây mạng Service Go khi đang xử lý, sau khi bật lại, giao dịch trong Queue tiếp tục chạy. Nếu lỗi Bank, tiền ví tự hoàn về (Saga). |
| **GĐ 4** | User không bao giờ thấy thông báo "Lỗi không xác định" khi server restart. Admin có tool để xử lý các ca lỗi hiếm gặp. |

### ĐỀ XUẤT THỨ TỰ TRIỂN KHAI CHO 60 SERVICES

1. **Nhóm Pilot:** `notification-service`, `promotion-service` (Ít rủi ro).
2. **Nhóm Gateway:** `admin-portal-gateway`, `merchant-portal-gateway`.
3. **Nhóm Core:** `user`, `auth`, `profile`.
4. **Nhóm Finance (Critical):** `wallet`, `payment`, `disbursement` (Go).
5. **Nhóm Connector:** `bidv`, `napas`... (Cần test timeout kỹ nhất).