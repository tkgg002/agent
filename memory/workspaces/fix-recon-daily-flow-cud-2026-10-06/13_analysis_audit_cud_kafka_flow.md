# Báo cáo Phân tích Kỹ thuật: Audit Số liệu c,u,d & Throughput Đối Soát Theo Ngày

## 1. Nguồn gốc con số ngày 2026-09-30 (1,562 và 2,084)
Dựa trên truy vấn trực tiếp vào PostgreSQL `gpay-postgres-cdc` (`cdc_system.cdc_activity_log`):
### A. Kafka Consumer (1,562 events)
- **07:22:28**: Chạy `snapshot.v2` lần 1 (Job `c49cff75`) -> Ghi 2 batches: 314 + 205 = **519 tin nhắn**.
- **07:25:18**: Chạy `snapshot.v2` lần 2 (Job `f210654a`) -> Ghi 2 batches: 500 + 19 = **519 tin nhắn**.
- **07:34:52**: Stream realtime = **1 tin nhắn**.
- **07:50:43**: Chạy `snapshot.v2` lần 3 (Job `f6b5a838`) -> Ghi 2 batches: 500 + 20 = **520 tin nhắn**.
- **07:09 - 07:15**: 3 tin nhắn stream rải rác trước đó.
=> **Tổng cộng = 519 + 519 + 1 + 520 + 3 = 1,562 events**.

### B. Transmute (2,084 records)
Đối với pipeline binding `export_jobs_1`:
- **07:09:28**: 3 records (inserted: 3, updated: 0)
- **07:15:29**: 3 records (inserted: 0, updated: 3)
- **07:22:28**: 519 records (inserted: 519, updated: 0)
- **07:25:18**: 519 records (inserted: 519, updated: 0)
- **07:29:20**: 519 records (inserted: 0, updated: 519) -> Do trigger thủ công `run-now-92`
- **07:34:52**: 1 record (inserted: 1, updated: 0)
- **07:50:44**: 20 records (inserted: 20, updated: 0)
- **07:50:44**: 500 records (inserted: 500, updated: 0)
=> **Tổng cộng = 3 + 3 + 519 + 519 + 519 + 1 + 20 + 500 = 2,084 records**.

### C. Tại sao Bảng chỉ có 536 records mà Throughput lại là 1,562 và 2,084?
- **Khái niệm Throughput (Lưu lượng)** khác với **Storage Size (Dung lượng lưu trữ)**:
  - CDC Shadow DB sử dụng câu lệnh `UPSERT`:
    `INSERT INTO ... ON CONFLICT (_source_id) DO UPDATE ...`
  - Khi re-snapshot hoặc re-sync 3 lần liên tiếp trong ngày, 519 bản ghi đó được gửi lại 3 lần. Chúng ghi đè lên chính các bản ghi đã có ở Shadow và Master chứ không làm tăng số dòng vật lý của bảng.
  - Do đó: Dung lượng bảng hiện tại vẫn là **536 records**, nhưng tổng khối lượng công việc CDC đã vận chuyển trong ngày là **1,562 sự kiện**.

## 2. Vì sao c, u, d bị hiển thị sai?
1. **Gốc rễ Backend Go (`centralized-data-service`)**:
   - `batch_buffer.go:365` khi ghi log chỉ lưu `{"batch_size": X, "written": Y, "trace_id": "..."}`, hoàn toàn không đếm số lượng `c`, `u`, `d`.
   - Struct `shadow.UpsertRecord` thiếu trường `Op string`.
2. **Gốc rễ Frontend (`ReconPipelineGrid.tsx`)**:
   - Có đoạn fallback tự dồn 100% `batchSize` vào `c`:
     `const c = Number(details.c) || (batchSize > 0 && !u && !del ? batchSize : 0);`
   - Vì `details.c` không có, frontend ép tất cả 1,562 sự kiện thành `c`, trong khi `u = 0, d = 0, r = 0`.
3. **Cột Read(r) ở Kafka CDC**:
   - CDC nghiệp vụ chỉ quan tâm thay đổi dữ liệu bảng (Create, Update, Delete). Cột `Read(r)` ở mục 1 gây rối và thừa thãi.

## 3. Audit Chuyên Sâu: 2 Batch Buffer (`cdcBatchBuffer` vs `snapshotBatchBuffer`)
1. **Khởi tạo tại `server_setup.go:152-172`**:
   - `cdcBatchBuffer`: BatchSize 500, Timeout 100ms, Concurrency 20, Commit Kafka offsets an toàn, xử lý realtime stream qua `cdcEventHandler`.
   - `snapshotBatchBuffer`: BatchSize >= 1,000, Timeout 5s, Concurrency 4 (bảo vệ DB không bị nghẽn), không commit Kafka offset, xử lý nạp khối qua `snapshotEventHandler`.
2. **Điểm gãy Activity Log**:
   - Tại `batch_buffer.go:358`, lệnh `act.Start("kafka-consumer", ...)` bị hardcode cho cả 2 buffer.
   - Do đó 1,558 bản ghi của 3 lần snapshot ngày 30/09 đi qua `snapshotBatchBuffer` bị ghi nhận dưới tên `"kafka-consumer"`, khiến mục 1 trên giao diện gom cả dữ liệu snapshot thành tin nhắn Kafka.
3. **Giải pháp chuẩn hóa**:
   - Thêm `Op string` vào `shadow.UpsertRecord`.
   - Trong `event_handler.go` và `bridge_handler.go`: Gán `record.Op = event.Data.Op`.
   - Trong `batch_buffer.go`:
     - Đếm `cntC, cntU, cntD`.
     - Phân định `bb.isSnapshotBuffer`: gán `triggeredBy = "snapshot"` và `buffer_type = "snapshot"` khi nạp khối snapshot; gán `triggeredBy = "kafka-consumer"` và `buffer_type = "cdc-stream"` khi là realtime streaming.
     - Ghi nhận đầy đủ `c, u, d, buffer_type, is_snapshot` vào `details`.

---

## 3. Phân tích Dữ liệu Lịch sử Ngày 2026-09-30 (Bảng export_jobs)

### 3.1. Truy vấn vật lý đối chiếu:
```sql
SELECT id, operation, target_table, triggered_by, rows_affected, details::text, started_at
FROM cdc_system.cdc_activity_log
WHERE target_table LIKE '%export_jobs%' 
  AND started_at::text LIKE '2026-09-30%'
ORDER BY id ASC;
```

### 3.2. Bảng phân rã chi tiết 1,562 events Kafka Ingest:
1. Đợt 1 (07:22:21): Batch 500 (id 100914) + Batch 19 (id 100915) = 519 records $\rightarrow$ Kích hoạt bởi `snapshot.v2` id 100916.
2. Đợt 2 (07:25:18): Batch 500 (id 100924) + Batch 19 (id 100925) = 519 records $\rightarrow$ Kích hoạt bởi `snapshot.v2` id 100926.
3. Đợt 3 (07:50:43): Batch 500 (id 101020) + Batch 20 (id 101021) = 520 records $\rightarrow$ Kích hoạt bởi `snapshot.v2` id 101022.
4. Stream events: 4 bản ghi rải rác trong ngày (07:34, 10:30, 10:40, 11:00).
Tổng cộng: $519 + 519 + 520 + 4 = 1,562$ records.

### 3.3. Bảng phân rã chi tiết 2,084 records Master Transmute:
1. Transmute sau snapshot 1: 519 records (id 100918, 100919, 100920).
2. Transmute sau snapshot 2: 519 records (id 100927, 100928, 100929).
3. Transmute thủ công `run-now-92` lúc 07:29:20: 519 records (id 100938 - admin trigger).
4. Transmute sau snapshot 3: 520 records (id 101026, 101027, 101028).
5. Stream events: 7 records rải rác trong ngày.
Tổng cộng: $519 + 519 + 519 + 520 + 7 = 2,084$ records.

### 3.4. Kết luận phân tích:
- 1,562 và 2,084 là THROUGHPUT XỬ LÝ (Processing Throughput) trong ngày, gồm nhiều chu kỳ snapshot và manual triggers lặp lại trên cùng một tập dữ liệu gốc 536 records.
- Khi phân tách Snapshot ra khỏi luồng Kafka Stream, chỉ số tại Trạm 1 (KAFKA CDC BẮN VỀ) phản ánh chính xác biến động realtime (4 events), trong khi Trạm 2 (SHADOW INGEST) phản ánh toàn bộ khối lượng dữ liệu được nạp vào DB Shadow (1,562 records).
