# 09 — Tasks Solution v2 (PROPOSE ONLY — chờ `APPROVE`)

> Brain không sửa source. Muscle thực thi sau APPROVE: **RED test trước**, rồi fix, rồi `/security-agent`.
> Thay thế đề xuất cũ cho các điểm đã đổi (F3c thu hẹp, F7 hạ cấp). Không xoá file `09_tasks_solution_miss_record_audit.md`.

## Hướng giải quyết duy nhất: "Commit offset chỉ tới điểm đã persist (hoặc đã DLQ thành công)"

| Task | Nội dung | Vị trí |
|---|---|---|
| T1 | `batchUpsert` trả **kết quả theo từng record** (`persisted | dlq'd | failed`). Retryable error KHÔNG `return` bỏ phần còn lại: đánh dấu phần còn lại là `failed`. | `batch_buffer.go:297-538` |
| T2 | `FlushWithContext`: với mỗi TopicPartition commit `min(maxOffset, minFailedOffset-1)`; record `failed` được requeue (retry giới hạn, backoff), quá ngưỡng → DLQ `failed_sync_logs` có kafka coords; DLQ lỗi ⇒ **giữ offset**. | `batch_buffer.go:204-295` |
| T3 | Serialize flush: `flushMu` bao toàn bộ swap → write → commit để thứ tự commit = thứ tự swap. | `batch_buffer.go:204-295` |
| T4 | `isRetryableDBError`: thêm SQLSTATE class `53` (53300 too many clients, 53200, 53400) và `57014`. | `batch_buffer.go:596-628` |
| T5 | Drop `missing_pk` và `nil_after_data` (non-delete) → ghi DLQ thay vì `return 0,nil` (giữ `empty_value` tombstone = drop có metric). | `event_handler.go:343-350`, `kafka_consumer.go:631-640` |
| T6 | `resolveDB`: role `shadow` với key rỗng hoặc `GetShadowDB` lỗi ⇒ trả lỗi (record `failed`), không rơi về `bb.db`. | `batch_buffer_utils.go:41-58` |
| T7 | `kc.batches`: thêm mutex `batchMu` HOẶC NATS callback chỉ set cờ atomic và loop tự gọi `RefreshTopics`. | `adaptive_batcher.go`, `server_setup.go:506-508` |
| T8 | Metric/alert: `PersistedVsCommittedGap`, cảnh báo khi `DLQWriteFail>0`. | `pkgs/metrics` |

## Definition of Done (G2/G3/G6)
1. **RED (integration thật, PG thật — không mock DB):** harness dựng PG với `max_connections` thấp và/hoặc `pg_terminate_backend` giữa flush; đẩy N record qua `BatchBuffer` thật với offset giả K partition. Trước fix: `count(shadow) < N` mà offset đã commit ≥ N ⇒ FAIL.
2. **GREEN:** sau fix `count(source)=count(shadow)` (sau replay/retry) và offset commit ≤ record persisted liên tục.
3. Test P5 (2 flush đồng thời, flush đầu thất bại) và P7 (missing PK → có dòng `failed_sync_logs`).
4. Chạy lại toàn bộ suite hiện có (G5); `/security-agent` review diff; ghi bằng chứng vào `06_validation_*` + `05_progress_*`.

## Điều kiện chặn
- Cần user xác nhận: (a) `cmd/worker` hay `cmd/sinkworker` chạy prod; (b) số instance; (c) kết quả checklist §5 của `13_analysis_miss_record_audit_v2.md`; (d) `APPROVE`.
