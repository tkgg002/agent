# Kế hoạch Refactor chi tiết: Luồng Hoàn tiền (Refund Flow)

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo không mất yêu cầu hoàn tiền khi service restart.
- **Idempotency:** Chống hoàn tiền 2 lần cho cùng một giao dịch.
- **Audit Trail:** Ghi log đầy đủ cho các hành động hoàn tiền (đặc biệt manual refund).

---

## 2. Các Service liên quan

- `payment-bill-service` (Xử lý refund request)
- `booking-ticket-service` (Refund vé)
- `wallet-service` (Hoàn tiền vào ví)

---

## 3. Phân tích luồng hiện tại

```
Trigger:
    - Auto: Saga rollback khi giao dịch thất bại
    - Manual: Admin bấm nút Refund trên CMS

         |
         v
payment-bill-service / booking-ticket-service: Tạo Refund Request
         |
         v
wallet-service: Cộng tiền vào ví user
         |
         v
notification-service: Thông báo cho user
```

**Điểm yếu:**
- Không có schema riêng cho Refund -> Khó track.
- Không có cơ chế chống duplicate refund.
- Manual refund không có audit log.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Chuẩn hóa Refund Schema**

#### 1.1. Schema `RefundRequest` (MongoDB)

```typescript
{
    refundId: string;
    originalTransactionId: string;  // ID giao dịch gốc
    originalTransactionType: 'PAYMENT_BILL' | 'BOOKING' | 'WALLET_TRANSFER';
    amount: number;
    reason: string;
    source: 'AUTO' | 'MANUAL';
    requestedBy: string;           // System hoặc Admin ID
    status: 'PENDING' | 'PROCESSING' | 'SUCCESS' | 'FAILED';
    walletTransactionId: string;   // ID giao dịch cộng tiền
    auditLog: [
        {
            action: string;
            performedBy: string;
            timestamp: Date;
            details: any;
        }
    ];
    createdAt: Date;
    updatedAt: Date;
    processedAt: Date;
}
```

**Index:**
- `UNIQUE INDEX` trên `originalTransactionId` (Chống duplicate)
- `INDEX` trên `status` + `createdAt`

### **Giai đoạn 2: Refund Service Logic**

#### 2.1. Xử lý Refund với Idempotency

**File:** `payment-bill-service/logics/refund.logic.ts`

```typescript
async processRefund(payload: RefundPayload) {
    const { originalTransactionId, amount, reason, source, requestedBy } = payload;

    // 1. Check Idempotency
    const existing = await RefundModel.findOne({ originalTransactionId });
    if (existing) {
        if (existing.status === 'SUCCESS') {
            return { success: true, duplicate: true, refundId: existing.refundId };
        }
        if (existing.status === 'PROCESSING') {
            throw new Error('Refund is already in progress');
        }
        // Nếu FAILED, cho phép retry
    }

    // 2. Tạo hoặc cập nhật refund record
    const refund = await RefundModel.findOneAndUpdate(
        { originalTransactionId },
        {
            $setOnInsert: {
                refundId: generateId(),
                originalTransactionId,
                createdAt: new Date(),
            },
            $set: {
                amount,
                reason,
                source,
                requestedBy,
                status: 'PROCESSING',
            },
            $push: {
                auditLog: {
                    action: 'REFUND_INITIATED',
                    performedBy: requestedBy,
                    timestamp: new Date(),
                    details: { amount, reason },
                },
            },
        },
        { upsert: true, new: true }
    );

    // 3. Lấy thông tin giao dịch gốc để xác định user
    const originalTx = await this.getOriginalTransaction(originalTransactionId);

    // 4. Thực hiện hoàn tiền
    try {
        const walletResult = await this.broker.call('wallet-service.credit', {
            walletId: originalTx.userId,
            amount,
            reference: `REFUND_${refund.refundId}`,
            description: `Hoàn tiền: ${reason}`,
        });

        refund.status = 'SUCCESS';
        refund.walletTransactionId = walletResult.transactionId;
        refund.processedAt = new Date();
        refund.auditLog.push({
            action: 'REFUND_SUCCESS',
            performedBy: 'SYSTEM',
            timestamp: new Date(),
            details: { walletTransactionId: walletResult.transactionId },
        });
        await refund.save();

        // 5. Thông báo user
        await this.broker.emit('notification.send', {
            userId: originalTx.userId,
            type: 'REFUND_SUCCESS',
            data: { amount, reason },
        });

        return { success: true, refundId: refund.refundId };

    } catch (error) {
        refund.status = 'FAILED';
        refund.auditLog.push({
            action: 'REFUND_FAILED',
            performedBy: 'SYSTEM',
            timestamp: new Date(),
            details: { error: error.message },
        });
        await refund.save();
        throw error;
    }
}
```

### **Giai đoạn 3: Admin Refund với Audit**

#### 3.1. API Endpoint cho Manual Refund

**File:** `admin-portal-gateway/routes/refund.route.ts`

```typescript
router.post('/refunds', authMiddleware, async (ctx) => {
    const { originalTransactionId, amount, reason } = ctx.request.body;
    const adminId = ctx.state.user.id;

    // Bắt buộc phải có reason
    if (!reason || reason.length < 10) {
        throw new Error('Reason is required (min 10 characters)');
    }

    const result = await ctx.call('payment-bill-service.processRefund', {
        originalTransactionId,
        amount,
        reason,
        source: 'MANUAL',
        requestedBy: adminId,
    });

    return result;
});
```

### **Giai đoạn 4: Integration với Saga Rollback**

Khi Saga cần rollback, gọi đến refund service:

```typescript
// Trong Saga's rollback logic
async executeRollback(saga: SagaDocument) {
    const paymentStep = saga.steps.find(s => s.name === 'PAYMENT');
    if (paymentStep?.status === 'SUCCESS') {
        await this.broker.call('payment-bill-service.processRefund', {
            originalTransactionId: paymentStep.paymentId,
            amount: saga.amount,
            reason: 'Saga rollback: ' + saga.error,
            source: 'AUTO',
            requestedBy: 'SYSTEM',
        });
    }
}
```

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `refund_requests` | `originalTransactionId` | Tạo UNIQUE INDEX |
| `wallet_logs` | `reference` | Đảm bảo UNIQUE |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Refund 2 lần cùng giao dịch | UNIQUE INDEX trên `originalTransactionId` |
| Admin gian lận | Audit Log bắt buộc, yêu cầu reason |
| Service restart giữa chừng | Status = PROCESSING, có thể retry |

---

## 7. Thứ tự triển khai

1. Tạo Schema `RefundRequest` và Index
2. Implement Refund Logic với Idempotency
3. Thêm Admin API với Audit
4. Tích hợp với Saga rollback
5. Test toàn diện
