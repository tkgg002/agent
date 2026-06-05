# Report — Default Collections Phase (`default_collections`)

> **Status**: 📋 **TEMPLATE** — Muscle fill các ô `<...>` sau khi thực thi xong.
> **Date**: 2026-05-25
> **Workspace**: `feature-connector-default-collections-2026-05-25`
> **Phase**: `default_collections`
> **Strategy**: Phương án A (xem `04_decisions_default_collections.md` ADR-001).

---

## 1. Executive summary

**Vấn đề**: Khi user tạo Mongo connector qua UI mà để trống field `Collections`, UX không rõ ràng → user không biết hành vi mặc định.

**Audit finding**: Pipeline thực tế đã hoạt động đúng (FE drop empty → BE pass-through → Debezium default CDC all). Gap duy nhất là UX hint + list view display.

**Giải pháp**: FE-only thêm `extra` text trong Form.Item Collections + render fallback `(All collections)` trong list view. KHÔNG đụng BE / Debezium / DB.

**Trạng thái**: <DONE / IN-PROGRESS / BLOCKED>

**Effort thực tế**: <X giờ> (planned: 2h + 30% buffer = 2h40m).

---

## 2. Files thay đổi

| File | Lines | Type | Diff summary |
|---|---|---|---|
| `data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx` | <N> lines | Edit | Thêm `extra` prop vào Form.Item Collections + update placeholder |
| `<list_view_file_path>` | <N> lines | Edit | Wrap render với fallback `(All collections)` |
| `data-hub/cdc-cms-web/src/locales/vi.json` (nếu có i18n) | <N> lines | Edit | Thêm key `connector.form.collections.extra` + `connector.list.collections.all` |
| `data-hub/cdc-cms-web/src/locales/en.json` (nếu có i18n) | <N> lines | Edit | Tương tự VN, English |

**Commit hash**: <hash>

---

## 3. Verify gates — kết quả thực tế

### Gate 0: Pre-flight audit (M0)
```
$ # T0.4 API test
$ curl -X POST http://localhost:<port>/api/system-connectors -d '<body>'
Status: <code>

$ # T0.5 Kafkacat verify
$ kafkacat -b localhost:9092 -t cdc.<server>.<db>.brand_new -C -e | head
<output>
```
Hypothesis confirmed: <YES/NO>
Debezium connector class: <FQDN>
Version: <X.Y.Z>
Status: <PASS / FAIL>

### Gate 1: Build (M3.2)
```
$ cd data-hub/cdc-cms-web && pnpm build
<output snippet>
Exit code: <0>
```
Status: <PASS>

### Gate 2: Lint (M3.3)
```
$ pnpm lint
Exit code: <0>
```
Status: <PASS>

### Gate 3: Typecheck (M3.4)
```
$ pnpm tsc --noEmit
Exit code: <0>
```
Status: <PASS>

### Gate 4: Smoke E2E create empty (TC-E-01, TC-E-02)
- Connector created: <YES/NO>
- Kafka Connect config: KHÔNG có `collection.include.list` → <CONFIRMED/FAILED>
- Mongo insert collection mới `brand_new_test_coll` → topic event: <PRESENT/ABSENT>
- List view hiển thị `(All collections)`: <YES/NO>
- Screenshots: `/tmp/default_collections_screenshots/`
Status: <PASS>

### Gate 5: Smoke backward compat (TC-E-03)
- Connector explicit `users,orders` → Kafka Connect config có key: <YES/NO>
- Mongo insert `brand_new_test_coll` → topic event KHÔNG xuất hiện cho connector này: <CONFIRMED/REGRESSION>
Status: <PASS>

### Gate 6: Edge case update (TC-E-06, TC-E-07)
- Edit empty → explicit → update OK: <YES/NO>
- Edit explicit → empty → key removed: <YES/NO>
Status: <PASS>

### Gate 7: Security review (M5)
```
$ /security-agent <files>
<output>
```
Findings: <none / list with severity>
Status: <PASS>

### Gate 8: Pre-existing regression check
```
$ pnpm test (nếu có test suite)
<output>
```
New failures: <none / list>
Status: <PASS>

---

## 4. Behavior changes (đối với operator / dev)

| Trước | Sau |
|---|---|
| Form Collections field chỉ có placeholder `users,orders,payments` | Form Collections field có hint text giải thích "Để trống = CDC tất cả" |
| User để trống → connector tạo thành công nhưng UI không feedback rõ behavior | User để trống → connector tạo + list view hiển thị `(All collections)` italic gray |
| List view không phân biệt "chưa cấu hình" vs "intentional all" | Phân biệt rõ qua visual style |
| BE handler logic | KHÔNG đổi |
| Debezium runtime | KHÔNG đổi (vẫn CDC all khi empty, vẫn filter khi explicit) |

---

## 5. Screenshots

### Form Create (with hint visible)
<![path or inline link])>

### List view — Connector empty Collections
<![path or inline link])>

### List view — Connector explicit Collections
<![path or inline link])>

---

## 6. Rollback plan

**Code rollback**: `git revert <commit_hash>` trên `data-hub/cdc-cms-web/`.

**No DB rollback** — không có migration trong phase này.

**No infra rollback** — không đụng Kafka Connect / Debezium config.

**Behavior fall về cũ**: placeholder + không có hint. Runtime vẫn đúng (Debezium default CDC all).

---

## 7. Lessons learned

(Sau khi done, APPEND vào `agent/memory/global/lessons.md` Global Pattern:)

**Pattern**: `[Khi user-reported gap nằm ở UX, BUT runtime pipeline X đã đúng (verified end-to-end), nên fix CHỈ ở layer UI L thay vì refactor pipeline]` → Yêu cầu:
1. Audit toàn bộ pipeline trước khi propose fix — verify runtime behavior thực tế.
2. Nếu runtime đã đúng → minimal-impact fix ở UI layer (hint, tooltip, display fallback).
3. KHÔNG over-engineer bằng cách inject explicit default ở BE — tăng surface area, tạo risk regression.
4. Document implicit framework default dependency (vd Debezium version) trong report để future readers track.

**Pattern cụ thể cho CDC connector**: `[Khi field config C nằm trong map M gửi đến framework F, drop key tự nhiên khi user empty là 1 implicit contract — KHÔNG validate "required" ở UI nếu framework F handle default đúng]` → Đúng: hint text + visual fallback display. Sai: thêm `rules: required` hoặc inject `wildcard` ở BE.

**Tags**: #ux-first #verify-pipeline-before-fix #minimal-impact #framework-default #debezium #cdc-connector #global-pattern

---

## 8. Open items / Defer

| Item | Lý do | Phase đề xuất |
|---|---|---|
| UI multi-select collection picker | Cần BE endpoint `GET /mongo/collections` | future phase `connector-collection-picker` |
| Validate format `db.collection,...` chuẩn Debezium | Out of scope | future phase `connector-filter-validate` |
| Document default behavior vào README/handbook | Cosmetic | future phase `docs-cms-handbook` |
| Apply pattern `(All X)` cho field `database.include.list` | Repeat pattern, not urgent | future phase `connector-default-display-unified` |
| Centralize wording sang shared package nếu multi-repo có FE khác | Cross-repo governance | future phase |

---

## 9. References

- `00_context_default_collections.md` — bối cảnh.
- `01_requirements_default_collections.md` — yêu cầu chi tiết.
- `02_plan_default_collections.md` — milestones.
- `03_implementation_default_collections.md` — technical design.
- `04_decisions_default_collections.md` — ADR-001..005.
- `06_test_cases_default_collections.md` — test matrix.
- `08_tasks_default_collections.md` — task checklist.
- `09_tasks_solution_default_collections.md` — code demo chi tiết.
- `10_gap_analysis_default_collections.md` — gap matrix.
- `05_progress.md` — audit log (append-only).

---

## 10. Pre-flight check (§14)

- [ ] Tất cả file workspace tồn tại vật lý (verify `ls -la`).
- [ ] `05_progress.md` APPEND đầy đủ entry cho M0..M6.
- [ ] `agent/memory/global/active_plans.md` workspace row updated → DONE.
- [ ] `agent/memory/global/lessons.md` APPENDED pattern mới (nếu có).
- [ ] KHÔNG có "shadow file" thảo luận trong chat mà thiếu file vật lý.
- [ ] Tất cả ô `<...>` trong report đã fill.
- [ ] Screenshots saved + referenced trong report.
- [ ] Log files trong `/tmp/default_collections_*.log` đã collect.

---

> **Note Muscle**: Report này KHÔNG được claim "Đã xong" nếu Gate 4 (smoke CDC) chưa PASS thực tế. Nếu KC infra down → mark IN-PROGRESS + note Open items "smoke CDC pending infra", KHÔNG fake PASS. CLAUDE.md §3 Verify before Done.
