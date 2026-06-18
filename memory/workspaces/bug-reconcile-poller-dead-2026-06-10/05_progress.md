# 05_progress — reconcile poller dead (APPEND-ONLY)

## [2026-06-10] Diagnose + fix (recover trong poller)
- Loại trừ thật: code path/reconCore/redis/lock đều OK → root = poller goroutine chết do thiếu recover() (1 panic ~06-02 giết im lặng; last_run kẹt, run_count=3).
- Fix: thêm `defer recover()` per-operation quanh switch trong worker_server.go poller + log zap.Stack (schedule_exec_panic).
- Verify: gofmt OK, `go build ./...` PASS, recover() xác nhận có ở dòng 938.
- CHƯA take effect: cần rebuild+restart worker (binary đang chạy là cũ). Sau restart: nếu còn panic → stack lộ ra mỗi phút → fix root reconcile; nếu không → reconcile auto chạy lại.

## [2026-06-11] ĐÍNH CHÍNH root cause sau khi quan sát worker có-fix
- recover() đã deploy (strings xác nhận hasFix=2 trong binary :8082 pid CDS worker) NHƯNG reconcile vẫn không auto-chạy đều → root KHÁC panic.
- Bằng chứng: recon_run `export_jobs` tier1 IN-PROGRESS 6+ phút, docs_scanned=0, pg_stat_activity trống → HANG ở source DB. CheckAll (recon_core.go:988) cố ý `time.Sleep(spread/len)` rải ~5 phút + jitter, gọi `RunTier1` với context.Background KHÔNG timeout per-table.
- ROOT: (1) CheckAll mất ≥5 phút/lần (stagger) nhưng poller gọi đồng bộ → block poller ≥5 phút; interval_minutes=1 bất khả thi. (2) 1 bảng hang (export_jobs, source unreachable, không timeout) → CheckAll kẹt hàng giờ → poller đóng băng → reconcile "ko chạy" (cadence ~6h). (3) phụ: 25 report error `column "_id" does not exist` (orphan_prune sai PK).
- FIX đề xuất: #1 timeout per-table quanh RunTier1 (chính); #2 interval thực tế ≥ stagger HOẶC reconcile chạy goroutine riêng + overlap-guard; #3 fix source export_jobs hang; #4 fix SQL _id. Chờ user chốt timeout/interval.

## [2026-06-11] Implement Fix #1 + #2 (user chọn c)
- Fix #1: recon_core.go CheckAll — bọc RunTier1 bằng `context.WithTimeout(ctx, 45s)` per-table (ctx được truyền xuống sourceAgent.Count* → Mongo/PG tôn trọng deadline → cắt hang). 1 bảng hang (export_jobs) không còn block cả CheckAll/poller.
- Fix #2: UPDATE cdc_worker_schedule interval_minutes 1→10 (stagger CheckAll ~5m nên interval 1m bất khả thi).
- Verify: gofmt OK, go build ./... PASS.
- CHƯA take effect: cần rebuild+restart worker. Sau restart: CheckAll sẽ hoàn tất (export_jobs timeout 45s → mark error thay vì treo), poller tick lại, reconcile auto mỗi ~10m.
- (b) export_jobs hang: source unreachable (sẽ lộ source URL trong log timeout sau restart). Phụ: 25 report error `column "_id" does not exist` (orphan_prune) — chưa fix.

## [2026-06-11] (b) Root export_jobs hang + LESSON lộ secret
- export_jobs source = MongoDB private-subnet remote (10.200.18x.x:27017, replicaSet goopay) KHÔNG reachable từ máy dev localhost → connect treo vô hạn → CheckAll hang → poller freeze. Fix #1 timeout 45s giờ chặn (mark error sau 45s).
- LESSON (Rule 19): connection_registry lưu FULL URI có credentials trong cột `host`/url → `SELECT host` lộ password ra output/transcript. Từ nay điều tra connection chỉ SELECT connection_code/engine_type/status; KHÔNG select host/url/secret_ref (hoặc mask). Global Pattern: [đọc registry/config table chứa secret] bằng SELECT cột thô → lộ credential. Đúng: chỉ select cột non-secret hoặc mask trong query.
