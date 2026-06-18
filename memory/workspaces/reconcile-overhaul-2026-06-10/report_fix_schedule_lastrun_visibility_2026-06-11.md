# report_fix_schedule_lastrun_visibility_2026-06-11.md — "Schedule nói chạy 09:56 mà chưa thấy chạy"

> Muscle:Claude-Opus-4.8 | 2026-06-11

## 1. Verify TRƯỚC (schedule có chết không?) — KHÔNG
- `recon_runs` có run mới `02:58:57Z` (= **09:58 +07**, đúng hẹn 09:56 + tick 60s); leader key đang heartbeat; log `executing scheduled operation reconcile`. → **Schedule đã kick đúng hẹn và đang chạy.**

## 2. Root cause cảm nhận sai (UI)
`worker_server.go` update `last_run_at/next_run_at/run_count` **SAU khi op chạy xong** — vòng `reconcile` (CheckAll stagger) kéo 5-10' → suốt thời gian đó UI đứng "09:26" → operator tưởng schedule chết. (Đúng lesson visibility-not-prevention.)

## 3. Fix
Di chuyển update block lên **ĐẦU vòng** (mark-at-start). Bonus đúng đắn: cadence chuẩn `next = start + interval` — không drift theo độ dài vòng như `next = end + interval` cũ.

## 4. Files đã sửa
| File | Đổi |
|---|---|
| `centralized-data-service/internal/server/worker_server.go` | di chuyển update schedule lên trước switch op (+comment, net ~0 LOC logic mới) |

## 5. Verify (bằng chứng thật)
- Build PASS; worker restart p4f (PID 85218).
- Ép due (`last_run −31'`) → tick kế: **`last_run_at=03:11:30Z` trong khi `NOW=03:11:34Z`** — update 4s sau khi vòng BẮT ĐẦU; cùng lúc leader key EXISTS=1 + log `reconcile cycle started` → vòng vẫn đang chạy mà UI đã nhảy. Hành vi mới chứng minh xong.
- run_count 8→9 tại start.

## 6. Note
- Working tree có thêm code orphan-prune + `usePruneMutation` (bên khác phát triển song song) — build chung PASS, không đụng.
