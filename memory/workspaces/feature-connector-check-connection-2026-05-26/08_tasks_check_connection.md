# 08_tasks_check_connection — Task Checklist

> **Phase**: `check_connection`
> **Owner Muscle**: TBD (sonnet-4-6 default; user có thể override `model: opus`)
> **Execution order**: TUẦN TỰ M0 → M8
> **Pre-condition**: User approve plan, verb "execute"

---

## M0 — Pre-flight audit verify

- [ ] **T0.1** Read `centralized-data-service/internal/service/mongo_introspection.go` lines 1-200. Verify signature `DiscoverDatabases(uri string)` + `DiscoverCollections(uri, dbName string)` + struct `IntrospectDiagnosis`. Quote vào `05_progress.md` Entry M0.
- [ ] **T0.2** Read `centralized-data-service/internal/handler/command_handler.go` lines 1162-1230. Quote `HandleDiscoverMongoDatabases` + `HandleDiscoverMongoCollections` DTO struct + URI build logic.
- [ ] **T0.3** Read `cdc-cms-service/internal/api/introspection_handler.go` lines 1-150. Quote DTO request/response + NATS subject string.
- [ ] **T0.4** Read `cdc-cms-web/src/pages/SourceConnectors.tsx` lines 107 (regex), 286-295 (state), 148-220 (buildConnectorConfig), 878-1026 (Modal render). Quote.
- [ ] **T0.5** Read `cdc-cms-web/src/services/api.ts` để verify axios setup pattern.
- [ ] **T0.6** Test existing endpoint local: `curl -s 'http://localhost:8001/api/introspection/mongo/databases?host=localhost&port=27017' -H "Authorization: Bearer <jwt>" | jq`. Document JWT acquisition flow.
- [ ] **T0.7** Verify Mongo local topology: `mongosh "mongodb://localhost:27017" --eval 'db.adminCommand({hello:1})'`. Note `setName` (rs) hay standalone.
- [ ] **T0.8** APPEND `05_progress.md` `## [YYYY-MM-DD HH:MM] [Agent:<model>] M0 done` với quotes + endpoint output.

**Exit gate**: T0.6 fail (endpoint không tồn tại) → STOP, escalate.

---

## M1 — Worker handler extend (centralized-data-service)

- [ ] **T1.1** Edit `command_handler.go` ~line 1164 `HandleDiscoverMongoDatabases`:
  - Extend DTO: thêm `Uri string \`json:"uri"\``
  - Logic: `uri := req.Uri; if uri == "" { uri = fmt.Sprintf("mongodb://%s:%s", req.Host, req.Port) }`
  - Validate: `if uri == "mongodb://:" { return error "missing connection" }`
  - Reference: `09_tasks_solution_check_connection.md` Edit #1.
- [ ] **T1.2** Edit `command_handler.go` ~line 1204 `HandleDiscoverMongoCollections`: tương tự + thêm validate `req.Database != ""`. Reference Edit #2.
- [ ] **T1.3** Verify reply payload include sanitized DSN + available_databases (cho db_missing case). Nếu service `DiscoverCollectionsDiagnose` chưa expose → optional bổ sung wrapper. Reference Edit #2 variant.
- [ ] **T1.4** `cd centralized-data-service && go build ./... 2>&1 | tee /tmp/check_connection_build_worker.log`. Exit 0.
- [ ] **T1.5** `go vet ./... 2>&1 | tee -a /tmp/check_connection_build_worker.log`. Exit 0.
- [ ] **T1.6** Add unit test theo TC-WU-01..06 trong `06_test_cases_check_connection.md`. File: `internal/handler/command_handler_test.go` (extend hoặc tạo nếu chưa có).
- [ ] **T1.7** `go test ./internal/handler/... -v -run TestHandleDiscoverMongo 2>&1 | tee /tmp/check_connection_worker_test.log`. All PASS.
- [ ] **T1.8** APPEND `05_progress.md` "M1 done — worker extended, 6 unit tests PASS".

---

## M2 — BE relay extend (cdc-cms-service)

- [ ] **T2.1** Edit `introspection_handler.go:25-75` `DiscoverMongoDatabases`: thêm POST variant nhận body `{uri}`. Reference Edit #3.
- [ ] **T2.2** Edit `introspection_handler.go:77-150` `DiscoverMongoCollections`: thêm POST nhận body `{uri, database}`. Reference Edit #4.
- [ ] **T2.3** Edit `router.go:331-332` thêm:
  ```go
  apiGroup.POST("/introspection/mongo/databases", h.DiscoverMongoDatabasesPost)
  apiGroup.POST("/introspection/mongo/collections", h.DiscoverMongoCollectionsPost)
  ```
  Reference Edit #4.5.
- [ ] **T2.4** `cd cdc-cms-service && go build ./... 2>&1 | tee /tmp/check_connection_build_be.log`. Exit 0.
- [ ] **T2.5** `go vet ./... 2>&1 | tee -a /tmp/check_connection_build_be.log`. Exit 0.
- [ ] **T2.6** Add unit test TC-BU-01..06. File: `internal/api/introspection_handler_test.go`.
- [ ] **T2.7** `go test ./internal/api/... -v 2>&1 | tee /tmp/check_connection_be_test.log`. All PASS.
- [ ] **T2.8** Integration smoke local: `curl -X POST http://localhost:8001/api/introspection/mongo/collections -H "Authorization: Bearer <jwt>" -H "Content-Type: application/json" -d '{"uri":"mongodb://localhost:27017","database":"goopay_pbs"}' | jq`. Verify 200 với collections list.
- [ ] **T2.9** APPEND `05_progress.md` "M2 done — BE relay extended, integration smoke OK, evidence: <log path>".

---

## M3 — FE service + hook

- [ ] **T3.1** CREATE `cdc-cms-web/src/services/connectorCheck.ts` với 2 function: `checkMongoDatabases({uri})` + `checkMongoCollections({uri, database})`. Reference Edit #5.
- [ ] **T3.2** CREATE `cdc-cms-web/src/hooks/useConnectorCheck.ts` với `useCheckMongoConnection()` (React Query `useMutation`). Trả state shape `{result, isPending, check, reset}`. Reference Edit #6.
- [ ] **T3.3** `cd cdc-cms-web && pnpm tsc --noEmit 2>&1 | tee /tmp/check_connection_tsc.log`. Exit 0.
- [ ] **T3.4** (Optional) Unit test hook (TC-FU-01..03) nếu Jest setup tồn tại. Skip nếu không có.
- [ ] **T3.5** APPEND `05_progress.md` "M3 done — FE service+hook created".

---

## M4 — FE UI integration

- [ ] **T4.1** Edit `SourceConnectors.tsx` extend state:
  - Import hook + types
  - `const checkHook = useCheckMongoConnection()`
  - Watch URI + Database value via `Form.useWatch`
  - `useEffect` reset checkHook khi URI/DB change
  Reference Edit #7.
- [ ] **T4.2** Edit Modal: thêm `<Button>Check Connection</Button>` ngay sau field Database (dạng inline `<Form.Item>` hoặc bên cạnh). Click handler gọi `checkHook.check({uri, database})`. Reference Edit #8.
- [ ] **T4.3** Edit Form field Collections (line ~966): REPLACE `<Input placeholder="users,orders,payments" />` bằng:
  ```tsx
  <Select
    mode="multiple"
    placeholder="Chọn collections..."
    options={(checkHook.result?.collections ?? []).map(c => ({value: c, label: c}))}
    disabled={!checkHook.result || checkHook.result.status !== 'ok'}
    showSearch
    allowClear
  />
  ```
  Reference Edit #9.
- [ ] **T4.4** Trên success: trong `onSuccess` callback của mutation, set form value: `form.setFieldValue('collectionNames', result.collections)` (auto-select-all). Reference Edit #10.
- [ ] **T4.5** Edit Modal `okButtonProps`: `{ disabled: editorMode === 'create' && (!checkHook.result || checkHook.result.status !== 'ok') }`. Edit mode không gate. Reference Edit #11.
- [ ] **T4.6** Edit error display: ngay dưới button Check, render `<Alert type="error" message={mapStatusToVi(checkHook.result)}>` khi status !== 'ok' && status !== null. Reference Edit #12 + Edit #13 (helper).
- [ ] **T4.7** Update `buildConnectorConfig` (line ~148-220): xử lý `collectionNames` từ `string | string[]`. Nếu array → `join(',')` với prefix db (Debezium spec: `db.col,db.col`). Reference Edit #14.
- [ ] **T4.8** Edit existing connector load path (~line 410, `onClickEdit`): khi load config có `collection.include.list` → split → set form value `collectionNames = [...]` array. Reference Edit #15.
- [ ] **T4.9** `pnpm tsc --noEmit && pnpm lint && pnpm build 2>&1 | tee /tmp/check_connection_build_fe.log`. Exit 0.
- [ ] **T4.10** APPEND `05_progress.md` "M4 done — FE UI integrated, build PASS".

---

## M5 — Smoke E2E happy path

- [ ] **T5.1** Spin local stack: confirm worker + cdc-cms + FE-dev + Mongo + Kafka Connect UP.
- [ ] **T5.2** `pnpm dev` cho FE. Open browser, login, navigate Create Connector page.
- [ ] **T5.3** TC-E-01: form kiểu MongoDB, URI `mongodb://localhost:27017`, Database `goopay_pbs`, click [Check Connection]. Verify Spin visible. Wait. Verify multi-select hiện collections + tất cả selected.
- [ ] **T5.4** Compare collections list với `mongosh "mongodb://localhost:27017/goopay_pbs" --eval 'db.getCollectionNames()'`. Phải match.
- [ ] **T5.5** Click Create. Verify toast success + entry trong list.
- [ ] **T5.6** Verify Kafka Connect: `curl -s http://localhost:8083/connectors/<name>/config | jq '.["collection.include.list"]'`. Phải có explicit list. Save `/tmp/check_connection_smoke_happy.log`.
- [ ] **T5.7** TC-E-02: tạo connector thứ 2 nhưng UNCHECK 2 collections trong multi-select trước Create. Verify Kafka Connect config chỉ có collection được check.
- [ ] **T5.8** TC-E-10: edit connector cũ (`collection.include.list = "a,b"`). Verify multi-select pre-fill `[a,b]`. KHÔNG ép re-check. Edit button enabled.
- [ ] **T5.9** Screenshot tất cả step → `/tmp/check_connection_screens/`.
- [ ] **T5.10** APPEND `05_progress.md` "M5 done — happy path PASS, screenshots: <paths>".

---

## M6 — Smoke E2E negative cases

- [ ] **T6.1** TC-E-03 cluster_err: URI `mongodb://localhost:9999` → Check → Alert VN visible, Create disabled. Screenshot.
- [ ] **T6.2** TC-E-04 db_missing: URI ok, Database `nonexistent_xyz` → Check → Alert + danh sách DB có sẵn hiển thị. Screenshot.
- [ ] **T6.3** TC-E-06 auth_err: URI `mongodb://wrong:wrong@localhost:27017/?authSource=admin` → Check → Alert đề cập auth. Screenshot.
- [ ] **T6.4** TC-E-07 state invalidate: sau TC-E-01 step pre-Create, đổi URI → verify multi-select disabled + Create disable. Screenshot.
- [ ] **T6.5** TC-E-09 spam click: click Check 10 lần trong 5s. Verify button disabled while pending. Network tab max 1 inflight.
- [ ] **T6.6** TC-E-13 worker offline: `kill <worker-pid>` → click Check → wait 10s → Alert "Worker không phản hồi". Restart worker.
- [ ] **T6.7** TC-I-06 backward compat: `curl -s 'http://localhost:8001/api/introspection/mongo/databases?host=localhost&port=27017' -H "Authorization: Bearer <jwt>" | jq`. Verify 200 với databases.
- [ ] **T6.8** APPEND `05_progress.md` "M6 done — negative TCs PASS".

---

## M7 — Security review

- [ ] **T7.1** Run `/security-agent` trên file list edited: worker `command_handler.go`, BE `introspection_handler.go`, FE `SourceConnectors.tsx` + `connectorCheck.ts` + `useConnectorCheck.ts`. Save `/tmp/check_connection_security.log`.
- [ ] **T7.2** TC-S-01: `grep -E 'mongodb://[^/]+:[^@]+@' /tmp/worker.log /tmp/cdc-cms.log` → expect 0 hits. Save `/tmp/check_connection_log_grep_password.log`.
- [ ] **T7.3** TC-S-02: Network tab inspect response của TC-I-03 (URI có auth). Verify `sanitized_dsn` field không có password.
- [ ] **T7.4** TC-S-03 XSS: Mongo create collection tên `<script>alert('xss')</script>` (nếu test allow), check render trong multi-select. Verify Antd escape OK.
- [ ] **T7.5** TC-S-05 JWT missing: `curl -X POST http://localhost:8001/api/introspection/mongo/collections -d '{"uri":"...","database":"..."}'` (no auth header) → 401.
- [ ] **T7.6** APPEND `05_progress.md` "M7 done — security <PASS / N findings: ...>".

---

## M8 — Report + memory update + supersede old workspace

- [ ] **T8.1** Fill `report_check_connection_2026-05-26.md`:
  - §1 Executive summary
  - §2 Files changed (path + LOC + diff summary)
  - §3 Verify gates M0..M7 results với evidence
  - §4 Behavior changes table
  - §5 Screenshots inline
  - §6 Rollback plan
  - §7 Lessons learned
  - §8 Open items / Defer
  - §9 Pre-flight check
- [ ] **T8.2** APPEND `agent/memory/global/lessons.md` Global Pattern:
  ```
  ## [YYYY-MM-DD] Pre-validate connection before create entity (5-case probe + multi-select UI)

  **Global Pattern**: Khi feature F yêu cầu tạo entity E phụ thuộc remote source S (DB, API, cluster) với credentials C, kiến trúc đúng = (1) sync probe endpoint P probe S, (2) trả về diagnosis D 5-case (cluster_err / namespace_missing / entity_missing / empty / data_shape_invalid), (3) trả về candidate list L cho UI multi-select, (4) UI gate "Create" button trên D=ok. KHÔNG dùng wizard session table cho state ephemeral.

  Variables: F, E, S, C, P, D, L
  Đúng: probe sync → diagnosis → candidate list → UI gate.
  Sai: tạo E ngay, validate sau → entity lỗi rò rỉ vào DB → tốn audit.
  ```
- [ ] **T8.3** APPEND lesson về reuse-vs-rebuild: "Khi user request feature X, audit codebase TRƯỚC. ~70% infra X có thể đã tồn tại từ feature Y khác. Strategy = extend, không refactor".
- [ ] **T8.4** Update `agent/memory/global/active_plans.md`:
  - `feature-connector-check-connection-2026-05-26`: DONE ✅
  - `feature-connector-default-collections-2026-05-25`: SUPERSEDED ⏭️ (by check-connection)
- [ ] **T8.5** APPEND `05_progress.md` "M8 done — phase check_connection COMPLETE — report at <path>".
- [ ] **T8.6** Pre-flight §14: `ls -la agent/memory/workspaces/feature-connector-check-connection-2026-05-26/` — verify 11 file tồn tại vật lý. Cross-check report §9 checklist.
- [ ] **T8.7** Report user verb hoàn thành (PR draft URL nếu Muscle tự push branch — only if user pre-authorized push).

---

## Skip conditions

- Local stack thiếu Kafka Connect → M5.6, M5.7 partial (verify UI + BE response). Defer Kafka Connect verify khi infra available.
- Local Mongo standalone → skip TC-I-02 replicaSet test.
- FE chưa có Jest setup → skip M3.4 + M4 unit test, dựa M5/M6 smoke.
- i18n chưa setup → hardcode VN (CLAUDE §0).

## Escalation

- Stuck > 3 lần ở 1 task → APPEND "STUCK at T#.#, escalate Brain re-plan", chờ user.
- Phát hiện gap mới (vd worker mongo_introspection cần thêm method) → APPEND "Open items" trong report, KHÔNG tự expand.

## Code change guard

- TUYỆT ĐỐI: trước Edit, re-read CLAUDE.md §12 + §8.
- TUYỆT ĐỐI: KHÔNG cheat DB, KHÔNG hack k8s config.
- TUYỆT ĐỐI: KHÔNG báo DONE nếu M5 + M6 + M7 chưa thực sự PASS trên environment.
- L-3070: verify END-TO-END call chain trên environment thực, KHÔNG isolated test.
- L-3275: TUYỆT ĐỐI không log raw URI có password.
