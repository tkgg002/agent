# Kế hoạch Refactor chi tiết: Luồng Chuyển tiền trong ví (Wallet to Wallet Transfer)

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo `wallet-trans-service` (Saga) không bị mất giao dịch khi restart.
- **Bảo vệ Saga:** Đảm bảo các bước Saga (Trừ ví nguồn -> Cộng ví đích) được hoàn thành hoặc rollback đúng cách.
- **Chống trùng lặp:** Đảm bảo Idempotency để retry không gây chuyển tiền 2 lần.

---

## 2. Các Service liên quan

- `wallet-trans-service` (Saga Orchestrator)
- `wallet-service` (Executor)
- `notification-service`

---

## 3. Phân tích luồng hiện tại

```
User Request -> wallet-trans-service (Saga)
                     |
                     v
            Step 1: wallet-service.deduct(source_wallet)
                     |
                     v
            Step 2: wallet-service.credit(dest_wallet)
                     |
                     v
            Step 3: notification-service.send()
```

**Điểm yếu:**
- Nếu `wallet-trans-service` restart giữa Step 1 và Step 2: Tiền đã trừ từ ví nguồn nhưng chưa cộng vào ví đích -> **Mất tiền**.
- Nếu retry mà không có Idempotency: Trừ tiền 2 lần.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng**

#### 1.1. Cấu hình Graceful Shutdown

**Service:** `wallet-trans-service`, `wallet-service`
**File:** `moleculer.config.ts`

```typescript
tracking: {
    enabled: true,
    shutdownTimeout: 45000, // Saga cần thời gian dài hơn để hoàn tất
},
```

#### 1.2. Cấu hình Retry Policy

**File:** `moleculer.config.ts`

```typescript
retryPolicy: {
    enabled: true,
    retries: 3,
    delay: 500,
    maxDelay: 3000,
    factor: 2,
    check: (err) => err && err.retryable,
},
```

### **Giai đoạn 2: Bảo vệ Saga (Quan trọng nhất)**

#### 2.1. Lưu trữ Saga State

**Mục đích:** Đảm bảo có thể resume Saga sau khi service restart.

**Schema `SagaTransaction` (MongoDB):**

```typescript
{
    sagaId: string;           // Unique ID cho mỗi Saga
    requestId: string;        // Idempotency Key
    sourceWallet: string;
    destWallet: string;
    amount: number;
    currentStep: 'INIT' | 'DEDUCTED' | 'CREDITED' | 'NOTIFIED' | 'COMPLETED' | 'ROLLBACK_PENDING' | 'ROLLED_BACK';
    steps: [
        { name: 'DEDUCT', status: 'PENDING' | 'SUCCESS' | 'FAILED', result?: any },
        { name: 'CREDIT', status: 'PENDING' | 'SUCCESS' | 'FAILED', result?: any },
        { name: 'NOTIFY', status: 'PENDING' | 'SUCCESS' | 'FAILED', result?: any },
    ];
    createdAt: Date;
    updatedAt: Date;
}
```

**Index:**
- `UNIQUE INDEX` trên `requestId` (Idempotency)
- `INDEX` trên `currentStep` + `updatedAt` (để query các Saga bị treo)

#### 2.2. Logic Saga Orchestrator

**File:** `wallet-trans-service/logics/transfer.saga.ts`

```typescript
async executeTransfer(payload: TransferPayload) {
    const { requestId, sourceWallet, destWallet, amount } = payload;

    // 1. Check Idempotency
    let saga = await SagaModel.findOne({ requestId });
    if (saga && saga.currentStep === 'COMPLETED') {
        return saga; // Đã xử lý, trả kết quả cũ
    }

    // 2. Nếu chưa có, tạo mới
    if (!saga) {
        saga = await SagaModel.create({
            sagaId: generateId(),
            requestId,
            sourceWallet,
            destWallet,
            amount,
            currentStep: 'INIT',
            steps: [
                { name: 'DEDUCT', status: 'PENDING' },
                { name: 'CREDIT', status: 'PENDING' },
                { name: 'NOTIFY', status: 'PENDING' },
            ],
        });
    }

    // 3. Resume từ step cuối cùng
    try {
        if (saga.currentStep === 'INIT' || saga.currentStep === 'ROLLBACK_PENDING') {
            await this.executeDeduct(saga);
        }
        if (saga.currentStep === 'DEDUCTED') {
            await this.executeCredit(saga);
        }
        if (saga.currentStep === 'CREDITED') {
            await this.executeNotify(saga);
        }
        
        saga.currentStep = 'COMPLETED';
        await saga.save();
        return saga;

    } catch (error) {
        // 4. Rollback nếu lỗi
        await this.executeRollback(saga, error);
        throw error;
    }
}
```

#### 2.3. Logic Rollback (Compensating Transaction)

```typescript
async executeRollback(saga: SagaDocument, error: Error) {
    saga.currentStep = 'ROLLBACK_PENDING';
    await saga.save();

    // Rollback theo thứ tự ngược
    if (saga.steps.find(s => s.name === 'DEDUCT' && s.status === 'SUCCESS')) {
        // Hoàn tiền về ví nguồn
        await this.broker.call('wallet-service.credit', {
            walletId: saga.sourceWallet,
            amount: saga.amount,
            reference: `ROLLBACK_${saga.sagaId}`,
        });
    }

    saga.currentStep = 'ROLLED_BACK';
    saga.error = error.message;
    await saga.save();
}
```

### **Giai đoạn 3: Job Sweeper (Dọn dẹp Saga bị treo)**

**Service:** `scheduler-service`
**Cron:** Mỗi 5 phút

```typescript
async sweepStuckSagas() {
    const stuckSagas = await SagaModel.find({
        currentStep: { $nin: ['COMPLETED', 'ROLLED_BACK'] },
        updatedAt: { $lt: new Date(Date.now() - 15 * 60 * 1000) }, // > 15 phút
    });

    for (const saga of stuckSagas) {
        this.logger.warn(`Found stuck saga: ${saga.sagaId}`);
        
        // Thử resume
        try {
            await this.broker.call('wallet-trans-service.resumeSaga', { sagaId: saga.sagaId });
        } catch (error) {
            // Nếu vẫn lỗi, đánh dấu cần review thủ công
            saga.currentStep = 'NEED_MANUAL_REVIEW';
            await saga.save();
            // Alert to Slack/Telegram
        }
    }
}
```

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Đã có Index? | Action |
|---|---|---|---|
| `saga_transactions` | `requestId` | ❓ | Tạo UNIQUE INDEX |
| `wallet_logs` | `reference_code` | ❓ | Tạo UNIQUE INDEX |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Service restart giữa Saga | Lưu Saga State vào DB, resume khi startup |
| Retry gây duplicate | Idempotency Key (`requestId`) |
| Saga bị treo mãi | Job Sweeper quét định kỳ |
| Rollback thất bại | Đánh dấu `NEED_MANUAL_REVIEW`, alert Admin |

---

## 7. Thứ tự triển khai

1. Thêm Schema `SagaTransaction` và Index
2. Refactor `wallet-trans-service` theo logic Saga mới
3. Deploy `wallet-trans-service` với Graceful Shutdown
4. Triển khai Job Sweeper trong `scheduler-service`
5. Test toàn diện với các kịch bản: Happy path, Restart giữa chừng, Retry
