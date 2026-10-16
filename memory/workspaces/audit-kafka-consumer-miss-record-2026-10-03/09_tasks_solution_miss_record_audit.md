# 09_tasks_solution — Đề xuất sửa miss record (PROPOSE ONLY — chờ User APPROVE)

> Brain không sửa source (Rule #13). Code dưới đây là **thiết kế minh hoạ** cho Muscle, chưa áp dụng.

## Nguyên tắc duy nhất (Single Best Approach)
**Offset chỉ được commit tới watermark liên tục của các record đã bền vững** (persist vào shadow HOẶC đã ghi DLQ thành công). Mọi record chưa bền vững phải được giữ lại/replay, TUYỆT ĐỐI không bị commit vượt qua.

## 1. `batchUpsert` trả kết quả theo từng record (thay vì chỉ `(written, err)`)

```go
// outcome của từng record sau flush
type flushOutcome struct {
    Durable map[*shadow.UpsertRecord]struct{} // persist OK hoặc DLQ OK
    Retry   []*shadow.UpsertRecord            // lỗi retryable còn lại → trả về buffer
}
```

- Lỗi non-retryable → `writeFailedSyncLog`; **chỉ coi là Durable nếu ghi DLQ thành công**, nếu DLQ lỗi → vào `Retry`.
- Lỗi retryable / `Prepare` lỗi / `schema == nil` → vào `Retry` (không `return` bỏ phần còn lại của chunk/group).
- Bỏ `return` sớm ở `chunkWritten == 0`: xử lý hết mọi chunk rồi mới tổng hợp.

## 2. Retry có giới hạn, rồi trả về buffer

```go
const maxFlushAttempts = 4
// backoff 100ms,200ms,400ms + jitter; hết lượt:
bb.requeueFront(outcome.Retry) // giữ thứ tự, KHÔNG commit qua các record này
```

## 3. Commit theo contiguous watermark

```go
// với mỗi (topic,partition): lowestUnresolved = min(offset của Retry)
// commit = (lowestUnresolved - 1) nếu có Retry, ngược lại max(offset Durable)
func watermark(batch []*shadow.UpsertRecord, retry []*shadow.UpsertRecord) map[TopicPartition]int64 { ... }
```

Chỉ gọi `onCommitOffsets` với watermark này; không còn commit "max offset của cả batch".

## 4. Serialize flush + commit
- Một mutex `flushMu` bao quanh `Flush` (timerLoop và `FlushBatchBuffer` của snapshot runner dùng chung) hoặc một committer goroutine duy nhất, loại race F5.

## 5. Backpressure
- `Add` chặn (hoặc fetch loop tạm dừng) khi `len(records) > hardLimit` (cấu hình, vd `4*maxSize`) — tránh RAM phình khi DB nghẽn (F6).

## 6. Giảm rebalance churn (F7)
- `RefreshTopics`: chỉ rebuild reader khi tập topic **khác ổn định qua ≥2 chu kỳ** + jitter ngẫu nhiên theo instance; không `ReloadAll` đồng loạt.

## 7. Ngân sách kết nối DB
- `dbPool.maxOpenConn` mỗi instance = `floor(PG_budget / N)`; xem xét pgbouncer. *(cấu hình hạ tầng — cần số liệu `max_connections` thật)*

## Kiểm thử bắt buộc (G2/G3/G4/G5/G6)
- **RED trước**: test `FlushWithContext` với DB giả lập trả deadlock `40P01` cho 1 group → khẳng định hiện tại offset VẪN bị commit (fail) → sau sửa PASS.
- Edge: `Prepare` lỗi, `schema nil`, DLQ lỗi, 2 flush đồng thời, partition có record lỗi ở giữa, buffer rỗng, offset = 0.
- Regression: chạy toàn bộ test `internal/handler/shadow`, `internal/service/shadow`.
- Dữ liệu thật: đối soát count Source = Shadow sau khi chạy lại + `/security-agent` trước khi báo Done.

## Rủi ro đã cân nhắc
- Requeue có thể gây duplicate khi retry → an toàn vì upsert idempotent + OCC `_source_ts`.
- Backpressure làm chậm fetch khi DB lỗi kéo dài → chấp nhận (đúng bản chất: không nuốt dữ liệu); cần alert lag.
