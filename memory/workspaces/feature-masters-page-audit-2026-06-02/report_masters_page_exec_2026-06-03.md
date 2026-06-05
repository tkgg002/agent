# report_masters_page_exec_2026-06-03.md

> **Workspace**: `feature-masters-page-audit-2026-06-02`
> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-03
> **Verb**: `execute` (cả 3 phase) — sau khi User báo plan chưa làm.

## Tóm tắt
Execute đủ 3 phase của `02_plan.md`, áp dụng các sửa từ workflow verify (P1 bug `.find`; P2 không đụng Config vì DB đã wired).

## Files đã thay đổi
| File | Phase | ≈ thêm | Thay đổi |
|------|-------|--------|----------|
| `cdc-cms-web/src/pages/MasterRegistry.tsx` | 1 | ≈ +100 | import Radio/Tooltip/SyncOutlined; state `syncRow`/`syncForm`; mutation `syncMut` (3 mode); cột "Sync"; Sync Modal. **FIX**: `.find(immediate)` → lọc thêm `&& s.master_table===row.master_name` (tránh run-now nhầm master). |
| `cdc-cms-web/src/pages/TransmuteSchedules.tsx` | 3 | ≈ +12 | import Tooltip/InfoCircleOutlined; wrap option `post_ingest` bằng Tooltip giải thích realtime. |
| `centralized-data-service/internal/sinkworker/sinkworker.go` | 2 | ≈ +55 | struct +`piCacheMu`/`postIngestCache` + type `piCacheEntry`; New() init cache; **gate `hasPostIngestSchedule`** (cache 30s, query parameterized) + guard trong `publishTransmuteTrigger`. |

## Khác biệt so với plan gốc (đúng theo verify)
- **P2 KHÔNG thêm `DB *gorm.DB` vào Config / KHÔNG sửa `worker_server.go`** — đã wired sẵn (`Config.DB:54`, `SinkWorker.db:31`, `New:78`, construct ở `cmd/sinkworker/main.go`). Plan gốc sai điểm này → bỏ.
- **P2 fail-OPEN** (nil DB / query error → publish) thay vì plan "fail-safe return false" — vì realtime là LUỒNG CHÍNH, không được ngắt im lặng khi lỗi tạm thời.
- **P2 chỉ đụng `publishTransmuteTrigger`** (trigger shadow→master), KHÔNG đụng `upsertWithFencing` (ghi shadow / db→shadow) — tôn trọng "ko sửa luồng db→shadow".
- **P1 fix bug** `.find` lọc theo master_table (verify phát hiện).
- Hợp đồng API verify khớp: POST `/api/v1/schedules` body `{master_table,mode,cron_expr,is_enabled,reason}`, mode∈{cron,immediate,post_ingest}, reason≥10 (modal validate), `/run-now` không đọc body.

## Verify (exit code THỰC TẾ)
- `go build ./internal/... ./cmd/worker ./cmd/sinkworker` → **EXIT 0**
- `go vet ./internal/sinkworker/` → sạch (chỉ cảnh báo `pkgs/idgen` pre-existing)
- `go test ./internal/service/ ./internal/service/transmute/ ./internal/handler/` → **PASS (no regression)**
- `cdc-cms-web`: `npx tsc -b` → **EXIT 0**; `npm run build` → **✓ built (462ms)**

## Hành vi thay đổi cần lưu ý
- **Gate post_ingest**: sau thay đổi, fan-out realtime CHỈ chạy cho shadow table có post_ingest schedule bật. Master chỉ dùng cron/run-now sẽ KHÔNG còn trigger per-write (giảm NATS noise) — đúng design G-5 + ăn khớp Sync Modal "Realtime" (tạo post_ingest schedule). Master không bật realtime vẫn sync qua cron/run-now.

## Manual test (chưa chạy — cần môi trường live, ghi rõ KHÔNG báo láo)
Theo `02_plan.md` Verification Plan: /masters → approve+active → nút Sync hiện → chọn 3 mode → check worker log. Cần CMS+worker+NATS live; build/test tĩnh đã pass, manual browser test để User/QA xác nhận.

## Gap còn lại (audit cũ, chưa làm — ngoài 3 phase)
- G-7 (UTC hint cột Reviewed) LOW: chưa làm.
- G-4 confirmed đúng design (scheduler chỉ cron) — không cần sửa.
- G-8 (approve_master column) verify: `master_table` tồn tại — không cần sửa.
