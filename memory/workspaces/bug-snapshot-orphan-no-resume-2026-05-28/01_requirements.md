# 01_requirements — Bug Snapshot Orphan No-Resume

## Mục tiêu
Cho phép snapshot tự **tiếp tục** sau khi worker crash/kill mà không cần human intervention; đồng thời cung cấp **lối thoát UI** khi cần force-resume thủ công.

## Functional Requirements

| ID | Yêu cầu | Acceptance |
|---|---|---|
| FR-1 | Worker `centralized-data-service` khi boot phải scan `cdc_system.snapshot_progress` tìm row `status='running'` với `updated_at < NOW() - staleAfter` → publish lại `cdc.cmd.snapshot.v2` cho từng row. | Log line `snapshot.v2 boot-reclaim N orphan rows` xuất hiện trong startup logs khi tồn tại orphan. |
| FR-2 | Trước khi publish, worker UPDATE row → `status='paused'` để `claimProgress` đi nhánh resume hợp lệ (line 637-655) thay vì bị block bởi zombie window. | DB row chuyển running→paused trong < 1s sau worker boot, sau đó claim chuyển sang running. |
| FR-3 | `staleAfter` mặc định 60s (đủ buffer cho batch lớn checkpoint mỗi 5s). Configurable qua env `SNAPSHOT_STALE_AFTER_SECONDS`. | `time.Since(updated_at) > 60s` → reclaim; < 60s → bỏ qua (worker khác có thể đang chạy). |
| FR-4 | FE `/snapshot-monitor` hiển thị nút **Resume** cho row `status='running'` AND `Date.now() - updated_at > 60s` (UI gọi đây là "stale"). Nút có icon cảnh báo + tooltip "Snapshot có thể đã orphan (worker chưa heartbeat 60s+)". | Inspect DOM: row có cả `status=running` + updated_at cũ 90s → thấy nút Resume. |
| FR-5 | Click "Resume" cho row stale gọi cùng endpoint `POST /api/v1/snapshot-progress/:source_object_id/resume`. Backend handler không cần đổi. | Network tab thấy POST resume thành công 200/204. |
| FR-6 | Confirm dialog Resume cho stale running hiển thị warning "Lưu ý: nếu worker thật sự đang chạy, có thể trigger double-process — chờ thêm 60s nếu không chắc." | Modal description chứa text cảnh báo. |

## Non-Functional Requirements

| ID | Yêu cầu |
|---|---|
| NFR-1 | Boot reclaim KHÔNG block worker startup nếu DB lỗi; log warn + skip, snapshot vẫn nhận message mới. |
| NFR-2 | Boot reclaim chỉ chạy 1 lần ở startup (không cron). Periodic check defer roadmap. |
| NFR-3 | Patch tối thiểu §6. Không thêm dependency. Không migration mới. |
| NFR-4 | Idempotent: nếu boot reclaim chạy 2 lần cùng row (vd worker double-instance restart), claim transaction lock phải đảm bảo chỉ 1 worker pick up. |

## Definition of Done

| DoD | Mô tả | Verify |
|---|---|---|
| DoD-1 | Worker boot reclaim function exist + integration test | `go test -run TestBootReclaim` PASS |
| DoD-2 | Worker server wire reclaim job vào startup | grep `BootReclaim\|ReclaimOrphans` ở `worker_server.go` |
| DoD-3 | FE render Resume cho stale running | Static scan: `SnapshotMonitor.tsx` chứa `isStale(updated_at)` helper |
| DoD-4 | FE confirm dialog có warning cho stale resume | grep "orphan\|stale\|chưa heartbeat" trong modal description |
| DoD-5 | `staleAfter` configurable | grep `SNAPSHOT_STALE_AFTER_SECONDS` env binding |
| DoD-6 | Build + test PASS (BE + FE) | `go build && go test && npx vite build` exit 0 |
| DoD-7 | Report có files thay đổi + LOC delta | bảng trong `report_*.md` |
| DoD-8 | Governance §1+§11+§12+§14 | Brain plan-only; APPEND-only; Pre-flight 12 file |

## Out-of-scope
- Đổi NATS subscription sang JetStream durable (refactor lớn).
- Periodic background reclaim (cron). Defer.
- Liveness probe / health check infrastructure.
- Auto-retry cho row `status='error'`.

## Constraints
- ✓ Đọc lesson trước.
- ✓ §1+§12 Brain plan-only.
- ✓ Plan + code demo.
- ✓ KHÔNG cheat DB.
- ✓ Report files + LOC.
- ✓ Verify trước báo done.
