# Kế hoạch Refactor chi tiết: Luồng Thanh toán Hóa đơn (Luồng thu hộ - Payment Bill Flow)

**Tệp:** `refactor-plan-payment-bill-flow.md`

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo các service trong luồng (`payment-gateway`, `payment-bill-service`, `payment-service`, `core-trans-proxy-service`) không bị mất giao dịch khi restart hoặc deploy (Graceful Shutdown).
- **Tăng độ tin cậy:** Tự động thử lại (retry) các lệnh gọi service-to-service nếu có lỗi mạng tạm thời.
- **Chống trùng lặp:** Đảm bảo cơ chế Idempotency để việc retry không dẫn đến xử lý một yêu cầu nhiều lần.
- **Cải thiện trải nghiệm người dùng (UX):** Chuyển từ mô hình đồng bộ (chờ đợi lâu) sang bất đồng bộ (phản hồi ngay lập tức).

---

## 2. Các Service liên quan

- `payment-gateway`
- `payment-bill-service`
- `payment-service`
- `core-trans-proxy-service`
- `bank-transfer-service`
- `wallet-service`

---

## 3. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng (Không thay đổi logic)**

#### **Đối với tất cả các service Node.js trong luồng:**

(`payment-gateway`, `payment-bill-service`, `payment-service`, `core-trans-proxy-service`, `bank-transfer-service`, `wallet-service`)

**1. Service: Bất kỳ service nào trong danh sách trên**

- **File:** `moleculer.config.ts`
- **Mục đích:** Bật cơ chế tracking request và graceful shutdown.
- **Thay đổi:**
  ```typescript
  // old_string
  const brokerConfig: BrokerOptions = {
      // ...
      logLevel: "info",
      // có thể có hoặc không có tracking
  };

  // new_string
  const brokerConfig: BrokerOptions = {
      // ...
      logLevel: "info",
      tracking: {
          enabled: true,
          shutdownTimeout: 30000, // 30 giây
      },
  };
  ```

**2. Service: Bất kỳ service nào trong danh sách trên**

- **File:** `moleculer.config.ts`
- **Mục đích:** Cấu hình retry policy toàn cục cho các lệnh gọi service.
- **Thay đổi:**
  ```typescript
  // old_string
  const brokerConfig: BrokerOptions = {
      // ...
      // có thể có hoặc không có requestTimeout
      requestTimeout: 60 * 1000,
  };

  // new_string
  const brokerConfig: BrokerOptions = {
      // ...
      requestTimeout: 60 * 1000,
      retryPolicy: {
          enabled: true,
          retries: 3,
          delay: 200,
          maxDelay: 2000,
          factor: 2,
          check: (err: Errors.MoleculerError) => err && !!err.retryable,
      },
  };
  ```

**3. Service: Bất kỳ service nào trong danh sách trên**

- **File:** `*.service.ts` (file định nghĩa service chính)
- **Mục đích:** Đảm bảo đóng kết nối DB khi service tắt.
- **Thay đổi (ví dụ cho service dùng Mongoose):**
  ```typescript
  // old_string
  // có thể không có hook stopped()

  // new_string
  // Thêm vào trong ServiceSchema
  stopped: async () => {
      try {
          await mongoose.disconnect();
          this.logger.info("MongoDB disconnected successfully.");
      } catch (error) {
          this.logger.error("MongoDB disconnection error.", error);
      }
  }
  ```

### **Giai đoạn 2: Tái cấu trúc Bất đồng bộ (Async Refactoring)**

**Mục tiêu:** Phá vỡ chuỗi gọi đồng bộ từ `payment-service` -> `core-trans-proxy-service`.

**1. Service: `payment-service` (Producer)**

- **File:** `logics/payment.logic.ts` (giả định) - Nơi xử lý logic thanh toán chính.
- **Mục đích:** Thay vì gọi trực tiếp `core-trans-proxy-service`, sẽ publish một event ra NATS.
- **Thay đổi:**
  ```typescript
  // old_string
  // Giả sử có đoạn code gọi trực tiếp
  const result = await ctx.call('core-trans-proxy.processPayment', {
      amount: paymentData.amount,
      orderId: paymentData.orderId,
      // ...
  });
  // Xử lý result...

  // new_string
  // Thay thế đoạn gọi trực tiếp bằng publish event
  await ctx.emit('payment.initiated', {
      amount: paymentData.amount,
      orderId: paymentData.orderId,
      paymentId: newPayment._id, // Cần có ID để theo dõi
      timestamp: new Date(),
      // ...
  });
  // Cập nhật trạng thái thanh toán trong DB thành 'PROCESSING'
  await this.paymentModel.updateOne({ _id: newPayment._id }, { $set: { status: 'PROCESSING' } });
  // Không cần chờ kết quả cuối cùng
  ```

**2. Service: `core-trans-proxy-service` (Consumer)**

- **File:** `services/core-trans-proxy.service.ts`
- **Mục đích:** Lắng nghe event `payment.initiated` và xử lý.
- **Thay đổi:**
  ```typescript
  // Thêm một event listener vào trong ServiceSchema
  events: {
      "payment.initiated": {
          group: "core-trans-proxy", // Để đảm bảo chỉ 1 instance nhận message
          async handler(ctx: Context<PaymentInitiatedPayload>) {
              this.logger.info("Received payment.initiated event", ctx.params);
              try {
                  // Lấy lại logic từ action 'processPayment' cũ
                  const paymentData = ctx.params;

                  // Gọi các service con như bank-transfer, wallet-service...
                  const finalResult = await this.processPaymentLogic(paymentData);

                  // Sau khi xử lý xong, publish event kết quả
                  await ctx.emit('payment.completed', { 
                      paymentId: paymentData.paymentId,
                      status: 'SUCCESS',
                      result: finalResult 
                  });

              } catch (error) {
                  this.logger.error("Error processing payment.initiated event", error);
                  // Publish event lỗi để service khác (ví dụ: payment-service) có thể rollback
                  await ctx.emit('payment.failed', {
                      paymentId: ctx.params.paymentId,
                      error: error.message,
                  });
              }
          }
      }
  }
  ```

**3. Service: `payment-service` (Consumer - để cập nhật trạng thái cuối)**

- **File:** `services/payment.service.ts`
- **Mục đích:** Lắng nghe các event kết quả (`payment.completed`, `payment.failed`) để cập nhật trạng thái cuối cùng của thanh toán.
- **Thay đổi:**
  ```typescript
  // Thêm event listeners vào ServiceSchema
  events: {
      "payment.completed": {
          group: "payment-service",
          async handler(ctx: Context<PaymentCompletedPayload>) {
              await this.paymentModel.updateOne(
                  { _id: ctx.params.paymentId },
                  { $set: { status: 'SUCCESS', transactionResult: ctx.params.result } }
              );
          }
      },
      "payment.failed": {
          group: "payment-service",
          async handler(ctx: Context<PaymentFailedPayload>) {
              await this.paymentModel.updateOne(
                  { _id: ctx.params.paymentId },
                  { $set: { status: 'FAILED', errorMessage: ctx.params.error } }
              );
              // (Tùy chọn) Kích hoạt logic hoàn tiền nếu cần
          }
      }
  }
  ```

### **Giai đoạn 3: Cải thiện Trải nghiệm Người dùng (UX)**

**1. Service: `payment-gateway`**

- **File:** (File xử lý route tạo thanh toán, ví dụ: `integrate-services/v1/payment/create.ts`)
- **Mục đích:** Trả về phản hồi ngay cho người dùng thay vì chờ xử lý xong.
- **Thay đổi:**
  ```typescript
  // old_string
  // Giả sử gateway đang chờ kết quả cuối cùng
  const result = await ctx.call('payment-service.createAndProcess', params);
  return ResponseHelper.resOK(result);

  // new_string
  // Gọi action mới của payment-service (chỉ đẩy vào queue)
  const result = await ctx.call('payment-service.initiatePayment', params);

  // Trả về HTTP 202 Accepted
  ctx.meta.$statusCode = 202; 
  return ResponseHelper.resOK({
      status: "PROCESSING",
      message: "Your payment is being processed.",
      paymentId: result.paymentId, // Trả về ID để client có thể theo dõi
  });
  ```

---

## 4. Rủi ro và Giải pháp

- **Mất message NATS:** Cần cấu hình NATS JetStream để đảm bảo message được lưu trữ bền vững (durable consumer).
- **Xử lý trùng lặp:** Các consumer phải có logic Idempotency, kiểm tra `paymentId` đã được xử lý chưa trước khi thực hiện nghiệp vụ.
- **Giao dịch treo:** Cần có một job định kỳ (trong `scheduler-service`) để quét các thanh toán có trạng thái `PROCESSING` quá lâu và thực hiện tra soát.
