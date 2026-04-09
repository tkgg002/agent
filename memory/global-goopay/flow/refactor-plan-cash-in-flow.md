# Kế hoạch Refactor chi tiết: Luồng Nạp tiền vào ví (Cash-in Flow)

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo không mất giao dịch nạp tiền khi Bank Connector hoặc service restart.
- **Xử lý Callback:** Đảm bảo nhận và xử lý đúng callback từ ngân hàng (BIDV, Napas...).
- **Idempotency:** Chống xử lý callback 2 lần (Bank có thể gửi lại callback).

---

## 2. Các Service liên quan

- `bank-transfer-service` (Tạo tài khoản ảo/QR)
- `bidv-connector-service` / `napas-connector-service` / `banvietbank-connector-service`
- `core-trans-proxy-service`
- `wallet-service`

---

## 3. Phân tích luồng hiện tại

```
User -> App yêu cầu nạp tiền
         |
         v
bank-transfer-service: Tạo Virtual Account / QR Code
         |
         v
User chuyển tiền từ Bank App
         |
         v
Bank gửi Callback -> [bank]-connector-service
         |
         v
core-trans-proxy-service: Xử lý giao dịch
         |
         v
wallet-service: Cộng tiền vào ví
```

**Điểm yếu:**
- Bank Connector restart khi đang nhận callback -> Mất callback -> User đã chuyển tiền nhưng ví không cộng.
- Bank gửi callback 2 lần (retry) -> Cộng tiền 2 lần nếu không có Idempotency.
- Connector timeout 30s nhưng Bank phản hồi 35s -> Graceful Shutdown cắt giữa chừng.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng**

#### 1.1. Cấu hình Graceful Shutdown cho Bank Connectors

**Service:** `bidv-connector-service`, `napas-connector-service`, `banvietbank-connector-service`
**File:** `moleculer.config.ts`

```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 60000, // 60s - Bank response chậm
},
```

**K8s Deployment:**

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 15"]
terminationGracePeriodSeconds: 80  # > 60s + 15s
```

#### 1.2. Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 5
```

### **Giai đoạn 2: Idempotency cho Callback**

#### 2.1. Schema `BankCallback` (MongoDB)

```typescript
{
    callbackId: string;      // ID từ Bank (unique)
    bankCode: string;        // BIDV, NAPAS...
    transactionRef: string;  // Mã giao dịch của Bank
    amount: number;
    status: 'RECEIVED' | 'PROCESSING' | 'SUCCESS' | 'FAILED';
    rawPayload: object;      // Lưu nguyên callback để debug
    processedAt: Date;
    createdAt: Date;
}
```

**Index:**
- `UNIQUE INDEX` trên `(bankCode, transactionRef)` hoặc `callbackId`

#### 2.2. Logic xử lý Callback

**File:** `[bank]-connector-service/handlers/callback.handler.ts`

```typescript
async handleCallback(payload: BankCallbackPayload) {
    const callbackKey = `${payload.bankCode}_${payload.transactionRef}`;

    // 1. Check duplicate
    const existing = await BankCallbackModel.findOne({ callbackId: callbackKey });
    if (existing) {
        if (existing.status === 'SUCCESS') {
            this.logger.info(`Callback already processed: ${callbackKey}`);
            return { success: true, duplicate: true };
        }
        // Nếu đang PROCESSING hoặc FAILED, có thể retry
    }

    // 2. Lưu callback vào DB
    const callback = await BankCallbackModel.findOneAndUpdate(
        { callbackId: callbackKey },
        {
            $setOnInsert: { callbackId: callbackKey, createdAt: new Date() },
            $set: { 
                status: 'PROCESSING',
                rawPayload: payload,
                amount: payload.amount,
            },
        },
        { upsert: true, new: true }
    );

    // 3. Xử lý nghiệp vụ
    try {
        await this.broker.call('core-trans-proxy-service.processCashIn', {
            transactionRef: payload.transactionRef,
            amount: payload.amount,
            bankCode: payload.bankCode,
        });

        callback.status = 'SUCCESS';
        callback.processedAt = new Date();
        await callback.save();

        return { success: true };

    } catch (error) {
        callback.status = 'FAILED';
        callback.error = error.message;
        await callback.save();
        throw error;
    }
}
```

### **Giai đoạn 3: Async Processing với Queue**

**Mục đích:** Tách việc nhận callback (nhanh) và xử lý callback (có thể chậm).

#### 3.1. Producer (Connector)

```typescript
async handleCallback(payload: BankCallbackPayload) {
    // Lưu callback ngay lập tức
    await BankCallbackModel.create({
        callbackId: `${payload.bankCode}_${payload.transactionRef}`,
        status: 'RECEIVED',
        rawPayload: payload,
    });

    // Đẩy vào Queue để xử lý async
    await this.broker.emit('bank.callback.received', payload);

    // Trả về OK cho Bank ngay lập tức
    return { success: true };
}
```

#### 3.2. Consumer (core-trans-proxy-service)

```typescript
events: {
    "bank.callback.received": {
        group: "core-trans-proxy",
        async handler(ctx: Context<BankCallbackPayload>) {
            const payload = ctx.params;
            // Xử lý nghiệp vụ (cộng tiền, update trạng thái...)
            await this.processCashIn(payload);
        }
    }
}
```

### **Giai đoạn 4: Job Sweeper cho Callback**

**Service:** `scheduler-service`

```typescript
async sweepPendingCallbacks() {
    const pending = await BankCallbackModel.find({
        status: { $in: ['RECEIVED', 'PROCESSING'] },
        createdAt: { $lt: new Date(Date.now() - 10 * 60 * 1000) }, // > 10 phút
    });

    for (const cb of pending) {
        this.logger.warn(`Retrying stuck callback: ${cb.callbackId}`);
        await this.broker.emit('bank.callback.received', cb.rawPayload);
    }
}
```

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `bank_callbacks` | `callbackId` | Tạo UNIQUE INDEX |
| `wallet_logs` | `reference_code` | Đảm bảo có UNIQUE INDEX |
| `transactions` | `bank_transaction_ref` | Tạo UNIQUE INDEX |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Connector restart khi nhận callback | Lưu callback vào DB ngay lập tức, xử lý async |
| Bank gửi callback 2 lần | Idempotency Key (transactionRef) |
| Callback bị treo | Job Sweeper quét định kỳ |
| Bank timeout > graceful timeout | Tăng shutdownTimeout cho Connector lên 60s |

---

## 7. Thứ tự triển khai

1. Tạo Schema `BankCallback` và Index
2. Cấu hình Graceful Shutdown cho Bank Connectors (60s)
3. Refactor callback handler với Idempotency
4. Triển khai Async Queue cho callback processing
5. Triển khai Job Sweeper
6. Test với các kịch bản: Callback bình thường, Callback duplicate, Connector restart
