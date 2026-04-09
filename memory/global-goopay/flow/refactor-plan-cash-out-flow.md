# Kế hoạch Refactor chi tiết: Luồng Rút tiền từ ví (Cash-out / Withdrawal Flow)

## 1. Mục tiêu

- **Ổn định hệ thống:** Đảm bảo không mất giao dịch rút tiền khi Go Service hoặc Bank Handler restart.
- **Saga Pattern:** Xử lý hoàn tiền tự động nếu chuyển khoản Bank thất bại.
- **Async Processing:** Chuyển từ đồng bộ sang bất đồng bộ để tránh timeout.

---

## 2. Các Service liên quan

- `wallet-service` (Node.js - Trừ tiền ví)
- `disbursement-service` (Go - Chuyển tiền ra Bank)
- `bank-handler-service` (Go - Giao tiếp với Bank API)

---

## 3. Phân tích luồng hiện tại

```
User Request -> wallet-service.deductWallet()
                     |
                     v
            disbursement-service (Go): Gọi Bank API
                     |
                     v
            bank-handler-service (Go): Thực hiện transfer
                     |
                     v
            Response: SUCCESS/FAILED
```

**Điểm yếu:**
- Nếu `disbursement-service` restart sau khi trừ ví nhưng trước khi gọi Bank -> Tiền đã trừ nhưng chưa chuyển.
- Nếu Bank xử lý thành công nhưng response bị timeout -> Trạng thái không đồng bộ.
- Giao tiếp Node -> Go qua HTTP có thể bị đứt khi restart.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Hạ tầng Go Services**

#### 1.1. Graceful Shutdown cho Go (disbursement-service)

**File:** `main.go`

```go
func main() {
    app := fiber.New()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    go func() {
        if err := app.Listen(":3000"); err != nil {
            log.Fatal(err)
        }
    }()

    <-quit
    log.Println("Gracefully shutting down...")

    // Chờ tối đa 60s để xử lý nốt request
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    if err := app.ShutdownWithContext(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    // Đóng DB connection
    sqlDB.Close()
    redisClient.Close()
    
    log.Println("Server exited")
}
```

#### 1.2. Tách Context cho Critical Section

**File:** `handlers/withdrawal.handler.go`

```go
func (h *Handler) ProcessWithdrawal(c *fiber.Ctx) error {
    payload := new(WithdrawalPayload)
    c.BodyParser(payload)

    // QUAN TRỌNG: Dùng Background Context cho DB Transaction
    // Không dùng c.Context() vì nó sẽ bị cancel khi HTTP connection đóng
    dbCtx := context.Background()
    
    tx, _ := h.db.BeginTx(dbCtx, nil)
    defer tx.Rollback()

    // Thực hiện nghiệp vụ với dbCtx
    result, err := h.bankService.Transfer(dbCtx, payload)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    tx.Commit()
    return c.JSON(result)
}
```

### **Giai đoạn 2: Chuyển sang NATS JetStream (Async)**

**Mục đích:** Thay thế HTTP call trực tiếp từ Node -> Go.

#### 2.1. Producer (wallet-service - Node.js)

```typescript
async initiateWithdrawal(payload: WithdrawalPayload) {
    // 1. Tạo record Withdrawal với status PENDING
    const withdrawal = await WithdrawalModel.create({
        requestId: payload.requestId,
        userId: payload.userId,
        amount: payload.amount,
        bankAccount: payload.bankAccount,
        status: 'PENDING',
    });

    // 2. Trừ tiền ví (Soft hold)
    await this.broker.call('wallet-service.holdBalance', {
        walletId: payload.walletId,
        amount: payload.amount,
        reference: withdrawal._id,
    });

    // 3. Publish event cho Go service xử lý
    await this.broker.emit('withdrawal.requested', {
        withdrawalId: withdrawal._id,
        amount: payload.amount,
        bankAccount: payload.bankAccount,
    });

    // 4. Trả về ngay, không chờ Bank
    return { withdrawalId: withdrawal._id, status: 'PROCESSING' };
}
```

#### 2.2. Consumer (disbursement-service - Go)

**File:** `workers/withdrawal_worker.go`

```go
func (w *Worker) SubscribeWithdrawal() {
    // Subscribe vào NATS JetStream
    sub, _ := w.js.PullSubscribe("withdrawal.requested", "disbursement-group", 
        nats.ManualAck(),
        nats.AckWait(60*time.Second),
    )

    for {
        msgs, _ := sub.Fetch(1, nats.MaxWait(5*time.Second))
        for _, msg := range msgs {
            var payload WithdrawalPayload
            json.Unmarshal(msg.Data, &payload)

            err := w.processWithdrawal(payload)
            if err != nil {
                // Nack để retry
                msg.Nak()
                continue
            }

            // Chỉ Ack khi đã xử lý xong
            msg.Ack()
        }
    }
}

func (w *Worker) processWithdrawal(payload WithdrawalPayload) error {
    // 1. Gọi Bank API
    result, err := w.bankHandler.Transfer(payload)
    if err != nil {
        // Publish event lỗi để Node rollback
        w.nc.Publish("withdrawal.failed", json.Marshal(map[string]interface{}{
            "withdrawalId": payload.WithdrawalId,
            "error": err.Error(),
        }))
        return err
    }

    // 2. Publish event thành công
    w.nc.Publish("withdrawal.completed", json.Marshal(map[string]interface{}{
        "withdrawalId": payload.WithdrawalId,
        "bankRef": result.TransactionRef,
    }))

    return nil
}
```

#### 2.3. Event Handlers (wallet-service - Node.js)

```typescript
events: {
    "withdrawal.completed": {
        group: "wallet-service",
        async handler(ctx: Context<WithdrawalCompletedPayload>) {
            const { withdrawalId, bankRef } = ctx.params;
            
            // Xác nhận trừ tiền (chuyển từ hold sang deducted)
            await this.broker.call('wallet-service.confirmDeduct', {
                reference: withdrawalId,
            });

            // Update trạng thái
            await WithdrawalModel.updateOne(
                { _id: withdrawalId },
                { $set: { status: 'SUCCESS', bankRef } }
            );
        }
    },
    "withdrawal.failed": {
        group: "wallet-service",
        async handler(ctx: Context<WithdrawalFailedPayload>) {
            const { withdrawalId, error } = ctx.params;
            
            // Hoàn tiền (release hold)
            await this.broker.call('wallet-service.releaseHold', {
                reference: withdrawalId,
            });

            // Update trạng thái
            await WithdrawalModel.updateOne(
                { _id: withdrawalId },
                { $set: { status: 'FAILED', error } }
            );
        }
    }
}
```

### **Giai đoạn 3: Job Sweeper (Tra soát)**

**Service:** `reconcile-service` (Go)

```go
func (s *Sweeper) SweepPendingWithdrawals() {
    // Tìm các withdrawal PENDING > 15 phút
    pendingList := s.repo.FindPendingWithdrawals(15 * time.Minute)

    for _, w := range pendingList {
        // Gọi API tra soát Bank
        status, err := s.bankHandler.QueryTransaction(w.BankRef)
        if err != nil {
            continue
        }

        if status == "SUCCESS" {
            // Cập nhật DB
            s.repo.UpdateStatus(w.ID, "SUCCESS")
            // Publish event
            s.nc.Publish("withdrawal.completed", ...)
        } else if status == "FAILED" || status == "NOT_FOUND" {
            s.nc.Publish("withdrawal.failed", ...)
        }
    }
}
```

---

## 5. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `withdrawals` | `requestId` | Tạo UNIQUE INDEX |
| `wallet_holds` | `reference` | Tạo UNIQUE INDEX |
| `disbursement_logs` (Go) | `withdrawal_id` | Tạo UNIQUE INDEX |

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Go service crash giữa chừng | NATS JetStream tự động requeue message (Manual ACK) |
| Bank timeout | Tra soát định kỳ qua Job Sweeper |
| Node gọi Go bị đứt | Chuyển sang Event-Driven (NATS) thay vì HTTP |
| Duplicate withdrawal | Idempotency Key (requestId) |

---

## 7. Thứ tự triển khai

1. Implement Graceful Shutdown cho Go services (signal.Notify)
2. Tạo Schema mới / Index cho Idempotency
3. Thiết lập NATS JetStream Stream
4. Refactor wallet-service: Producer
5. Refactor disbursement-service: Consumer
6. Triển khai Event Handlers cho completion/failure
7. Triển khai Job Sweeper trong reconcile-service
8. Test toàn diện
