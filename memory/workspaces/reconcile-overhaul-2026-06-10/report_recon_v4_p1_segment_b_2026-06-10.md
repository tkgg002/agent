# report_recon_v4_p1_segment_b_2026-06-10.md — Recon V4 Phase 1: Segment B (shadow↔master)

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-10 | Verb: `approve recon v4` → thực thi P1 theo `09_tasks_solution_recon_v4.md`

## 1. Đã làm gì
Xây **Segment B** — tầng đối soát transmute path (shadow 5436 ↔ master 5434) theo `(_gpay_id, _source_ts)`:
- **Tái dùng tối đa** (Simplicity First): `HashWindow`/`ListIDsInWindow`/`MaxWindowTs`/`diffIDs`/`buildWindows`/lock/run-state/report-store có sẵn — KHÔNG viết agent mới, chỉ tạo instance `ReconDestAgent` thứ 2 trỏ master DB (`RoleDestination`).
- `RunSegmentB`: per 15-min window — count+XOR fingerprint 2 phía → lệch thì drill-down diff đích danh `_gpay_id` (cap 50k chống OOM); window id-set bằng nhưng XOR lệch = stale (ts lệch); window scan lỗi ĐƯỢC ĐẾM và đẩy status `error` (không nuốt thành "khớp").
- `CheckAllSegmentB` (leader-gated) + `RunSegmentBFor` (per master table); run-state ghi `recon_runs` tier=4 với tên FQN master (không đụng unique-running của Segment A).
- NATS: `cdc.cmd.recon-check` thêm field `segment:"shadow_master"`; 0-checked → `warning` (không success giả).
- Migration `081`: cột `segment` (default `source_shadow`) + index đọc latest per segment.

## 2. Files THỰC TẾ đã sửa (git diff)
### centralized-data-service — diff lũy kế nhánh recon: **8 files +376/−28**; riêng phần P1:
| File | P1 thay đổi |
|---|---|
| `internal/service/recon_core.go` | +~233 dòng (block Segment B: consts, MasterBindingRef, listActiveMasterBindings, RunSegmentB, RunSegmentBFor, CheckAllSegmentB, field masterAgent) |
| `internal/handler/recon_handler.go` | +~40 (payload `segment` + `handleReconCheckSegmentB`) |
| `internal/server/worker_server.go` | +10 (wire masterDB → `SetMasterAgent`, warn+fix_hint khi thiếu) |
| `internal/model/reconciliation_report.go` | +4 (field `Segment`) |
*(các dòng còn lại trong stat thuộc phase hồi sinh Segment A đã report trước — `report_reconcile_overhaul_phase1`)*

### cdc-cms-service — 1 file MỚI
| File | Nội dung |
|---|---|
| `migrations/schema/recon_dlq/081_recon_segment_b.sql` | ALTER report +`segment` + index (idempotent IF NOT EXISTS) |

## 3. Verify (bằng chứng thật)
| Bước | Kết quả |
|------|---------|
| `go build ./...` / `go test service+handler+transmute` | ✅ PASS (vet warn `pkgs/idgen/sonyflake` = pre-existing, ngoài diff) |
| Migration 081 apply (5433) | ✅ ALTER+CREATE INDEX; cột `segment` tồn tại |
| Worker restart binary P1 (PID 30499) | ✅ masterAgent wired (không có warn disable) |
| **E2E** `{"segment":"shadow_master","table":"*"}` | ✅ 6/6 bindings checked, 672 windows/binding |
| **Drift transmute THẬT bắt được** | `b3`: shadow 11 vs master 4, missing 8; `export_jobs_mt`: missing 2 + orphan 161; `export_jobs_mt_02`: orphan 162; `aaa`/`aaaa2`/`wallet_capsets`: ok |
| **Đích danh ID + cross-check tay** | `missing_ids=["57409138828771376","58019585579810876"]`; ID đầu: shadow_has=1, master_has=0 ✅ |

→ DoD P1 ("lệch shadow↔master → bắt được, report segment='shadow_master' đích danh ID") đạt — bằng **drift thật**, mạnh hơn fault-injection.

## 4. Phát hiện vận hành từ kết quả thật (cho Boss)
- `b3` thiếu 8 row, `export_jobs_mt` thiếu 2 row phía master → transmute đã từng rớt row — P2 (heal re-trigger qua `cdc.cmd.transmute SourceIDs`) sẽ tự phục hồi đúng các ID này.
- `export_jobs_mt`/`_mt_02` có **orphan lớn ở master** (161/162 row master có mà shadow không còn trong window) — dấu hiệu shadow từng được re-snapshot/reset sau khi master đã materialise; P2 sẽ xử lý orphan theo chính sách soft-delete có ngưỡng (KHÔNG tự xoá hàng loạt — vượt ngưỡng 5000/5% chỉ alert).

## 5. Services sau task
Worker PID 30499 (`/tmp/cdc-worker-recon-p1`) RUNNING 8082; cms 8083 RUNNING (migration 081 đã apply thẳng DB — file nằm trong embed để các môi trường khác tự apply khi boot); FE không đổi.

## 6. Next
`P2 — Self-healing re-trigger` (theo roadmap đã approve; bắt đầu khi Boss ra lệnh hoặc tiếp tục ngay nếu Boss muốn chạy liền mạch).
