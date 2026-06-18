# 02_plan

## Root cause (xác nhận qua code + DB)
- **Stale runs**: `recon_core.go beginRun` (L283-324) chỉ tự-cancel stale >15' khi có run MỚI **cùng table_name** đụng unique `recon_runs_one_running`. → reactive; bảng không còn active không bao giờ trigger → run treo vĩnh viễn. Mỗi restart worker (rebuild p4d→p4g) bỏ lại run 'running' của instance_id cũ.
- **Warn**: `metadata_registry_service.go` RefreshCache (L227-243) loop **tất cả** connections gọi `resolveSourceURIFromConn` → role=master/dest không có source-URI → warn. `default_master` 0 nguồn bind (xác nhận: `SELECT count(*)…=0`).

## FIX 1 — Proactive stale-run reaper (recon_core.go + worker_server.go)
1. Thêm method `ReconCore.ReapStaleRuns(ctx) (int64, error)`: cancel **toàn cục** (mọi table) `status='running' AND started_at < NOW()-interval '15 minutes'` → `status='cancelled', finished_at=NOW(), error_message='stale running reaped (worker restart/hung)'`. Trả RowsAffected, log Warn nếu >0. Tái dùng ngưỡng 15' (đồng nhất beginRun).
2. Gọi `ReapStaleRuns` **một lần lúc startup** trong `WorkerServer.Start()` (sau khi reconCore wired) → dọn ngay các run treo cũ.
3. Gọi `ReapStaleRuns` **đầu mỗi chu kỳ reconcile** (case "reconcile"/runReconcileCycle) → tự lành định kỳ kể cả không restart.

→ Đáp R1.2/1.3 (proactive + vươn tới bảng inactive), R1.4 (ngưỡng 15'), R1.1/1.5 (startup reaper dọn 7 row cũ ngay).

## FIX 2 — Chỉ resolve source-URI cho connection có nguồn (metadata_registry_service.go)
4. Trước loop L227, build `referencedConnIDs := set{ sources[i].<SourceConnectionID> }`. Trong loop: nếu `connections[i].ID ∉ referencedConnIDs` → `continue` (skip im lặng, optional debug-log). Giữ nguyên warn cho connection có nguồn mà resolve fail (R2.3).
   - Cần xác nhận tên field connection-id trên model SourceObjectRegistry khi implement.

→ Đáp R2.1/2.2/2.3/2.4.

## VERIFY (red→green, exercise-driven — G2/G3)
5. `go build ./...` = 0.
6. Build binary worker mới + dừng binary cũ + chạy lại (mirror cách đang chạy: detached + log).
7. R1: `count(running & stale)` 7→0 sau startup reaper; log mới không còn `23505 beginRun failed`.
8. R2: log worker mới KHÔNG còn warn `default_master`; (nếu có connection nguồn lỗi thật thì vẫn warn — chấp nhận).
9. Ghi 03_implementation, 06_validation (lệnh+output PASS), append 05_progress.

## Rollback/safety
- Backup file trước sửa (Rule 18, copy *.bak). Không commit/push.
- Reaper chỉ cancel run >15' → không giết run live. Fix warn chỉ bỏ qua connection không có nguồn → không mất tín hiệu lỗi thật.
