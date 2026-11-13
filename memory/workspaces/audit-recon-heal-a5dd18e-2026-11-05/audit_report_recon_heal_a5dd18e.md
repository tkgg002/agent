---
tier: T2
mode: audit
suffix: recon_heal_a5dd18e
workspace: audit-recon-heal-a5dd18e-2026-11-05
created_at: 2026-11-05T13:06:00+07:00
---
# Audit Report — a5dd18e còn lỗi recon/heal không?

## 0. Kết luận
CÓ. a5dd18e (Merge develop → release/1.1.0, 2026-10-26) chứa đủ các commit fix recon/heal tới `786db91 fix recon/heal` (code ≡ 786db91, chỉ khác Dockerfile + 1 migration). Nhưng mọi fix recon/heal làm SAU 786db91 đều đang nằm ở working tree CHƯA COMMIT trên nhánh `recon-heal` (kể cả `b8326bf` cũng không có) ⇒ a5dd18e vẫn còn các lỗi dưới.

## 1. Ancestry (git, read-only)
- `git merge-base --is-ancestor`: 786db91, 34374a9, 3369c31, 08c05a8, 30b1b00, 0039205, 5c9e367, 98cf347 đều IN a5dd18e.
- a5dd18e không nằm trên branch local nào; HEAD = `b8326bf` (recon-heal) chỉ thêm Mongo master cho recon, không sửa các lỗi dưới.

## 2. Lỗi còn tồn tại tại a5dd18e
| ID | Lỗi | Bằng chứng @a5dd18e | Fix ở đâu |
|---|---|---|---|
| R-01 | Heal dùng chung `snapshotEventHandler`/`snapshotBatchBuffer` với SnapshotRunner → heal flush va chạm/ăn lỗi async của snapshot (F16 pg_oom, C1 duplicate-source-ts) | `server_setup.go:474-480` `WithEventHandler(snapshotEventHandler)` | WT chưa commit: `healBatchBuffer`/`healEventHandler` (`server_setup.go:159,206,461`), `batch_buffer_async_err.go` (untracked) |
| R-02/R-03 | Source PG không có timestamp_field → mặc định `updated_at` và trả `primary, primary, nil` dù cột không tồn tại → recon A/B hash theo cột sai/không có | `recon_tier_a.go:201-205`, `:259-260` | WT: `_id_only` mode (`recon_tier_a.go`, `recon_hash.go`) |
| R-04 | Smoke B tính drift trừ thô `shadowActive - masterActive` → báo lệch giả khi transmute lag; Segment A còn Two-Guard cũ + `smokeCountCache` | `recon_smoke.go:668`, `:354-408`, `:78-100`; `recon_smoke_guard.go`/`recon_window_index_probe.go` không tồn tại | WT: `recon_smoke_guard.go` (untracked) |
| R-05 | Segment B ts col chỉ lấy từ registry, bỏ qua `transform_spec.timestamp_field` | `recon_tier_b.go:532,797` `tsCol := "_source_ts"` khối cũ | WT: `resolveSegmentBTSCol` (`recon_engine_segment_b.go`) |

## 3. DG3
- `06_laya_summary_dg3_20261105T130520.json`: R-01..R-04 `FLAG` → resolve bằng `REF:` verbatim (git show a5dd18e) → gate `PASS`. Controls: CTRL-TRUE-1, CTRL-FALSE-1/2.

## 4. Giới hạn
- Chỉ đối chiếu code (static). Chưa trace runtime prod; chưa biết User đang gặp triệu chứng cụ thể nào → cần User cung cấp log/report_id để map đúng R-xx.
- Fix trong WT chưa commit, chưa qua review/test trong phiên này.

## 5. Six-step QC
- **(a) Line-by-line / git diff:** Không có code change trong phiên. Đã đọc `git diff HEAD` working tree các file recon/heal (`recon_tier_a.go`, `recon_tier_b.go`, `recon_hash.go`, `recon_engine_segment_b.go`, `recon_execute_heal_segb.go`, `server_setup.go`) và `git diff a5dd18e b8326bf` để xác định fix nào chưa có ở a5dd18e. TASK-01..03 trong 08_tasks đều có bằng chứng ở mục 1–3.
- **(b) Anti smoke-test / runtime trace:** Kết luận chỉ dựa trên `git show a5dd18e:<file>` (static). Không chạy runtime prod; không suy diễn hành vi runtime từ unit test.
- **(c) Concurrency / timeout-heartbeat / continuity:** R-01 là vấn đề concurrency (heal và snapshot cùng một batch buffer, flush/async-error dùng chung). R-04 liên quan watermark/lag (smoke B không lùi freeze cutoff). Chưa đo trên multi-worker thật.
- **(d) Suy diễn & trung thực:** Mức độ ảnh hưởng thực tế trên prod = HYPOTHESIS cho tới khi User cung cấp log/report. Fix WT chưa được test trong phiên này.
- **(e) Self-improvement / lessons:** Không bị User sửa lưng; không có lesson mới. Áp dụng L030, L034, L035, L058.
- **(f) DG3 (laya_gate):** xem mục 3 — gate PASS.
