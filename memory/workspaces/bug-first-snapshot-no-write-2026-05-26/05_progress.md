# 05_progress — APPEND ONLY

## 2026-05-26 13:43 — Workspace khởi tạo
- Brain: chưa, Muscle thực thi.
- Đọc lessons.md (focus snapshot/cache/route), project_context.md.
- Đọc file code: snapshot_runner_handler.go, event_handler.go,
  batch_buffer.go, schema_adapter.go, metadata_registry_service.go,
  connection_manager.go, worker_server.go (subscribers).
- Đọc CMS side: snapshot_progress_handler.go, source_object_actions_handler.go,
  recon_async.go, register_registry.go, source_object_v2_sync.go, nats_client.go.

## 2026-05-26 13:55 — Root cause xác định
Chain failure:
1. `event_handler.processEvent` L86-89 silent-skip khi `ResolveSourceRoutes`
   rỗng → trả `(0, nil)`. Log ở mức Debug (operator không bao giờ thấy).
2. `snapshot_runner_handler.runSnapshot` L442 vứt giá trị `written`
   (`if _, err := r.eventHandler.HandleRaw(...)`).
3. L475 `rowsTotal += int64(len(batch))` đếm batch size, không phải
   actual writes.
4. `writeActivity` báo success với rows = số doc Mongo Find quét được,
   trong khi shadow 0 row.

Ba nguyên nhân làm cache empty cho first run:
- (a) Race với `schema.config.reload` NATS fire-and-forget.
- (b) `sourceRepo.GetActive` filter `is_active=true` (L109) → source
  vừa tạo với active=false bị skip.
- (c) `shadowRepo.ListBySourceObject` filter `binding.IsActive` (L125) →
  binding inactive bị skip.

## 2026-05-26 14:05 — Plan + docs viết xong
- 00_context.md, 01_requirements.md, 02_plan.md hoàn thành.
- Phase 1: event_handler log skip → WARN có context.
- Phase 2: snapshot_runner pre-flight reload + assert route.
- Phase 3: dùng `written` thật, treat 0 như doc error, đếm rowsTotal đúng.

## (sẽ append sau khi implement)

## 2026-05-26 14:30 — Implementation hoàn thành
- event_handler.go: skip log Debug → Warn (subject + db + table context).
- snapshot_runner_handler.go: pre-flight ReloadAll + hard-assert route + use written return + accurate rowsTotal.
- `go build ./...` PASS, `go vet ./internal/handler/...` PASS, `go test ./internal/handler/ -count=1` PASS (0.9s).
- report_first_snapshot_no_write_2026-05-26.md viết xong.
- 03_implementation.md, 09_tasks_solution.md viết xong.
- lesson L-CDC-route-empty-silent-skip-2026-05-26 append vào lessons.md.

## DoD audit (theo 01_requirements.md)
- [x] go build pass
- [x] processEvent skip log Debug → Warn
- [x] runSnapshot pre-flight reload + assert route
- [x] rowsTotal dùng written thật
- [x] written==0 treat như doc error
- [x] report_*.md ghi đủ
