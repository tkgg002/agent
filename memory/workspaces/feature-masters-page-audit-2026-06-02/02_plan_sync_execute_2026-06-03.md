# 02_plan_sync_execute_2026-06-03.md — Khôi phục Sync Modal + Activity Log (Execute)

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-03
> **Nguồn**: User duyệt "làm trọn gói" sau `10_gap_analysis.md`.
> **Mục tiêu nghiệm thu**: Trên `/masters`, chọn Sync Modal + nhấn chạy → **worker chạy thật** (bắn NATS `cdc.cmd.transmute` theo mode đã chọn) + **CÓ ghi activity log**.
> **Nguyên tắc**: Simplicity First, minimal impact, đúng pattern source, core-systems, KHÔNG cheat DB/config.

---

## 1. Cơ chế đã verify (nền tảng — không cần dựng lại)
- `POST /api/v1/schedules` (upsert, mode∈cron/immediate/post_ingest, reason≥10) — `transmute_schedule_handler.go:77`.
- `POST /api/v1/schedules/:id/run-now` → `bus.Dispatch(TransmuteRunCommand{MasterTable})` → NATS `cdc.cmd.transmute` → `HandleTransmute` → `TransmuterModule.Run` (đọc `mapping_rule_master`, map Shadow→Master, ghi master DB). Subscriber **vô điều kiện** (`worker_server.go:412`), không black-hole. → **Chạy thật OK.**
- Cron: `transmute_scheduler.go` publish CÙNG subject `cdc.cmd.transmute` (kèm `schedule_id`).
- post_ingest: `sinkworker.publishTransmuteTrigger` (gate `hasPostIngestSchedule`) → `cdc.cmd.transmute-shadow` → `HandleTransmuteShadow` fan-out → `cdc.cmd.transmute`.
- **GAP**: pipeline transmute KHÔNG ghi `cdc_activity_log` (chỉ `sync_runtime_state`). `ActivityLogger` đã có (`service/activity_logger.go`) nhưng chưa inject vào `TransmuteHandler`.

---

## 2. Thay đổi cụ thể (per file)

### A. WORKER `centralized-data-service` — Activity Log cho transmute (BẮT BUỘC)
**A1. `internal/handler/transmute_handler.go`**
- Import thêm `centralized-data-service/internal/model`.
- Struct `TransmuteHandler` thêm field `activity *service.ActivityLogger`.
- `NewTransmuteHandler(...)` thêm param `activity`.
- Trong `HandleTransmute`, bọc `h.svc.Run` bằng lifecycle:
```go
var logEntry *model.ActivityLog
if h.activity != nil {
    logEntry = h.activity.Start("transmute", req.MasterTable, req.TriggeredBy) // status=running
}
res, err := h.svc.Run(ctx, req.MasterTable, req.SourceIDs)
...
if err != nil {
    if logEntry != nil { h.activity.Fail(logEntry, err.Error()) }
} else if logEntry != nil {
    h.activity.Complete(logEntry, res.Inserted+res.Updated, map[string]interface{}{
        "scanned": res.Scanned, "inserted": res.Inserted, "updated": res.Updated,
        "skipped": res.Skipped, "rule_misses": res.RuleMisses, "type_errors": res.TypeErrors,
        "active_gate": res.ActiveGate, "duration_ms": res.DurationMs, "correlation_id": req.CorrelationID,
    })
}
```
→ Mỗi run (run-now=actor / cron=scheduler / post_ingest=sinkworker-hook) sinh 1 row `cdc_activity_log` operation=`transmute`, target=master_table, status running→success/failed, rows_affected=Inserted+Updated.

**A2. `internal/server/worker_server.go`**
- Khai báo `activityLogger := service.NewActivityLogger(db, logger)` TRƯỚC dòng 411 (transmuteHandler).
- `NewTransmuteHandler(transmuter, db, natsClient.Conn, logger, activityLogger)`.
- XOÁ dòng `activityLogger := ...` trùng ở ~668 (tái dùng var đã khai báo) — tránh `no new variables`.

### B. CMS `cdc-cms-service` — robustness (M1, M4)
**B1. `internal/app/commands/create_master.go:213`** — check error vòng lặp clone (đang nuốt lỗi):
```go
if execErr := h.db.WithContext(ctx).Exec(`INSERT ... ON CONFLICT DO NOTHING`, ...).Error; execErr != nil {
    h.logger.Error("clone master rule failed", zap.Error(execErr),
        zap.Int64("master_binding_id", masterBindingID), zap.String("target_column", r.TargetColumn))
}
```
(fail-soft: log rõ, không chặn tạo master; nhưng KHÔNG còn im lặng.)

**B2. `internal/api/master_mapping_rule_handler.go:228`** — guard shadow_schema rỗng trong Flatten:
```go
if shadow.ShadowTable == "" || shadow.ShadowSchema == "" {
    return c.Status(404).JSON(fiber.Map{"error": "shadow binding/schema not found"})
}
```

### C. FRONTEND `cdc-cms-web` — Sync Modal (H1), flatten (H2), edit rule (M3), tooltip (L4)
**C1. `src/pages/MasterRegistry.tsx`** (H1+H2):
- Import thêm `Radio, Tooltip` (antd) + `SyncOutlined, InfoCircleOutlined` (icons).
- `TRANSFORM_TYPES` thêm `'flatten'`; khi chọn flatten → prefill `spec` `{"pk":"_source_id","explode_path":"items[*]"}` + hint.
- State `syncRow`, `syncForm{mode,cron_expr,reason}`; mutation `syncMut`:
  - `run_now`: POST /schedules {mode:'immediate'} → GET /schedules → `find(s.mode==='immediate' && s.master_table===row.master_name)` → POST /schedules/:id/run-now.
  - `cron`: POST /schedules {mode:'cron', cron_expr}.
  - `post_ingest`: POST /schedules {mode:'post_ingest'}.
- Cột "Sync" (button SyncOutlined, disabled nếu `schema_status!=='approved'`) + Sync Modal (Radio 3 mode + Tooltip giải thích + cron input khi cron + reason≥10 + Alert "bắn worker NATS, ghi Activity Log; master phải Active để thực thi").

**C2. `src/pages/MasterMappingFieldsPage.tsx`** (M3 edit rule):
- Cột Actions thêm nút "Sửa" → mở modal với `modalForm` prefill (set `id>0`); submit POST upsert (đã có, `Save` upsert theo `(master_binding_id,target_column)`).

**C3. `src/pages/TransmuteSchedules.tsx`** (L4 tooltip):
- Import `Tooltip` + `InfoCircleOutlined`; option `post_ingest` label kèm icon Tooltip giải thích realtime.

### D. Infra
**D1. `git init` `data-hub`** + `.gitignore` (node_modules, dist, vendor, *.zip, *.log, .env, config-local*) → commit snapshot để FE không bị mất âm thầm lần nữa. KHÔNG push (GEMINI §8).

---

## 3. Quyết định Simplicity (ghi rõ để khỏi hiểu lầm "bỏ sót")
- **M2 explode_path: KHÔNG thêm.** Backend `discoverJsonPaths/extractPaths` (`master_mapping_rule_handler.go:307-342`) **đã tự bóc nested + array `[*]`** từ `source_field`. `source_field` chính là cột JSON cần explode. Thêm input explode_path = thừa + dễ lệch. Auto-discovery đơn giản và đúng hơn plan gốc.

---

## 4. Verification Plan (phải chứng minh, không báo láo)
1. Build: `cd centralized-data-service && go build ./...` (EXIT 0); `cd cdc-cms-service && go build ./...` (EXIT 0); `cd cdc-cms-web && npx tsc -b && npm run build` (EXIT 0).
2. `go vet ./internal/handler/ ./internal/server/` sạch.
3. End-to-end audit (sub-agent + đọc code): Sync Modal mỗi mode → endpoint đúng → NATS subject đúng → HandleTransmute chạy → activity log row được tạo (Start→Complete). Trace `file:line`.
4. Ghi `report_masters_sync_execute_2026-06-03.md`: files changed + LoC thực.

---

## 5. Task checklist
- [x] A1 transmute_handler.go activity log
- [x] A2 worker_server.go wiring
- [x] B1 create_master.go error check
- [x] B2 flatten guard shadow_schema
- [x] C1 MasterRegistry Sync Modal + flatten
- [x] C2 MasterMappingFieldsPage edit rule
- [x] C3 TransmuteSchedules tooltip
- [~] D1 git: 3 service ĐÃ có git riêng → KHÔNG init data-hub (đã gỡ .git lỡ tạo). Việc đang uncommitted → đề xuất restore-point commit (chờ User).
- [x] Build verify ×3 (worker/CMS go build+vet+test, FE tsc+build — tất cả PASS)
- [~] End-to-end audit: chain verified-by-trace + read-only DB (gap activity-log thật, target sssss hợp lệ). LIVE activity-row cần restart worker (binary đang chạy = bản cũ).
- [x] report_masters_sync_execute_2026-06-04.md + append 05_progress.md
