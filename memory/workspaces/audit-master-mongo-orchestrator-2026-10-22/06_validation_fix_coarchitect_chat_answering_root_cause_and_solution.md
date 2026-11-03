# 06. Validation — Kiểm chứng Vật lý & Biên lai Runtime (`fix_coarchitect_chat_answering_root_cause_and_solution`)

## Danh mục Biên lai Kiểm chứng Vật lý (`EV-...` qua `evidence_run.py` sau lần sửa code cuối `TASK-CHAT-13` & `TASK-CHAT-14`)

| STT | Hạng mục Kiểm chứng | Lệnh Thực thi | Biên lai (`EV-...`) | Kết quả |
| :---: | :--- | :--- | :--- | :---: |
| 1 | Kiểm tra tĩnh toàn bộ codebase (`go vet`) | `go vet ./...` | `EV-20261027T163234-4a05` | `exit=0` (`PASS` — `EV-20261027T163234-4a05`) |
| 2 | Build Production Binary `./bin/orchestrator` | `go build -o bin/orchestrator ./cmd/orchestrator` | `EV-20261027T163234-53bb` | `exit=0` (`PASS` — `EV-20261027T163234-53bb`) |
| 3 | Unit Test `-race` cho 3 package layer (`internal/gitclient`, `internal/notify`, `internal/runner`) | `go test -v -race ./internal/gitclient/... ./internal/notify/... ./internal/runner/...` | `EV-20261027T163237-de4e` | `exit=0` (`PASS` — `EV-20261027T163237-de4e`) |
| 4 | Integration Test `-race` `TestTelegramCompactFormat` & `TestArchitectLLMBridge_SkipDistAndBoilerplateAndRePlanV1V2` | `go test -v -race -run TestTelegramCompactFormat ./internal/webui` & `TestArchitectLLMBridge_SkipDistAndBoilerplateAndRePlanV1V2` | `EV-20261027T163251-3ea7`, `EV-20261027T163346-cf8b` | `exit=0` (`PASS` — `EV-20261027T163251-3ea7`, `EV-20261027T163346-cf8b`) |
| 5 | Runtime Check `go run ./cmd/orchestrator daemon --once` | `go run ./cmd/orchestrator daemon --once` | `EV-20261027T163300-dab2` | `exit=0` (`PASS` — `EV-20261027T163300-dab2`) |
| 6 | Live Runtime HTTP `/api/dashboard?task_key=Core-01` & CSS Full-Height Scroll trên `:8090` (`PID 54761`) | `curl -fsS -o /tmp/core01_dash.json http://127.0.0.1:8090/api/dashboard?task_key=Core-01` | `EV-20261027T163335-7c60` | `exit=0` (`PASS` — `EV-20261027T163335-7c60`) |
| 7 | Cổng `/laya_router` `DG3` (`Mode A: run --stage dg3` + `Mode B: review`) sau lần sửa code cuối (`TASK-CHAT-13` & `TASK-CHAT-14`) | `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 ...` & `laya_gate.py review ...` | `06_laya_summary_dg3_20261027T163637.json` & `06_laya_summary_dg3review_20261027T163836.json` | `gate=PASS` (`EV-20261027T163335-7c60`) |

