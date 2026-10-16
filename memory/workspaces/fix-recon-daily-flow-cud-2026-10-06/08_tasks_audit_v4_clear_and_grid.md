# 08 — Tasks: Audit V4

## P1 (chờ APPROVE)
- [x] T1 `batch_buffer.go`: `shadow_schema` vào details (Complete + FailWithDetails)
- [x] T2 `batch_buffer_fanout.go`: `triggered_by = "snapshot"` khi `isSnapshotBuffer`
- [x] T3 `transmute_handler.go`: forward `snapshot-hook`; `groupDebouncedTasks` + `runDebouncedGroup`; `master_binding_id` ở 6 điểm log; details cho `transmute_sub`
- [x] T4 Unit test `groupDebouncedTasks` (mixed binding/trigger, thứ tự giữ nguyên)
- [x] T5 `activity_log_read_repo_gorm.go`: predicate `shadow_schema`/`master_binding_id` ở LATERAL + `scope_counts`; viết lại `ClearActivity`
- [x] T6 `list_activity_logs.go`: `ClearActivityParams{…, Before *time.Time}`
- [x] T7 `activity_log_handler.go`: validate scope + RFC3339
- [x] T8 `router.go`: chuyển sang `destructiveChain`
- [x] T9 FE hook + grid (payload, op `transmute,transmute_sub`, snapshot-hook, dayjs local, bỏ `'cdc_shadow'`, `status==='error'`)
- [x] T10 `go build` + `go test ./internal/handler/master/... ./internal/handler/shadow/...` (CDS), `go build`/`go test ./...` (CMS), `npm run build` (web)
- [x] T11 Kịch bản SQL 2 connector trùng tên trên Postgres docker (xem `06_validation_audit_v4.md`)
- [x] T12 /security-agent review (route destructive, SQL binding)
- [ ] T13 Commit tách riêng theo feature (FE đang gộp diff nhiều phiên từ 30/09)

## P2 (sau P1)
- [ ] T14 Query `daily-flow` server-side + FE dùng thay reducer

## Cập nhật 2026-10-12 11:18
- [x] T9b (phát sinh khi chạy T11) FE `ReconPipelineGrid.tsx`: modal Clear truyền `masterTable={historyMaster}` (bare) thay vì `pipeline.masterName` (FQN) — trước đó transmute logs không bao giờ bị xoá.
- T13: chờ User commit (diff FE/CDS gộp nhiều phiên chưa commit).
- T14 (P2): chờ User quyết định.
