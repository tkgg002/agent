# Kế hoạch Refactor chi tiết: Luồng Chi hộ (Disbursement Flow)

## 1. Mục tiêu

- **Ổn định Go Service:** Đảm bảo `disbursement-service` (Go) không mất job khi restart.
- **Xử lý Batch:** Hỗ trợ chi hộ hàng loạt (bulk) một cách tin cậy.
- **Retry \u0026 Recovery:** Tự động retry các giao dịch thất bại và có cơ chế tra soát.

---

## 2. Các Service liên quan

- `disbursement-service` (Go 1.23, Fiber, MongoDB, Redis)
- `wallet-service` (Node.js - Chi vào ví GooPay)
- `external-adapter-service` (Node.js - Gọi ra Bank/Partner)

---

## 3. Phân tích luồng hiện tại

```
Merchant/Admin tạo Phiếu Chi (Disbursement Ticket)
         |
         v
disbursement-service (Go): Xử lý từng dòng trong phiếu
         |
         +--> Nếu chi vào ví: wallet-service.credit()
         |
         +--> Nếu chi ra bank: external-adapter-service.transfer()
```

**Điểm yếu:**
- Go service restart khi đang xử lý batch -> Mất trạng thái, phải xử lý lại từ đầu hoặc bỏ sót.
- Không có cơ chế resume cho batch job.
- Gọi Node.js từ Go qua HTTP có thể timeout.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Go Service**

#### 1.1. Graceful Shutdown với Worker Pool

**File:** `main.go`

```go
func main() {
    app := fiber.New()
    
    // Khởi tạo Worker Pool
    workerPool := workers.NewDisbursementWorkerPool(5) // 5 workers
    workerPool.Start()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    go func() {
        app.Listen(":3000")
    }()

    <-quit
    log.Println("Shutting down...")

    // 1. Dừng nhận job mới
    workerPool.StopAccepting()

    // 2. Chờ các job đang chạy hoàn thành (max 60s)
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    workerPool.WaitWithContext(ctx)

    // 3. Tắt HTTP Server
    app.ShutdownWithContext(ctx)

    log.Println("Server exited")
}
```

#### 1.2. Worker Pool với WaitGroup

**File:** `workers/pool.go`

```go
type DisbursementWorkerPool struct {
    wg          sync.WaitGroup
    jobQueue    chan DisbursementJob
    stopAccept  bool
    mutex       sync.Mutex
}

func (p *DisbursementWorkerPool) Submit(job DisbursementJob) error {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    if p.stopAccept {
        return errors.New("worker pool is shutting down")
    }

    p.wg.Add(1)
    p.jobQueue <- job
    return nil
}

func (p *DisbursementWorkerPool) worker() {
    for job := range p.jobQueue {
        p.processJob(job)
        p.wg.Done()
    }
}

func (p *DisbursementWorkerPool) WaitWithContext(ctx context.Context) {
    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("All jobs completed")
    case <-ctx.Done():
        log.Println("Timeout waiting for jobs")
    }
}
```

### **Giai đoạn 2: Persistent Job State**

**Mục đích:** Lưu trạng thái job vào DB để có thể resume sau restart.

#### 2.1. Schema `DisbursementJob` (MongoDB)

```go
type DisbursementJob struct {
    ID              primitive.ObjectID `bson:"_id"`
    TicketID        string             `bson:"ticket_id"`
    LineIndex       int                `bson:"line_index"`
    RecipientType   string             `bson:"recipient_type"` // WALLET | BANK
    RecipientInfo   bson.M             `bson:"recipient_info"`
    Amount          float64            `bson:"amount"`
    Status          string             `bson:"status"` // PENDING | PROCESSING | SUCCESS | FAILED
    RetryCount      int                `bson:"retry_count"`
    Error           string             `bson:"error,omitempty"`
    BankRef         string             `bson:"bank_ref,omitempty"`
    CreatedAt       time.Time          `bson:"created_at"`
    UpdatedAt       time.Time          `bson:"updated_at"`
    ProcessedAt     *time.Time         `bson:"processed_at,omitempty"`
}
```

**Index:**
- `INDEX` trên `ticket_id` + `status` (để query theo ticket)
- `INDEX` trên `status` + `updated_at` (để Sweeper quét)

#### 2.2. Job Processing Logic

**File:** `workers/disbursement_worker.go`

```go
func (w *Worker) processJob(job *DisbursementJob) {
    // 1. Cập nhật status = PROCESSING
    w.repo.UpdateStatus(job.ID, "PROCESSING")

    var err error
    var result interface{}

    // 2. Xử lý theo loại recipient
    switch job.RecipientType {
    case "WALLET":
        result, err = w.creditWallet(job)
    case "BANK":
        result, err = w.transferBank(job)
    }

    // 3. Cập nhật kết quả
    if err != nil {
        job.RetryCount++
        if job.RetryCount >= 3 {
            job.Status = "FAILED"
            job.Error = err.Error()
        } else {
            job.Status = "PENDING" // Sẽ được retry bởi Sweeper
        }
    } else {
        job.Status = "SUCCESS"
        job.ProcessedAt = timePtr(time.Now())
        if bankResult, ok := result.(*BankTransferResult); ok {
            job.BankRef = bankResult.TransactionRef
        }
    }

    w.repo.Update(job)
}
```

### **Giai đoạn 3: Startup Recovery**

**Mục đích:** Khi service khởi động lại, resume các job đang dở.

**File:** `main.go`

```go
func main() {
    // ... khởi tạo app, db...

    // Resume pending jobs khi startup
    go func() {
        time.Sleep(5 * time.Second) // Đợi service ổn định
        recoverPendingJobs()
    }()

    // ... listen server
}

func recoverPendingJobs() {
    // Tìm các job đang PROCESSING (bị interrupt do restart)
    // và chuyển về PENDING để xử lý lại
    jobs := repo.FindByStatus("PROCESSING")
    for _, job := range jobs {
        job.Status = "PENDING"
        repo.Update(job)
        log.Printf("Recovered job: %s", job.ID.Hex())
    }

    // Đẩy các job PENDING vào queue
    pendingJobs := repo.FindByStatus("PENDING")
    for _, job := range pendingJobs {
        workerPool.Submit(job)
    }
}
```

### **Giai đoạn 4: Retry \u0026 Sweeper**

**File:** `workers/sweeper.go`

```go
func (s *Sweeper) Run() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        s.sweepFailedJobs()
        s.sweepStuckJobs()
    }
}

func (s *Sweeper) sweepFailedJobs() {
    // Tìm job FAILED có retryCount < 3 và updatedAt > 10 phút
    jobs := s.repo.FindRetryableJobs()
    for _, job := range jobs {
        job.Status = "PENDING"
        s.repo.Update(job)
        s.workerPool.Submit(job)
    }
}

func (s *Sweeper) sweepStuckJobs() {
    // Tìm job PROCESSING > 15 phút (bị stuck)
    jobs := s.repo.FindStuckJobs(15 * time.Minute)
    for _, job := range jobs {
        // Tra soát với Bank nếu là BANK type
        if job.RecipientType == "BANK" && job.BankRef != "" {
            status := s.bankHandler.QueryTransaction(job.BankRef)
            if status == "SUCCESS" {
                job.Status = "SUCCESS"
            } else {
                job.Status = "PENDING"
                job.RetryCount++
            }
        } else {
            job.Status = "PENDING"
        }
        s.repo.Update(job)
    }
}
```

---

## 5. Giao tiếp Go -> Node.js

#### Option A: HTTP với Retry (Đơn giản)

```go
func (w *Worker) creditWallet(job *DisbursementJob) (interface{}, error) {
    client := retryablehttp.NewClient()
    client.RetryMax = 3
    client.RetryWaitMin = 1 * time.Second
    client.RetryWaitMax = 5 * time.Second

    resp, err := client.Post(
        "http://wallet-service:3000/internal/credit",
        "application/json",
        bytes.NewBuffer(payload),
    )
    // ...
}
```

#### Option B: NATS (Khuyến nghị cho hệ thống lớn)

Tương tự như Cash-out Flow, dùng NATS JetStream để decouple.

---

## 6. Checklist Idempotency

| Bảng/Collection | Cột Unique | Action |
|---|---|---|
| `disbursement_jobs` | `ticket_id` + `line_index` | Tạo UNIQUE INDEX |
| `wallet_logs` | `reference` (= job_id) | Đảm bảo có UNIQUE INDEX |

---

## 7. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| Go restart khi đang xử lý | Lưu state vào DB, resume khi startup |
| Job bị stuck | Sweeper quét định kỳ |
| Timeout gọi Node | HTTP Retry với exponential backoff |
| Duplicate processing | Idempotency Key (ticket_id + line_index) |

---

## 8. Thứ tự triển khai

1. Implement Graceful Shutdown với Worker Pool
2. Tạo Schema `DisbursementJob` và Index
3. Implement Job Processing với state persistence
4. Implement Startup Recovery
5. Implement Sweeper
6. Test toàn diện với các kịch bản restart
