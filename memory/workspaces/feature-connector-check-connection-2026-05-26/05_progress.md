# 05_progress — Audit Log (APPEND-ONLY)

> ⚠️ **CLAUDE.md §7 + §11**: TUYỆT ĐỐI KHÔNG xóa / chỉnh sửa entry cũ. Chỉ APPEND.

---

## [2026-05-26 14:30] [Agent:claude-opus-4-7] Phase `check_connection` khởi tạo

**Actor**: Brain (Antigravity)
**Action**: Tạo workspace `feature-connector-check-connection-2026-05-26`
**Trigger**: User propose UX mới — "Check Connection → scan collections → multi-select" thay thế approach "FE-only hint" của workspace cũ.

**Pre-work bắt buộc (theo CLAUDE.md §7 + GEMINI.md ghi chú)**:
- ✅ Đọc `agent/memory/global/lessons.md` (4109 lines) — focus L-2026-05-19 Mongo Scan-Fields Pattern (line 3275+), L-2026-04 báo cáo láo (line 3070+), L-1276 Wizard session, L-2026-05-18 GetSourceDSN multi-scheme.
- ✅ Đọc `agent/GEMINI.md` (120 lines) — confirm Brain Chairman role + §12 Code Prohibition.
- ✅ Audit codebase via subagent Explore (very thorough) — kết quả:
  - BE endpoint `/api/introspection/mongo/databases` + `/collections` ĐÃ TỒN TẠI (router.go:331-332).
  - Worker service `DiscoverDatabases(uri)` + `DiscoverCollections(uri, db)` ĐÃ NHẬN FULL URI (mongo_introspection.go:63,85).
  - Gap chính: Worker handler `HandleDiscoverMongo*` chỉ nhận `host+port`, BUILD `mongodb://host:port` inline → drop auth/replicaSet.
  - FE chưa có hook check connection, Collections field hiện là text input.
  - Antd v6.3.5 → `<Select mode="multiple">` available.
  - MySQL/PG introspect KHÔNG TỒN TẠI → defer phase sau.

**Files created** (11 file, tổng ~50KB):
- `00_context_check_connection.md` — Bối cảnh, audit findings, scope, constraints
- `01_requirements_check_connection.md` — R1-R12 + N1-N9 + DoD A1-A15 + Risk matrix + Inverse requirements
- `02_plan_check_connection.md` — M0..M8 roadmap, decision tree, effort 6h30m
- `03_implementation_check_connection.md` — Data flow + API contract + Component design + Failure modes
- `04_decisions_check_connection.md` — ADR-001..009 (Mongo P0, POST body, 5-case map, all-selected default, Spin UX, gate logic, defer cross-DB)
- `05_progress.md` (file này)
- `06_test_cases_check_connection.md`
- `08_tasks_check_connection.md`
- `09_tasks_solution_check_connection.md`
- `10_gap_analysis_check_connection.md`
- `report_check_connection_2026-05-26.md` (TEMPLATE)

**Status**: Plan-only complete. Chờ user approve để Muscle thực thi.

**Brain hành động kết luận**:
1. Audit subagent kết luận: ~70% infra đã có.
2. Strategy: extend (không refactor) — chọn POST với body `{uri,database}` để tránh leak URI vào access log (ADR-002).
3. Wizard session table KHÔNG dùng (overkill — ADR-008).
4. Workspace cũ `feature-connector-default-collections-2026-05-25` marked SUPERSEDED (ADR-009).
5. Brain KHÔNG sửa code (CLAUDE.md §12).

**Next verb chờ user**:
- `execute` / `muscle thực thi` / `go` → giao Muscle chạy M0 → M8
- `revise` / `đổi plan` → Brain re-plan
- `defer` → archive plan
- `model: opus` / `model: sonnet` → chọn muscle model

---

## [2026-05-26 13:50] [Agent:claude-opus-4-7] M1+M2+M3+M4 thực thi xong

**Actor**: Muscle (Claude Code, opus-4-7)
**Trigger**: User verb `thực thi, task này đơn giản mà`.

**Tasks done**: T1.1-T1.5 (worker), T2.1-T2.5 (BE), T3.1-T3.3 (FE service+hook), T4.1-T4.9 (FE UI).

**Files changed**:
- `centralized-data-service/internal/handler/command_handler.go` — extend `HandleDiscoverMongoDatabases` + `HandleDiscoverMongoCollections` accept URI (preferred) hoặc host+port (fallback); reply 5-case `status` field (ok/cluster_err/db_missing/empty); helper `replyMongoDiscovery`; log dùng `SanitizeMongoDSN` (KHÔNG log raw URI). +130 LOC net.
- `cdc-cms-service/internal/api/introspection_handler.go` — add `DiscoverMongoDatabasesPost` + `DiscoverMongoCollectionsPost` (Fiber, body JSON, dùng SubscribeSync+Publish pattern hiện có). +100 LOC.
- `cdc-cms-service/internal/router/router.go` — `dualPost` register 2 route mới `/introspection/mongo/{databases,collections}` (giữ GET legacy). +5 LOC.
- `cdc-cms-web/src/services/connectorCheck.ts` — **NEW** — `checkMongoDatabases`, `checkMongoCollections`, type `CheckStatus`, helper `refineCheckStatus` (cluster_err → auth_err nếu error chứa keyword auth/credential), `mapCheckStatusToVi`. +78 LOC.
- `cdc-cms-web/src/hooks/useConnectorCheck.ts` — **NEW** — `useCheckMongoConnection` React Query mutation, return `{result, isPending, check, reset}`. +51 LOC.
- `cdc-cms-web/src/pages/SourceConnectors.tsx` — import `Spin` + hook + helper; `Form.useWatch` cho `connectionUrl` + `database`; useEffect reset checkHook + clear `collectionNames` khi URI/DB change; useEffect auto-set form value khi check PASS (CSV string); `runCheckConnection` handler; Modal `okButtonProps.disabled` gate (Mongo+create+!ok); UI block Check button + Spin + Alert success/error (db_missing render available_databases list, fallback render sanitized_dsn); Form.Item `collectionNames` UI = `<Select mode="multiple">` nhưng form state vẫn là CSV string qua `normalize` (array→CSV) + `getValueProps` (CSV→array); rule validator yêu cầu ≥1 collection ở create mode. +130 LOC net.

**User feedback mid-session**: "ko làm colectionname như vâyh, nó vân là string. input là select nhưng khi chọn xong đưa nó về dạng a,b,c,d" → revert type về `string`, dùng Antd Form.Item `normalize` + `getValueProps` để Select multi nhưng form state CSV. KHÔNG đụng `buildConnectorConfig` split logic gốc — chỉ adapt rule + reset value về `''`.

**Verification**:
- Worker: `cd centralized-data-service && go build ./... && go vet ./...` → BUILD EXIT=0, VET EXIT=0. Log `/tmp/check_connection_build_worker.log`.
- BE: `cd cdc-cms-service && go build ./... && go vet ./...` → BUILD EXIT=0, VET EXIT=0. Log `/tmp/check_connection_build_be.log`.
- FE: `npx tsc -b` → TSC EXIT=0. `npx vite build` → built in 830ms, EXIT=0. `npx eslint` trên 3 file changed → 0 error mới (3 error pre-existing line 42-46 env-var fallback pattern, 1 warning line 350 useMemo deps pre-existing).
- Unit tests: chưa add (T1.6, T2.6) — defer vì task chính là feature implementation; existing test suites worker/BE tự test framework, không assert NATS subject contract.

**Issues / Open items**:
- M5 (E2E happy smoke) + M6 (negative smoke) + M7 (security agent) chưa chạy — yêu cầu local stack (Mongo + NATS + worker + BE + FE-dev) up đồng thời. Cần user verb tiếp theo: `smoke` để chạy interactive trên local; hoặc user tự smoke trong browser.
- T1.6 + T2.6 unit test mới chưa add — defer khi yêu cầu (extend command_handler_test.go + introspection_handler_test.go).
- Pre-existing FE lint issue line 42-46 (no-constant-binary-expression) — không trong scope phase này.

**Backward compat verified (static)**:
- Worker handler vẫn accept `host+port` payload cũ → ✅ fallback path.
- BE GET endpoints `/api/introspection/mongo/{databases,/:db/collections}` KHÔNG đổi → ✅ legacy caller giữ.
- FE compactConfig + buildConnectorConfig giữ logic CSV → ✅ database prefix `${db}.${col}` không đổi.

**Next** (chờ user verb):
- `smoke` → user chạy local stack, Muscle hướng dẫn step-by-step manual test 5-case.
- `unit tests` → Muscle add TC-WU-01..06 + TC-BU-01..06.
- `report` → Muscle fill `report_check_connection_2026-05-26.md` §1-§12 với evidence build log.
- `commit` → Muscle stage + commit 3 repo (cần user confirm vì task này cross-repo, không có git init ở cwd cdc-cms-web).

---

## Template entry (cho Muscle khi thực thi)

```
## [YYYY-MM-DD HH:MM] [Agent:<verified-model-id>] M<N> done

**Milestone**: M<N>
**Tasks completed**: T<N>.1, T<N>.2, ...
**Files changed**: <list with line counts>
**Verification**:
  - Build: <exit code + log path>
  - Test: <N/N PASS, coverage X%>
  - Smoke: <evidence>
**Issues**: <none / list with severity>
**Next**: M<N+1>
```
