# report_fix_chain_counts_lag_window_2026-06-11.md — Chuỗi fix theo 3 báo cáo của Boss

> Muscle:Claude-Opus-4.8 | 2026-06-11 | Boss: (1) b3 thiếu 1 + "lag 17.4h là gì"; (2) export_jobs Source=0 → ingest +170 giả; (3) mt_02 master 332 "không đúng"

## 1. Trả lời + root cause từng câu (verify bằng số thật)
| Câu hỏi | Trả lời |
|---|---|
| **17.4h là gì** | `transmute_lag = max(_source_ts)shadow − max(_source_ts)master` = 02:29Z(10/6) − 09:08Z(9/6) = 17h21'. Row mới nhất shadow (`_gpay_id=58019585403650053`) chưa từng sang master vì **b3 Sync=Manual** (Boss xác nhận đã hiểu) |
| **Source=0 / ingest +170 giả** | A-row latest = error 23505 `recon_runs_one_running`: **6 run 'running' mồ côi** (worker kill giữa vòng) chặn recon các bảng VĨNH VIỄN → totals NULL → FE "0". Source THẬT = **168** (đếm qua remote cluster dev000 — 10.200.187.x, db `centrallized-…` 3L là tên thật bên remote) → đúng phải là **ingest +2 (orphan thật)** |
| **mt_02 master 332** | Đếm thật hiện tại = **170**. 332 là số **ĐÚNG tại thời điểm đo 10:56Z** — sau đó orphan-prune (đang dev song song) dọn 162 row; report stale vì **Segment B chưa nằm trong scheduler** |

## 2. Fixes (5 fix code + 1 data-fix)
| # | Fix | File |
|---|---|---|
| 1 | **Window edge +1ms**: upper EXCLUSIVE = max-ts → row mới nhất vĩnh viễn ngoài window khi idle (B báo ok dù totals lệch) — fix cả Segment B lẫn A | `recon_core.go` ×2 |
| 2 | **beginRun tự hồi phục stale-running**: gặp 23505 → auto-cancel run 'running' >15' + retry (worker crash/restart không còn làm kẹt bảng vĩnh viễn) | `recon_core.go` |
| 3 | Data-fix: cancel 6 orphan running | recon_runs |
| 4 | **Segment B vào scheduler**: `runReconcileCycle` += `CheckAllSegmentB` — số B tươi mỗi 30' (lưu ý: leader `release()` đã DEL key ownership-guarded nên B chạy ngay sau A trong cùng cycle OK) | `worker_server.go` |
| 5 | **EnsureMaster index cột ma**: Apply tạo UNIQUE INDEX theo `spec.pk` (flatten default `_source_id`) — cột đã bỏ khỏi master → mọi transmute/heal của bảng chết 42703. Guard `realCols` (≠ `seen` vốn reserve cả tên-đã-bỏ; file vừa được bên khác thêm cột `_id`) | `master_ddl_generator.go` |

## 3. Verify E2E
- Sau fix #1: re-check B b3 → window bắt **đích danh** `missing=["58019585403650053"]` ✅ (trước đó 11=11 "ok" giả).
- Sau fix #2+#3: re-check A export_jobs chạy lại được → totals **168/170** ✅.
- Sau fix #4: re-check mt_02 → **ok 170/170** ✅ (số tươi).
- Sau fix #5: heal b3 hết 42703; transmute giờ **từ chối ghi đúng gate**: `scanned=1 skipped=1 type_errors=1` — row test này có field không pass type-validation trong flatten transform (data-quality của row, KHÔNG phải bug recon; b3 là bảng Manual test — deep-dive riêng nếu Boss cần).
- Build PASS toàn bộ; worker p4j RUNNING.

## 4. Note
- Working tree song song của bên khác: master DDL contract mới (+cột `_id`), orphan-prune, connectionOverrides — build chung PASS, tôi chỉ đụng phần recon.
- b3 row cuối: muốn ghi được cần sửa data/rule của row đó hoặc chấp nhận (bảng test).
