# Kế hoạch Refactor chi tiết: Luồng Đối soát (Reconciliation Flow)

## 1. Mục tiêu

- **Ổn định Go Service:** Đảm bảo `reconcile-service` (Go) xử lý đối soát không bị gián đoạn.
- **Batch Processing:** Xử lý file đối soát lớn một cách hiệu quả.
- **Alert & Reporting:** Cảnh báo khi có bất thường.

---

## 2. Các Service liên quan

- `reconcile-service` (Go 1.22, Echo, MySQL, GORM, Redis, NATS)
- `scheduler-service` (Node.js - Trigger cronjob)

---

## 3. Phân tích luồng hiện tại

```
Cronjob (mỗi ngày/giờ) -> scheduler-service: Trigger reconcile
         |
         v
reconcile-service (Go): 
    1. Lấy file đối soát từ Bank/Partner (SFTP/API)
    2. Parse file
    3. So khớp với dữ liệu trong DB
    4. Ghi kết quả (Matched, Unmatched, Disputed)
    5. Alert nếu có bất thường
```

**Điểm yếu:**
- File lớn có thể mất nhiều thời gian xử lý.
- Nếu Go service restart giữa chừng -> Mất tiến độ, phải chạy lại từ đầu.
- Không có cơ chế resume.

---

## 4. Kế hoạch thực thi chi tiết

### **Giai đoạn 1: Ổn định Go Service**

#### 1.1. Graceful Shutdown

**File:** `main.go`

```go
func main() {
    e := echo.New()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    go func() {
        e.Start(":8080")
    }()

    <-quit
    log.Println("Shutting down...")

    // Dừng cronjob scheduler
    cronScheduler.Stop()

    // Chờ các job đang chạy hoàn thành
    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // 2 phút
    defer cancel()
    
    reconcileWorker.WaitWithContext(ctx)
    e.Shutdown(ctx)
}
```

### **Giai đoạn 2: Persistent Job State**

#### 2.1. Schema `ReconcileJob` (MySQL)

```sql
CREATE TABLE reconcile_jobs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(50) UNIQUE NOT NULL,
    partner_code VARCHAR(20) NOT NULL,
    file_date DATE NOT NULL,
    file_path VARCHAR(255),
    total_records INT DEFAULT 0,
    processed_records INT DEFAULT 0,
    matched_count INT DEFAULT 0,
    unmatched_count INT DEFAULT 0,
    disputed_count INT DEFAULT 0,
    status ENUM('PENDING', 'DOWNLOADING', 'PROCESSING', 'COMPLETED', 'FAILED') DEFAULT 'PENDING',
    error TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_status_created (status, created_at),
    INDEX idx_partner_date (partner_code, file_date)
);
```

#### 2.2. Checkpoint Processing

**Mục đích:** Lưu tiến độ sau mỗi N records để có thể resume.

**File:** `workers/reconcile_worker.go`

```go
func (w *Worker) processFile(job *ReconcileJob) error {
    file, _ := os.Open(job.FilePath)
    scanner := bufio.NewScanner(file)

    lineNumber := 0
    
    // Resume từ vị trí đã xử lý
    for i := 0; i < job.ProcessedRecords; i++ {
        scanner.Scan()
        lineNumber++
    }

    batchSize := 100
    batch := make([]ReconcileRecord, 0, batchSize)

    for scanner.Scan() {
        lineNumber++
        record := w.parseLine(scanner.Text())
        batch = append(batch, record)

        if len(batch) >= batchSize {
            w.processBatch(job, batch)
            batch = batch[:0]

            // Checkpoint: Cập nhật tiến độ
            job.ProcessedRecords = lineNumber
            w.repo.Update(job)
        }
    }

    // Xử lý batch còn lại
    if len(batch) > 0 {
        w.processBatch(job, batch)
    }

    job.Status = "COMPLETED"
    job.CompletedAt = timePtr(time.Now())
    w.repo.Update(job)

    return nil
}
```

### **Giai đoạn 3: Startup Recovery**

**File:** `main.go`

```go
func recoverIncompleteJobs() {
    jobs := repo.FindByStatus("PROCESSING", "DOWNLOADING")
    
    for _, job := range jobs {
        log.Printf("Resuming job: %s (processed: %d/%d)", 
            job.JobID, job.ProcessedRecords, job.TotalRecords)
        
        // Đẩy vào queue để worker xử lý tiếp
        workerQueue <- job
    }
}
```

### **Giai đoạn 4: Alert & Reporting**

#### 4.1. Threshold Alerting

```go
func (w *Worker) checkAndAlert(job *ReconcileJob) {
    // Alert nếu unmatched > 5%
    unmatchedRate := float64(job.UnmatchedCount) / float64(job.TotalRecords) * 100
    if unmatchedRate > 5 {
        w.alertService.SendSlack(AlertPayload{
            Level:   "WARNING",
            Title:   fmt.Sprintf("High unmatched rate: %.2f%%", unmatchedRate),
            Details: fmt.Sprintf("Partner: %s, Date: %s", job.PartnerCode, job.FileDate),
        })
    }

    // Alert nếu disputed > 0
    if job.DisputedCount > 0 {
        w.alertService.SendSlack(AlertPayload{
            Level:   "CRITICAL",
            Title:   fmt.Sprintf("Disputed transactions found: %d", job.DisputedCount),
            Details: "Manual review required",
        })
    }
}
```

### **Giai đoạn 5: Parallel Processing (Optional)**

Nếu file quá lớn, chia thành nhiều chunk và xử lý song song:

```go
func (w *Worker) processFileParallel(job *ReconcileJob) error {
    chunks := w.splitFile(job.FilePath, 10000) // 10k records per chunk

    var wg sync.WaitGroup
    results := make(chan ChunkResult, len(chunks))

    for _, chunk := range chunks {
        wg.Add(1)
        go func(c Chunk) {
            defer wg.Done()
            result := w.processChunk(c)
            results <- result
        }(chunk)
    }

    wg.Wait()
    close(results)

    // Aggregate results
    for result := range results {
        job.MatchedCount += result.Matched
        job.UnmatchedCount += result.Unmatched
    }

    return nil
}
```

---

## 5. K8s Configuration

```yaml
terminationGracePeriodSeconds: 180  # 3 phút cho file lớn
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2"
```

---

## 6. Rủi ro và Giải pháp

| Rủi ro | Giải pháp |
|---|---|
| File quá lớn, OOM | Streaming processing, không load toàn bộ vào memory |
| Service restart giữa chừng | Checkpoint processing, resume từ vị trí đã xử lý |
| Unmatched transactions nhiều | Threshold alerting |
| Timeout download file | Retry với exponential backoff |

---

## 7. Thứ tự triển khai

1. Tạo Schema `ReconcileJob`
2. Implement Checkpoint Processing
3. Implement Startup Recovery
4. Cấu hình Graceful Shutdown (2 phút)
5. Implement Alert Service
6. (Optional) Parallel Processing
7. Test với file lớn và kịch bản restart
