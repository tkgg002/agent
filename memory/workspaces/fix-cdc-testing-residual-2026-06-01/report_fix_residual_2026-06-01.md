# Report — Fix CDC Testing Residual (2026-06-01)

## Tóm tắt
- 8/8 gap residual đã đóng theo wave ưu tiên P0 → P2.
- Toàn bộ unit test + build PASS trên các package đã chạm.
- Áp dụng §6 Simplicity First: không thêm Makefile/README phụ trợ, không refactor ngoài scope.

## Wave 1 (P0) — Đã hoàn thành trước compaction
| Gap     | File                                                                  | LOC delta | Verify                         |
|---------|-----------------------------------------------------------------------|-----------|--------------------------------|
| G5-RES  | cdc-cms-service/test/internal/api/mapping_rule_handler_test.go        | ±2        | test PASS                      |
| G5-RES  | cdc-cms-service/test/internal/app/commands/sync_metadata_test.go      | ±1        | test PASS                      |
| G8-RES  | agent/memory/workspaces/plan-cdc-qa-gap-fix-2026-05-27/05_progress.md | +30 APPEND| n/a (doc)                      |
| G3-RES  | centralized-data-service/pkgs/database/metrics_callback_test.go       | +72 NEW   | 2/2 PASS                       |
| G1-RES  | centralized-data-service/pkgs/metrics/prometheus.go                   | +8        | build PASS                     |
| G1-RES  | centralized-data-service/internal/handler/kafka_consumer.go           | +3        | build PASS                     |

## Wave 2 (P1)
| Gap     | File                                                              | LOC delta | Verify                                  |
|---------|-------------------------------------------------------------------|-----------|-----------------------------------------|
| G4-RES  | centralized-data-service/scripts/load_test_cdc.js                 | +93 NEW   | node --check SYNTAX_OK                  |
| G2-RES  | centralized-data-service/internal/service/wal_monitor.go          | +201 NEW  | build PASS                              |
| G2-RES  | centralized-data-service/internal/service/wal_monitor_test.go     | +93 NEW   | 4/4 PASS in 0.746s                      |
| G2-RES  | centralized-data-service/pkgs/metrics/prometheus.go               | +10       | metric WALSnapshotResumeTotal active    |

## Wave 3 (P2)
| Gap     | File                                                                  | LOC delta | Verify                          |
|---------|-----------------------------------------------------------------------|-----------|---------------------------------|
| G7-RES  | centralized-data-service/internal/handler/kafka_consumer.go           | +24       | build PASS                      |
| G7-RES  | centralized-data-service/internal/handler/adaptive_batcher_test.go    | +82 NEW   | 4/4 PASS in 0.667s              |
| G7-RES  | centralized-data-service/pkgs/metrics/prometheus.go                   | +11       | metric DestThrottledTotal active|
| G6-RES  | centralized-data-service/scripts/chaos_network.sh                     | +78 REWRITE | bash -n SYNTAX_OK             |

## Tổng LOC delta
- **Code Go**: +432 (4 file mới + 2 patch nhỏ)
- **Test Go**: +247 (3 file test mới)
- **Scripts**: +93 (k6) + 78 (chaos) - 26 (cũ chaos) = +145 ròng
- **Docs/workspaces**: +5 file workspace + APPEND lessons

## Verify suite cuối (re-run)
```
go test ./internal/handler/ ./internal/service/ ./pkgs/database/ ./pkgs/metrics/
ok    centralized-data-service/internal/handler   0.762s
ok    centralized-data-service/internal/service   0.536s
ok    centralized-data-service/pkgs/database      1.214s
?     centralized-data-service/pkgs/metrics       [no test files]
EXIT=0

cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
go test ./test/internal/api/ ./test/internal/app/commands/
ok    cdc-cms-service/test/internal/api          1.005s
ok    cdc-cms-service/test/internal/app/commands 0.477s
EXIT=0
```

## Vet status
- Pre-existing noise tại `pkgs/idgen/sonyflake.go:77,82` (sync.Once copy), không liên quan patches.
- Pre-existing noise tại `scratch/` (main redeclared), không nằm trong release path.
- Zero new warning từ patches.

## Maturity update (dự kiến)
- Trước rerun: L3, 50/64 (78.1%).
- Sau khi đóng 8 gap (4 Tier code + 4 Tier script/doc): L4 đạt được với điều kiện:
  - G2 wired vào `cmd/worker/main.go` (TODO follow-up: bật `NewWALMonitor(...)` + `go monitor.Run(ctx)`).
  - G7 wired bằng `kc.SetDestHealthCheck(healthFn)` trong wiring.
  - G4/G6 chạy thử trên staging với pumba/xk6 binary cài sẵn.

## Risk & rollback
- G2 service **chưa được wire** vào worker startup → zero runtime impact cho đến khi orchestrator opt-in. Patch an toàn để merge.
- G7 destHealth nil-by-default → giữ behavior cũ exact. Opt-in qua `SetDestHealthCheck`.
- G1 metric mới chỉ thêm series, không xóa series cũ.
- G4/G6 là test/chaos scripts — không chạm runtime.

## Lesson candidates (để APPEND vào lessons.md sau gate review)
- `Global Pattern [A wires telemetry-only metric B for failover signal X] → Result Y: must paired with Set() call at source-of-truth (ticker), không chỉ declare.` — gốc G1-RES.
- `Global Pattern [A claims test pass for module B without artifact X on filesystem] → FAKE detection by Y: ls + cat verify before mark done.` — gốc G3-RES.
- `Global Pattern [A clamps adaptive throttle B by health probe X] → Result Y: bypass throttle's own time-gate khi probe=false (urgent brake semantics).` — gốc G7-RES.

## Skills sử dụng (CLAUDE.md §0)
- Bash, Read, Write, Edit, Grep
- TaskList / TaskUpdate (workflow tracking)
- ScheduleWakeup không dùng (không có long-poll)
- Subagents không dùng (scope đủ nhỏ để direct exec)
