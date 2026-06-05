# 02_plan_check_connection — Milestone Roadmap

> **Phase**: `check_connection`
> **Execution order**: TUẦN TỰ M0 → M8
> **Effort estimate**: 6h work + 30% buffer ≈ 8h
> **Risk level**: MEDIUM (cross-service change: FE + cdc-cms + worker)

---

## Strategy decision (xem `04_decisions_check_connection.md`)

- **ADR-001** (Cross-DB scope): P0 = Mongo-only. MySQL/PG defer.
- **ADR-002** (BE DTO shape): Extend handler accept `uri` field via POST body (giữ GET host+port backward compat).
- **ADR-003** (Error UX): Map 5-case `IntrospectDiagnosis` → user-friendly Vietnamese message.
- **ADR-004** (Multi-select default): Auto-select-all on PASS.
- **ADR-005** (Progress UX): `<Spin>` indeterminate + step label, KHÔNG `<Progress percent>` (vì không có real progress).
- **ADR-006** (Gate Create): Disable Create until check PASS + invalidate khi URI/DB đổi.

---

## M0 — Pre-flight audit verify

> Mục đích: confirm hypothesis trước khi code. Nếu gãy → re-plan ngay (CLAUDE §3).

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T0.1 | Read `centralized-data-service/internal/service/mongo_introspection.go:1-200` xác nhận signature `DiscoverDatabases(uri)` + `DiscoverCollections(uri, db)` + `IntrospectDiagnosis` struct | Muscle | Quote exact lines vào `05_progress.md` |
| T0.2 | Read `centralized-data-service/internal/handler/command_handler.go:1162-1230` xác nhận `HandleDiscoverMongoDatabases/Collections` DTO + URI build logic | Muscle | Quote exact |
| T0.3 | Read `cdc-cms-service/internal/api/introspection_handler.go:1-150` xác nhận HTTP DTO + NATS request flow | Muscle | Quote exact |
| T0.4 | Read `cdc-cms-web/src/pages/SourceConnectors.tsx:107` (regex) + `:286-295` (state) + `:878-1026` (modal) + `:148-220` (`buildConnectorConfig`) | Muscle | Quote exact |
| T0.5 | Test existing endpoint với local stack: `curl 'http://localhost:<port>/api/introspection/mongo/databases?host=localhost&port=27017'` → confirm 200 với danh sách | Muscle | Output JSON |
| T0.6 | Local Mongo có replica set hay standalone? Note để tests M4 chọn đúng URI format | Muscle | Note vào progress |
| T0.7 | APPEND `05_progress.md` "M0 done — audit verified, signature confirmed, endpoint live, mongo topology = <X>" | Muscle | Entry exists |

**Exit gate**: Nếu T0.5 fail (endpoint không tồn tại / 404) → STOP, audit phải lại. KHÔNG proceed M1.

---

## M1 — Worker side: extend handler accept full URI

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T1.1 | Edit `command_handler.go:1164-1180` `HandleDiscoverMongoDatabases`: extend DTO thêm field `Uri string \`json:"uri"\``. Logic: nếu `req.Uri != ""` → dùng trực tiếp; else fallback build `mongodb://host:port`. Xem `09_tasks_solution` Edit #1. | Muscle | Diff applied |
| T1.2 | Edit `command_handler.go:1204-1224` `HandleDiscoverMongoCollections`: tương tự T1.1 | Muscle | Diff applied |
| T1.3 | `cd centralized-data-service && go build ./... && go vet ./...` | Muscle | Exit 0 |
| T1.4 | Unit test mới: `command_handler_test.go` (HOẶC file mới) — test 4 case: (a) chỉ uri, (b) chỉ host+port, (c) cả 2 (uri ưu tiên), (d) cả 2 đều rỗng → error | Muscle | All PASS |
| T1.5 | APPEND `05_progress.md` "M1 done — worker handlers extended" | Muscle | Entry exists |

---

## M2 — BE relay: cdc-cms-service introspection_handler

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T2.1 | Edit `cdc-cms-service/internal/api/introspection_handler.go:25-75` `DiscoverMongoDatabases`: chấp nhận thêm POST với body `{"uri":"...","host":"...","port":"..."}` (HOẶC extend GET với query param `uri`). Xem `09_tasks_solution` Edit #3. Khuyến nghị POST để tránh URI có credentials xuất hiện trong URL log nginx/access log. | Muscle | Diff applied |
| T2.2 | Edit `introspection_handler.go:77-150` `DiscoverMongoCollections`: tương tự T2.1 với param `database` chuyển từ path sang body OR giữ path | Muscle | Diff applied |
| T2.3 | Edit `router.go:331-332` thêm POST route (nếu chọn POST mới): `POST /api/introspection/mongo/databases` + `POST /api/introspection/mongo/collections`. Giữ GET cũ cho backward compat (R12). | Muscle | Diff applied |
| T2.4 | `cd cdc-cms-service && go build ./... && go vet ./... && go test ./internal/api/...` | Muscle | Exit 0, no regression |
| T2.5 | Test integration với worker (cần stack up): curl POST với uri thật → response có collections | Muscle | JSON OK |
| T2.6 | APPEND `05_progress.md` "M2 done — BE relay extended, POST endpoint live" | Muscle | Entry exists |

---

## M3 — FE service + hook

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T3.1 | Edit `cdc-cms-web/src/services/api.ts` (HOẶC file mới `src/services/connectorCheck.ts`): thêm `checkMongoDatabases({uri})` + `checkMongoCollections({uri, database})`. Return shape: `{ collections: string[], status: 'ok'|'cluster_err'|'db_missing'|'empty', error?: string, availableDbs?: string[] }`. Xem `09_tasks_solution` Edit #5. | Muscle | Service exports |
| T3.2 | Create `cdc-cms-web/src/hooks/useConnectorCheck.ts`: React Query mutation hook wrap `checkMongoCollections`. Trả state: `{ status, collections, error, isPending, mutate, reset }`. Xem `09_tasks_solution` Edit #6. | Muscle | Hook exported |
| T3.3 | `cd cdc-cms-web && pnpm tsc --noEmit` | Muscle | Exit 0 |
| T3.4 | APPEND `05_progress.md` "M3 done — FE service+hook" | Muscle | Entry exists |

---

## M4 — FE UI integration

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T4.1 | Edit `SourceConnectors.tsx`: extend state thêm `checkResult: CheckResult \| null` + `checkPending: boolean`. Watch URI + database value change → reset checkResult về null + reset collectionNames | Muscle | State logic compile |
| T4.2 | Edit form modal: thêm `<Button onClick={handleCheck} loading={checkPending}>Check Connection</Button>` đặt sau field Database. Xem `09_tasks_solution` Edit #7. | Muscle | Button visible |
| T4.3 | Edit form modal: replace `<Input placeholder="users,orders,payments" />` field Collections bằng `<Select mode="multiple" options={checkResult?.collections.map(c => ({value: c, label: c}))} disabled={!checkResult?.collections} />`. Auto-select-all logic ở handler `handleCheck` success (set form value). Xem Edit #8. | Muscle | Multi-select renders với data |
| T4.4 | Edit Modal footer: `okButtonProps={{ disabled: !checkResult || checkResult.status !== 'ok' }}` để gate Create button. Xem Edit #9. | Muscle | Create button toggles correct |
| T4.5 | Edit error display: nếu `checkResult.status !== 'ok'` → show Antd `<Alert type="error" message={mapStatusToVi(checkResult)}>` với danh sách `availableDbs` nếu có. Xem Edit #10. | Muscle | Error visible |
| T4.6 | Watch URI change → `useEffect` gọi `setCheckResult(null)` + `form.setFieldValue('collectionNames', undefined)`. Same cho database. | Muscle | State invalidates on change |
| T4.7 | `pnpm tsc --noEmit && pnpm lint && pnpm build` | Muscle | Exit 0 |
| T4.8 | APPEND `05_progress.md` "M4 done — FE UI integrated, build PASS" | Muscle | Entry exists |

---

## M5 — Smoke test E2E happy path

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T5.1 | Spin local stack: docker compose / k8s, verify worker + cdc-cms + FE + Mongo + Kafka Connect UP | Muscle | All healthy |
| T5.2 | Open FE Create Connector modal → kiểu MongoDB → nhập URI `mongodb://localhost:27017` + Database `goopay_pbs` → click Check | Muscle | Spinning, then result |
| T5.3 | Verify multi-select hiển thị các collection thực tế của DB | Muscle | Match `mongosh "..." --eval 'db.getCollectionNames()'` |
| T5.4 | Verify Create button enabled. Click Create. | Muscle | Connector created 200 OK |
| T5.5 | Verify Kafka Connect: `curl http://<connect>:8083/connectors/<name>/config \| jq '.["collection.include.list"]'` → có explicit list = các collection được chọn | Muscle | jq match expected |
| T5.6 | Variant: uncheck 1-2 collection trước Create → verify config chỉ có collection được chọn | Muscle | jq match |
| T5.7 | Capture screenshot: form, check progress, result, multi-select, success toast | Muscle | Files trong `/tmp/check_connection_screens/` |
| T5.8 | APPEND `05_progress.md` "M5 done — happy path PASS, evidence: <paths>" | Muscle | Entry exists |

---

## M6 — Smoke test E2E negative cases

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T6.1 | TC `cluster_err`: URI `mongodb://localhost:9999` (port sai) → Check → expect error "Không kết nối được..." + Create disabled | Muscle | Match A6 |
| T6.2 | TC `db_missing`: URI đúng + Database `nonexistent_db_xyz` → Check → expect error "Database `nonexistent_db_xyz` không tồn tại. Có sẵn: <list>" | Muscle | Match A7 |
| T6.3 | TC `empty`: Tạo DB rỗng `db_empty_test` trên Mongo → Check → expect error "Database `db_empty_test` chưa có collection nào" | Muscle | Match A8 |
| T6.4 | TC `auth_err` (sub-case của cluster_err): URI có wrong credentials `mongodb://wrong:wrong@localhost:27017/?authSource=admin` → expect error đề cập auth | Muscle | Sanitized error |
| T6.5 | TC backward compat: legacy GET caller `curl 'http://<be>/api/introspection/mongo/databases?host=localhost&port=27017'` → 200 OK | Muscle | A9 PASS |
| T6.6 | TC state invalidate: PASS rồi đổi URI → multi-select clear + Create disable | Muscle | A8 visual |
| T6.7 | APPEND `05_progress.md` "M6 done — negative TCs PASS" + logs | Muscle | Entry exists |

---

## M7 — Security review

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T7.1 | Run `/security-agent` trên 3 service file changed list | Muscle | Output saved |
| T7.2 | Manual grep log files của worker + cdc-cms: `grep -E 'mongodb://[^/]+:[^@]+@' /tmp/worker.log` → expect 0 hit (password đã sanitize) | Muscle | 0 hits |
| T7.3 | Manual verify error response payload không chứa raw URI | Muscle | response inspect |
| T7.4 | Verify rate-limit / abuse: spam click Check 10 lần trong 5s → button disable / debounce kick in | Muscle | N8 PASS |
| T7.5 | APPEND `05_progress.md` "M7 done — security clean / N findings: <list>" | Muscle | Entry exists |

---

## M8 — Report + memory update + supersede old workspace

| Task | Action | Owner | Exit criteria |
|---|---|---|---|
| T8.1 | Fill `report_check_connection_2026-05-26.md` với evidence thực tế | Muscle | Report complete |
| T8.2 | APPEND `agent/memory/global/lessons.md` Global Pattern: `[Khi feature F yêu cầu pre-validate connection trước tạo entity E, kiến trúc tốt = sync probe endpoint với 5-case diagnosis (cluster/namespace/entity/empty/data-shape), trả về candidate list cho UI multi-select; KHÔNG dùng wizard session table cho ephemeral state]` | Muscle | Lesson abstracted |
| T8.3 | Update `active_plans.md`: workspace `feature-connector-check-connection-2026-05-26` → DONE. Workspace `feature-connector-default-collections-2026-05-25` → SUPERSEDED (by check-connection). | Muscle | active_plans updated |
| T8.4 | APPEND `05_progress.md` "M8 done — phase check_connection COMPLETE" | Muscle | Entry exists |
| T8.5 | Pre-flight §14: verify all 11 .md files trong workspace tồn tại vật lý. Cross-check report deliverables. | Muscle | Pre-flight passed |
| T8.6 | Report user verb hoàn thành. KHÔNG tự push remote nếu user chưa yêu cầu. | Muscle | User informed |

---

## Decision tree (nếu lệch khỏi plan)

```
M0 fail (endpoint không tồn tại)
  → Audit lại router.go + introspection_handler.go
  → Có thể đã refactor sau audit subagent → STOP, re-plan

M1 fail (worker build fail)
  → Đọc error → fix → re-run
  → Stuck 3 lần → STOP, escalate

M2 fail (BE build / route conflict)
  → Có thể path POST conflict với existing → đổi prefix `/v2/introspection/...`
  → Update FE service tương ứng

M3 fail (TS compile)
  → React Query v5 API: dùng `mutationFn` chuẩn, không `mutate` direct
  → Verify type signatures

M4 UI fail (multi-select không render)
  → Verify Antd v6 import: `import { Select } from 'antd'`
  → Verify `options` prop shape `{value, label}[]`

M5 smoke fail (CDC config sai)
  → Verify `buildConnectorConfig` line 168 — collection.include.list build từ collectionNames
  → Nếu format `db.col1,db.col2` (Debezium spec) thì FE phải prepend db name

M6 backward compat fail
  → KHÔNG xóa GET route cũ ở M2
  → Add integration test

M7 security HIGH
  → Fix root cause, KHÔNG skip
  → Re-review

M8 lesson không abstract
  → Re-write theo §13 (dùng biến A/B/X/Y)
```

## Effort breakdown

| Milestone | Estimate |
|---|---|
| M0 audit | 30m |
| M1 worker | 60m |
| M2 BE relay | 60m |
| M3 FE service+hook | 45m |
| M4 FE UI | 90m |
| M5 happy smoke | 45m |
| M6 negative smoke | 45m |
| M7 security | 30m |
| M8 report+memory | 45m |
| **Total** | **6h30m** |

## Skip conditions

- Nếu local stack không có Kafka Connect: M5.5 partial — chỉ verify FE → BE → worker happy path. Defer Kafka Connect verify khi infra available. Note trong report.
- Nếu Mongo local là standalone (không replicaSet): test URI format `mongodb://localhost:27017` đủ. Skip test `mongodb+srv://` (cần Atlas hoặc local SRV mock).
- Nếu i18n setup phức tạp: hardcode tiếng Việt, defer i18n keys sang follow-up.

## Escalation

- Stuck > 3 lần ở 1 task → APPEND `05_progress.md` "STUCK at T#.#, escalate Brain re-plan", chờ user verb.
- Phát hiện gap ngoài scope (vd: cần refactor schema_adapter để support new field) → KHÔNG tự ý expand, APPEND "Open items" trong report.

## Code change guard (CRITICAL)

- TUYỆT ĐỐI: trước khi edit code → re-read CLAUDE.md §12. Actor PHẢI là Muscle.
- TUYỆT ĐỐI: KHÔNG cheat DB / KHÔNG hack config / KHÔNG bypass validation.
- TUYỆT ĐỐI: KHÔNG báo done nếu M5+M6+M7 chưa PASS thực tế trên environment.
- L-3070: verify END-TO-END call chain trên environment thực, KHÔNG chỉ test isolated.
