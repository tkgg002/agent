# Kế hoạch Refactor chi tiết: Luồng Đặt vé xe (Booking Ticket Flow - Futa)

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo không mất đơn đặt vé khi `futa-connector-service` hoặc `booking-ticket-service` restart.
- **Xử lý Timeout:** Futa API có thể chậm, cần Graceful Shutdown phù hợp.
- **Saga Pattern:** Xử lý hoàn tiền/hủy vé nếu một bước trong quy trình thất bại.

---

## 2. Các Service liên quan

- `booking-ticket-service` (Orchestrator)
- `futa-connector-service` (Gọi API Futa)
- `payment-service` (Thanh toán)
- `insurance-service` (Mua bảo hiểm đi kèm - Optional)

---

## 3. Phân tích luồng hiện tại

```
User -> booking-ticket-service: Tìm chuyến xe
         |
         v
futa-connector-service: Gọi API Futa tìm kiếm
         |
         v
User chọn chuyến -> booking-ticket-service: Hold vé
         |
         v
futa-connector-service: Gọi API Hold vé
         |
         v
User thanh toán -> payment-service: Trừ tiền
         |
         v
futa-connector-service: Confirm vé
         |
         v
(Optional) insurance-service: Mua bảo hiểm
```

**Điểm yếu:**
- Futa API response chậm (5-10s) -> Cần timeout cao.
- Nếu thanh toán thành công nhưng Confirm vé thất bại -> Cần hoàn tiền.
- Service restart giữa các bước -> Đơn hàng bị treo.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng**

#### 1.1. Graceful Shutdown cho Futa Connector

**File:** `moleculer.config.ts`

```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 45000, // Futa API chậm
},
```

**K8s:**

```yaml
terminationGracePeriodSeconds: 60
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 10"]
```

### **Giai đoạn 2: Saga Pattern cho Booking**

#### 2.1. Schema `BookingSaga` (MongoDB)

```typescript
{
    sagaId: string;
    requestId: string;          // Idempotency Key
    userId: string;
    tripInfo: {
        departureDate: Date;
        from: string;
        to: string;
        seatNumbers: string[];
    };
    currentStep: 'INIT' | 'HELD' | 'PAID' | 'CONFIRMED' | 'INSURED' | 'COMPLETED' | 'ROLLBACK_PENDING' | 'CANCELLED';
    steps: [
        { name: 'HOLD', status: string, futaRef?: string },
        { name: 'PAYMENT', status: string, paymentId?: string },
        { name: 'CONFIRM', status: string, ticketCode?: string },
        { name: 'INSURANCE', status: string, insuranceId?: string },
    ];
    createdAt: Date;
    updatedAt: Date;
    expiresAt: Date;  // Hold vé có thời hạn
}
```

#### 2.2. Saga Orchestrator Logic

**File:** `booking-ticket-service/logics/booking.saga.ts`

```typescript
async executeBooking(payload: BookingPayload) {
    // 1. Check Idempotency
    let saga = await BookingSagaModel.findOne({ requestId: payload.requestId });
    if (saga?.currentStep === 'COMPLETED') {
        return saga;
    }

    // 2. Tạo mới nếu chưa có
    if (!saga) {
        saga = await BookingSagaModel.create({
            sagaId: generateId(),
            requestId: payload.requestId,
            userId: payload.userId,
            tripInfo: payload.tripInfo,
            currentStep: 'INIT',
            expiresAt: new Date(Date.now() + 15 * 60 * 1000), // 15 phút
        });
    }

    try {
        // 3. Resume từ step cuối cùng
        if (saga.currentStep === 'INIT') {
            await this.executeHold(saga);
        }
        if (saga.currentStep === 'HELD') {
            await this.executePayment(saga);
        }
        if (saga.currentStep === 'PAID') {
            await this.executeConfirm(saga);
        }
        if (saga.currentStep === 'CONFIRMED' && payload.withInsurance) {
            await this.executeInsurance(saga);
        }

        saga.currentStep = 'COMPLETED';
        await saga.save();
        return saga;

    } catch (error) {
        await this.executeRollback(saga, error);
        throw error;
    }
}
```

#### 2.3. Rollback Logic

```typescript
async executeRollback(saga: BookingSagaDocument, error: Error) {
    saga.currentStep = 'ROLLBACK_PENDING';
    await saga.save();

    // Rollback theo thứ tự ngược
    const paymentStep = saga.steps.find(s => s.name === 'PAYMENT');
    if (paymentStep?.status === 'SUCCESS') {
        // Hoàn tiền
        await this.broker.call('payment-service.refund', {
            paymentId: paymentStep.paymentId,
            reason: 'Booking failed: ' + error.message,
        });
    }

    const holdStep = saga.steps.find(s => s.name === 'HOLD');
    if (holdStep?.status === 'SUCCESS') {
        // Hủy hold vé
        await this.broker.call('futa-connector-service.cancelHold', {
            futaRef: holdStep.futaRef,
        });
    }

    saga.currentStep = 'CANCELLED';
    saga.error = error.message;
    await saga.save();
}
```

### **Giai đoạn 3: Job Sweeper (Xử lý vé hết hạn Hold)**

```typescript
async sweepExpiredBookings() {
    const expired = await BookingSagaModel.find({
        currentStep: 'HELD',
        expiresAt: { $lt: new Date() },
    });

    for (const saga of expired) {
        // Hủy hold và đánh dấu expired
        await this.broker.call('futa-connector-service.cancelHold', {
            futaRef: saga.steps.find(s => s.name === 'HOLD')?.futaRef,
        });
        saga.currentStep = 'EXPIRED';
        await saga.save();
    }
}
```

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `booking_sagas` | `requestId` | Tạo UNIQUE INDEX |
| `payments` | `booking_saga_id` | Tạo INDEX |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Futa API timeout | Tăng shutdownTimeout lên 45s |
| Thanh toán thành công, confirm thất bại | Saga rollback tự động hoàn tiền |
| Hold vé hết hạn | Sweeper quét và hủy |
| Service restart giữa chừng | Saga state persistent, resume khi startup |

---

## 7. Thứ tự triển khai

1. Tạo Schema `BookingSaga` và Index
2. Cấu hình Graceful Shutdown cho futa-connector (45s)
3. Implement Saga Orchestrator
4. Implement Rollback Logic
5. Triển khai Sweeper cho expired bookings
6. Test toàn diện
