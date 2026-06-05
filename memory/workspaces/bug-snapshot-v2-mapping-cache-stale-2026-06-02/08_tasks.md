# 08_tasks.md — Task list (REVISED)

| # | Task | File | Owner | Status | DoD |
|---|---|---|---|---|---|
| 1 | Thêm method `MapDataWithRules` | `dynamic_mapper.go` | Muscle | TODO | ~10 LOC, unit test pass |
| 2 | Snapshot query DB direct + cursor loop dùng rules-provided | `snapshot_runner_handler.go` | Muscle | TODO | ~10 LOC, build pass |
| 3 | Bỏ `r.registrySvc.ReloadAll(ctx)` trong snapshot | `snapshot_runner_handler.go` | Muscle | TODO | line cũ removed |
| 4 | Restart `centralized-data-service` | k8s | Muscle | TODO | Pod healthy |
| 5 | Smoke test theo `06_validation.md §1-7` | — | Muscle | TODO | Column rule mới có ở shadow |
| 6 | Regression check realtime CDC | — | Muscle | TODO | Stream event ok |
| 7 | User sign-off prod | — | User | TODO | OK |
| 8 | Update `05_progress.md` final + `07_status.md` → DONE | workspace | Brain | TODO | Status DONE |
