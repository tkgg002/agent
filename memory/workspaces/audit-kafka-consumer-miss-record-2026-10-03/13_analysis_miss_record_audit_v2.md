# 13 — Analysis v2: Miss record Source → Shadow khi nhiều Kafka consumer

> 2026-10-04 · Brain · Phân tích tĩnh `data-hub/centralized-data-service`, từng flow, từng dòng. Verdict Laya xem `06_validation_miss_record_audit_v2.md`.
> **Mọi mục dưới đây là "đường code tồn tại"; chưa có bằng chứng prod.**

## 1. Cơ chế gốc (VERIFIED code): "Commit offset ≠ đã persist"

```
Fetch (loop đơn, kafka_consumer.go:318)
  → processMessage (decode) → HandleRaw → processEvent
  → batchBuffer.Add(record)            event_handler.go:460   ← trả rows=1 = "success"
  ...(≤5s hoặc ≥maxSize)...
  → timerLoop → FlushWithContext       batch_buffer.go:204
       g.Go(batchUpsert) per table     :237-274  (lỗi chỉ log, err=first error)
       g.Wait()                        :276
       onCommitOffsets(max offset/partition của TOÀN batch)   :279-291   ← KHÔNG xét err
  → commitOffsets → reader.CommitMessages   kafka_consumer.go:761-807
```

Hệ quả: bất kỳ record nào **không persist** trong `batchUpsert` mà không được DLQ thành công đều bị offset commit "nuốt".

## 2. Các đường làm record biến mất sau khi offset được commit

| # | Đường | Dòng | Lệ thuộc số consumer? | Trạng thái |
|---|---|---|---|---|
| P1 | Prepare/schema lỗi → `return 0, err` không DLQ | `batch_buffer.go:362-374` | Gián tiếp (DB stress) | VERIFIED code |
| P2 | Fallback gặp lỗi *retryable* (deadlock/40001/57P01/08xxx/conn reset) → `return`, bỏ phần còn lại của chunk + mọi chunk sau, không ghi `failed_sync_logs` | `:474-479, :500-505, :384` | **Có** (DB tải cao/failover) | VERIFIED code |
| P3 | Lỗi **53300 too many clients** (và cả class 53) KHÔNG nằm trong `isRetryableDBError` → đi `writeFailedSyncLog`; nếu DLQ ghi cũng lỗi → chỉ `DLQWriteFail.Inc()` | `:596-628`, `:492-506`, `:578-593` | **Có** — số kết nối = N instance × (100 gorm + 100 pgx) | Code VERIFIED; việc chạm `max_connections` = **HYPOTHESIS** |
| P4 | `chunkWritten==0` sau fallback → `return` bỏ chunk sau | `:522-525` | Không | VERIFIED code (hiếm) |
| P5 | Flush chạy đồng thời (timerLoop + `FlushBatchBuffer` của snapshot runner / recon heal) → batch sau (offset cao) commit trước batch trước còn đang chạy/thất bại | `:217-225`; callers `snapshot_runner_handler.go:827,889`, `recon_heal_fetch.go:91,104` | Không trực tiếp | VERIFIED cấu trúc; cần snapshot/recon chạy cùng CDC |
| P6 | Message lỗi (mapper/route) → DLQ OK nhưng offset bị vượt bởi message sau cùng partition → DLQ là bản sao duy nhất; nếu DLQ ghi lỗi → `continue`, vẫn bị vượt | `kafka_consumer.go:500-526` | Gián tiếp | VERIFIED code |
| P7 | Silent drop `return 0,nil`: empty value, foreign topic, `nil after` (non-delete), **missing PK** | `kafka_consumer.go:538-555,631-640`; `event_handler.go:343-350` | Không | VERIFIED code; chỉ thấy qua `EventsDropped{reason}` |
| P8 | `resolveDB` rơi về control-plane DB khi `ConnectionKey` rỗng | `batch_buffer_utils.go:41-58`; `metadata_registry_service.go:254` | Không | Code VERIFIED; trigger UNVERIFIED |

## 3. Rebalance / nhiều consumer — kết luận có kiểm soát

- Rebalance/rebuild reader **tự nó** → *redelivery (trùng)*, không miss (commit chỉ sau flush, `CommitInterval:0`) — **code-reasoned (C9)**, chưa chạy thực nghiệm.
- Rebuild reader chỉ xảy ra khi topic-set đổi (`topic_helper.go:43-47`) ⇒ nhận định "churn mỗi 60s" **bị bác**.
- Nhiều instance ⇒ ghi trùng + ghi lệch thứ tự giữa 2 owner → ảnh hưởng **drift** (OCC `_source_ts` strict `<`, `schema_adapter.go:644`), không phải thiếu.
- **Cầu nối "nhiều consumer → miss" có cơ sở code duy nhất:** tăng instance ⇒ tăng kết nối/tải DB ⇒ lỗi DB (53300 / deadlock / failover) ⇒ P2/P3 ⇒ commit offset nuốt record. Đây là **giả thuyết cần kiểm bằng số liệu PG prod**, không phải kết luận.

## 4. Phát hiện phụ (không gây miss)
- `kc.batches` map không khoá, `RefreshTopics` chạy từ NATS callback goroutine (`server_setup.go:506-508`) — rủi ro `fatal error: concurrent map iteration and map write` khi topic-set đổi → crash → redelivery, không mất.
- `Subscribe` (không queue group) ⇒ mọi instance cùng nhận lệnh refresh.

## 5. Checklist xác nhận trên production (user chạy)
1. `SELECT count(*) FROM pg_stat_activity` vs `SHOW max_connections` (shadow + control-plane) tại thời điểm miss.
2. Log worker: `"batch upsert failed"` có `persisted < count`; `"soft delete failed"`/`"upsert failed"`; `"failed to write failed sync log to database"`.
3. Metric: `EventsDropped{reason}`, `DLQWriteFail`, `SyncFailed`, `ConsumerLag`.
4. Bảng `cdc_system.failed_sync_logs` theo `kafka_topic/partition/offset` quanh thời điểm miss.
5. PG log: `too many clients`, `deadlock detected`, failover.
6. `kafka-consumer-groups --describe --group <groupId>`: số member, rebalance count.
7. Xác nhận prod chạy `cmd/worker` hay `cmd/sinkworker` (sinkworker commit sau upsert đồng bộ — không dính P1–P6).

## 6. Mức tin cậy
- Cao (đọc trực tiếp, cấu trúc không mơ hồ): P1, P2, P4, P5, P6, P7.
- Trung bình (cần số liệu prod): P3 (hệ quả `max_connections`), P8.
- Laya: không phải verifier ổn định (xem 06); chỉ dùng làm tín hiệu phụ.
