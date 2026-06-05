# 10_gap_analysis — Audit Snapshot Zero Records

## GAP-1: Silent-swallow xuất hiện ở nhiều layer khác
- Search pattern `if err := .*; err != nil { .*Error(.*); .*}` toàn repo → tìm nơi error log rồi drop.
- Action: defer thành lint rule hoặc audit riêng.

## GAP-2: BatchBuffer timer loop vẫn silent
- Sau SOL-5, timer loop dùng `_, _ = bb.Flush()` — best-effort, vẫn drop.
- Đúng theo invariant: timer flush là background, không có request đang chờ response.
- Nhưng nếu cluster đang stress + timer Flush fail liên tục → log spam mà không alert.
- Action future: bổ sung metric `BatchFlushErrors` + alert khi rate > threshold.

## GAP-3: `failed_sync_logs` viết im lặng
- Fallback path line 277-302 ghi row vào `failed_sync_logs` + bump `SyncFailed` metric, nhưng KHÔNG return cho caller.
- Plan A đã giảm bớt: nếu fallback persist được = 0 → escalate err. Nếu persist được >0 row trong N → vẫn return không err.
- Action future: thêm threshold "nếu chunk fail > X%" → return err (giống `snapshotV2MaxBatchErrorRatio`).

## GAP-4: Counter `snapshot_progress.rows_processed` có thể double-count
- Nếu resume snapshot từ checkpoint, counter cộng tiếp từ `rowsSoFar`. SOL-4 không thay đổi logic checkpoint resume.
- Action future: verify resume path consume `persisted` đúng.

## GAP-5: `markProgressDone` không verify hậu nghiệm
- Sau Plan A, status=success chỉ khi mọi flush nil err. Nhưng vẫn không có "verify SQL count = expected".
- Action future: optional post-snapshot probe `SELECT count(*) FROM shadow.table WHERE _synced_at >= snapshot_start`.

## GAP-6: Bug Source thực sự bị giấu — cần observability hơn
- Sau Plan A, error sẽ bubble lên `markProgressError`. Nhưng error format `"batch upsert failed: %w"` có thể không đủ context cho operator (SQL nào fail? Column nào constraint?).
- Action future: enrich error với SQLSTATE, table, PK của row failed đầu tiên.

## GAP-7: Workspace `bug-first-snapshot-no-write-2026-05-26` đã sửa lớp 1, hôm nay sửa lớp 2 — vẫn có thể có lớp 3
- Sau Plan A, snapshot path đáng tin hơn. Nhưng kafka consumer path (timer loop) vẫn dùng async BatchBuffer.
- Pattern "counter ≠ persist" có thể tái xuất hiện ở route khác (debezium real-time CDC).
- Action future: extend persistence-accurate counter sang kafka consumer metrics.
