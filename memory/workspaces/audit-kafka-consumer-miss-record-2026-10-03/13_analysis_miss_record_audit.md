# 13_analysis — Audit: Source → Shadow miss record khi tăng số Kafka consumer (production)

- Ngày: 2026-10-03
- Phạm vi: `data-hub/centralized-data-service` (đường Kafka → BatchBuffer → Shadow). Đọc code tĩnh, KHÔNG có log/metric/DB production.
- Trạng thái: **Static audit — chưa runtime-verified.** Mọi kết luận ghi rõ `file:dòng`. Phần chưa chứng minh được nằm ở mục 5.

## 1. Luồng thực tế (đã đọc code)

```
Kafka ──FetchMessage──► KafkaConsumer.Start loop (1 goroutine, 1 reader/GroupTopics)
        ──processMessage──► EventHandler.HandleRaw ──► processEvent
        ──► BatchBuffer.Add(record)  [chỉ append RAM, return rows=1 "success"]
                                   │
   timerLoop (5s / 1000 rec) ─► FlushWithContext ─► batchUpsert (≤20 group song song)
                                   │
                                   └─► onCommitOffsets(max offset / partition) ─► reader.CommitMessages
```

## 2. Findings đã xác minh bằng code

| ID | Mức | Phát hiện | Bằng chứng |
|----|-----|-----------|-----------|
| F1 | Critical | **Ack-before-persist**: `processEvent` chỉ `batchBuffer.Add` rồi trả `rows=1`; consumer ghi `success`. "Success" = đã vào RAM, chưa ghi DB. | `event_handler.go:460-462`, `kafka_consumer.go:473-483` |
| F2 | **Critical (root cause chính)** | **Commit offset vô điều kiện sau flush**: sau `g.Wait()` luôn gọi `onCommitOffsets` cho TOÀN BỘ `batch`, không kiểm tra `err`/record nào ghi thất bại. Offset = max theo partition. | `batch_buffer.go:276-292` (max: dòng 284-286) |
| F3 | **Critical** | Các đường `batchUpsert` **drop record không DLQ** rồi F2 vẫn commit → Kafka không replay: (a) `PrepareForCDCInsertInSchema` lỗi → bỏ cả group; (b) `schema == nil`; (c) lỗi "retryable" (deadlock 40P01, 40001, 08xxx, 57P0x, "connection reset/refused", "bad connection") → `return written+chunkWritten, err` ngay: phần còn lại của chunk + mọi chunk sau KHÔNG được thử, KHÔNG ghi `failed_sync_logs`; (d) `chunkWritten==0` return sớm bỏ các chunk sau; (e) `writeFailedSyncLog` lỗi chỉ tăng metric `DLQWriteFail`. | `batch_buffer.go:362-374`, `474-479`, `500-505`, `522-525`, `586-592`, `596-628` |
| F4 | High | Cơ chế "chặn commit offset khi lỗi" ở consumer loop **vô hiệu**: commit theo max-offset/partition nên message sau thành công đẩy watermark vượt message lỗi. Nếu `writeDLQ` cũng lỗi (ghi vào system DB `kc.db`) → `continue`, message mất. | `kafka_consumer.go:500-526`, `dlq_helper.go:54` |
| F5 | High | **Hai điểm gọi Flush đồng thời** trên cùng 1 BatchBuffer: `timerLoop` và `FlushBatchBuffer` (snapshot runner). Không có contiguous-watermark → commit có thể đi lùi/vượt. | `batch_buffer.go:180-195`, `event_handler.go:97-102`, `server_setup.go:375-388` |
| F6 | Medium | **Không có backpressure**: `Add` không chặn, fetch loop chạy tiếp khi flush chậm → RAM phình khi DB nghẽn. | `batch_buffer.go:166-178` |
| F7 | Medium | **Rebalance churn**: mỗi instance chạy `RefreshTopics` mỗi 60s (kèm `ReloadAll`); nếu tập topic khác nhau → `Close()` reader + `buildReader` → leave/join group → rebalance toàn nhóm. `flushAllBatches` ở đây chỉ flush **stats/activity**, không flush BatchBuffer. | `kafka_consumer.go:264-289`, `topic_helper.go:43-111`, `adaptive_batcher.go:123-130` |
| F8 | Medium | OCC guard dùng `chunk[0].SourceTsMs>0` cho cả chunk; so sánh `_source_ts` **strict `<`** nên event cùng ms bị bỏ im lặng (RowsAffected=0, không lỗi). Bảng không có `_hash`/`_raw_data` → change-gate chỉ so `_deleted` ⇒ update field nghiệp vụ không ghi. (Gây **lệch**, không gây thiếu row.) | `schema_adapter.go:644`, `766-816` |

## 3. Vì sao "nhiều consumer" làm miss xuất hiện (cơ chế, mức chắc chắn)

Chuỗi nhân quả đã chứng minh được bằng code: **lỗi DB tạm thời/quá tải → F3 (drop không DLQ) → F2 (commit vẫn đi) → mất vĩnh viễn.** Tăng số consumer chỉ làm tăng tần suất "lỗi DB tạm thời":

1. **Áp lực kết nối/lock lên cùng PG** *(cơ chế chắc chắn; số liệu cần verify)*: mỗi instance cấu hình `dbPool.maxOpenConn=100`, `worker.poolSize=50`, flush song song 20 group (`config-production.yml:17-20,78-86`, `batch_buffer.go:235`). N instance nhân lên theo N trên cùng PG → `too many clients`/conn reset/timeout/deadlock → đều rơi vào nhóm retryable của F3. `max_connections` thực tế: **chưa biết**.
2. **Rebalance (F7)**: thêm/bớt consumer + refresh topic không đồng bộ ⇒ nhiều lần rebalance; trong lúc đó flush kéo dài, commit đi qua generation mới, tăng xác suất F5/F3.
3. **Trigger/DDL đồng thời** *(giả thuyết, cần verify)*: `PrepareForCDCInsertInSchema` có thể chạy `ALTER TABLE/ADD CONSTRAINT` lần đầu (`schema_adapter.go:196-243`) — nhiều instance cùng khởi động → lock contention; nếu có trigger trên bảng đích (như trigger sync `transaction_history` của bạn) thì footprint lock của mỗi batch lớn hơn.

## 4. Giả thuyết đã loại (có chứng cứ)

- **Registry reload tạo cửa sổ cache rỗng → "route not found"**: loại. `ReloadAll` query ngoài lock rồi swap dưới `rs.mu.Lock()`, `ResolveSourceRoutes` dùng `RLock` (`metadata_registry_service.go:161-183`, `592-593`).
- **OCC làm mất row INSERT**: loại cho trường hợp thiếu row — OCC chỉ chặn khi row đã tồn tại; nó gây *lệch giá trị* (F8), không gây *thiếu row*.

## 5. Chưa chứng minh được (cần dữ liệu production)

- Prod chạy `cmd/worker` (BatchBuffer) hay `cmd/sinkworker` (ghi đồng bộ từng message, commit sau khi ghi OK — **không** dính F1-F3). Bảng `cdc_transaction_history` bạn gửi có quy ước `ux_/ix_` khớp `master_ddl_generator.go:220-222,299` (tức lớp **Master**, do Transmuter ghi) → audit này chỉ áp dụng cho hop Source→Shadow.
- Số instance N, `max_connections`, số partition/topic, tần suất rebalance.
- Thực sự có record rơi qua F3 hay không (xem checklist ở `06`/`08`).

## 6. Checklist xác minh (đọc-chỉ, không sửa data)

1. **Chứng minh silent-drop**: lấy vài `_source_id` đang thiếu → tìm trong `cdc_system.failed_sync_logs`. *Có trong DLQ* ⇒ lỗi non-retryable đã DLQ (replay được). *Không có ở cả shadow lẫn DLQ* ⇒ rơi vào F3.
2. **Log app** quanh thời điểm miss: `"batch upsert failed"` có `persisted < count` (`batch_buffer.go:247-253`), `"upsert failed"`, `"soft delete failed"`, ngay sau đó `"kafka batch commit success"` (`kafka_consumer.go:802`).
3. **PG**: `pg_stat_database.deadlocks`; `count(*) pg_stat_activity` vs `max_connections`; log PG `deadlock detected` / `too many clients` / `terminating connection`.
4. **Kafka**: `kafka-consumer-groups --describe --group <groupId>` (lag, member count, generation); tần suất log `"topic set changed recreating reader"` (`topic_helper.go:49`); broker log `Preparing to rebalance`.
5. **Metrics**: `DLQWriteFail`, `SyncFailed`, `EventsDropped{reason}`, `ConsumerLag`.

## 7. Đối chiếu Lessons đã có

- Lặp lại pattern `[2026-09-17] DLQ Silent Offset Commit` và `Broker Silent Drop`: nguyên tắc "không commit khi ghi chưa thành công" **đã bị phá ở tầng BatchBuffer (F2/F3)**, dù tầng consumer có comment chặn (F4).
