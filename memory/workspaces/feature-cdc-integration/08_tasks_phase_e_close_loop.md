# 08 — Atomic tasks Phase E

| ID | Task | Owner | Depends | Est | Verify |
|---|---|---|---|---|---|
| E3-1 | Brain check file `prune_legacy_v1_bindings.sql` tồn tại + nội dung idempotent | Brain | - | 2m | grep `WHERE is_active = true` |
| E3-2 | Muscle exec script lần 1 | Muscle | E3-1 | 2m | output `pruned_sources=10` |
| E3-3 | Muscle exec script lần 2 (idempotency) | Muscle | E3-2 | 2m | output `pruned_sources=0` |
| E3-4 | Brain APPEND `05_progress` E3 closure | Brain | E3-3 | 2m | wc -l increased |
| E4-1 | Muscle run 5 diagnostic queries cho `orders_addtest` | Muscle | - | 5m | output captured |
| E4-2 | Muscle write `report_g4_diag_<ts>.md` (file vật lý) | Muscle | E4-1 | 5m | file exists |
| E4-3 | Brain APPEND `05_progress` E4 closure | Brain | E4-2 | 2m | - |
| E1-1 | Muscle EDIT `helpers.go::extendDatabaseList` (new helper) | Muscle | - | 10m | go build PASS |
| E1-2 | Muscle EDIT `helpers.go::extendDebeziumInclude` 2-tier dispatch | Muscle | E1-1 | 10m | go build PASS |
| E1-3 | Muscle EDIT `types.go::RegisterSourceResponse` thêm `Warnings []string` | Muscle | E1-2 | 5m | go build PASS |
| E1-4 | Muscle EDIT `source_register.go` set Warnings từ wasAdded | Muscle | E1-3 | 5m | go build PASS |
| E1-5 | Muscle EDIT `server_test.go` thêm 3 test cho 2-tier | Muscle | E1-4 | 15m | go test PASS |
| E1-6 | Muscle restart admin-api (kill + go run) | Muscle | E1-5 | 2m | curl /healthz returns 200 |
| E1-7 | Muscle smoke live: PUT collection ở namespace mới | Muscle | E1-6 | 5m | connector config has both tiers |
| E1-8 | Brain verify connector config delta + APPEND `05_progress` E1 closure | Brain | E1-7 | 5m | jq output OK |
| E2-1 | Muscle PUT register `payment_bills_addtest` | Muscle | E1-8 | 3m | 200 OK |
| E2-2 | Muscle INSERT mongo doc | Muscle | E2-1 | 1m | 1 doc inserted |
| E2-3 | Muscle wait 30s, query shadow count | Muscle | E2-2 | 1m | shadow >= 1 |
| E2-4 | Muscle wait 60s, query master count | Muscle | E2-3 | 1m | master >= 1 |
| E2-5 | Brain APPEND `05_progress` E2 closure | Brain | E2-4 | 2m | - |
| E5-1 | Brain invoke `/security-agent` skill | Brain | E2-5 | 15m | report file generated |
| E5-2 | Brain APPEND `05_progress` E5 closure + summarize HIGH issues | Brain | E5-1 | 5m | - |
| EX-1 | Brain write `report_phase_e_close_loop_<ts>.md` (master report) | Brain | E5-2 | 15m | file exists |
| EX-2 | Brain end-to-end verify all 5 acceptance criteria | Brain | EX-1 | 5m | all PASS |
| EX-3 | Brain optional: APPEND lesson nếu phát sinh | Brain | EX-2 | 5m | - |

**Total**: ~24 task, ETA ~2h.

## Sequencing rules

- E3 trước E4 trước E1 (zero risk → diagnostic → code change).
- E2 BLOCKED đến khi E1-7 PASS (G7 fix prerequisite).
- E5 sau cùng (cần code stable để security review không bị nhiễu).
- Mỗi closure entry trong `05_progress` ghi đầy đủ: file changed, build/test result, smoke result, lesson nếu có.
