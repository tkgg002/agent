# 01_requirements_check_connection — Yêu cầu

> **Phase**: `check_connection`
> **Source of truth**: User message ngày 2026-05-26.

---

## 1. User story

```
Là Ops admin sử dụng CMS,
Tôi muốn khi tạo Source Connector mới (Mongo, sau này MySQL/PG),
Sau khi nhập Connection URL + Database, bấm nút "Check Connection",
Hệ thống PHẢI kiểm tra kết nối, hiển thị progress,
Nếu OK → trả về danh sách collections của DB,
FE hiển thị multi-select với mặc định CHỌN HẾT,
Tôi có thể uncheck collection không cần CDC,
Nút "Create" CHỈ enable khi check đã pass,
Để tôi không tạo nhầm connector lỗi và không phải gõ tên collection bằng tay.
```

## 2. Functional requirements

| ID | Yêu cầu | Tiêu chí Done |
|---|---|---|
| **R1** | Form New Connector kiểu MongoDB PHẢI có nút **Check Connection** | Nút visible bên cạnh field Connection URL + Database |
| **R2** | Khi bấm Check, FE PHẢI gửi `{ uri, database }` tới BE (URI full, chứa `mongodb://` hoặc `mongodb+srv://`) | Network tab cho thấy POST/GET request có payload |
| **R3** | Trong khi đang check, FE PHẢI hiển thị loading state (Spin / Steps / indicator). Nút Create PHẢI disabled | UX visual feedback, nút Create greyed out |
| **R4** | Nếu BE trả PASS với danh sách collections (non-empty) → FE PHẢI render `<Select mode="multiple">` với options = danh sách, defaultValue = TẤT CẢ | Multi-select visible, tất cả collection initially selected, dropdown có search |
| **R5** | Nếu BE trả FAIL → FE PHẢI hiển thị error message phân biệt 5 case (xem ADR-003 + L-2026-05-19): `cluster_err`, `db_missing`, `coll_missing` (N/A cho UC này — chỉ list collections), `empty` (DB không có collection nào), `no_fields` (N/A). Mỗi case map sang message tiếng Việt friendly + actionable | Error visible trên UI, user biết phải fix gì |
| **R6** | Khi error `db_missing` → message hiển thị danh sách database AVAILABLE (top 50) để user chọn lại | Tooltip / dropdown với danh sách DB |
| **R7** | Nút **Create** DISABLED ban đầu (trước khi check) HOẶC sau khi check FAIL. ENABLED chỉ khi check PASS | Disable/Enable state logic chính xác |
| **R8** | Nếu user đổi URI hoặc Database sau khi check PASS → FE PHẢI invalidate check state (Create lại disable, multi-select reset / clear) | Form value watch trigger reset |
| **R9** | Submit form Create → payload bao gồm `collection.include.list` = `db.col1,db.col2,...` theo selection (logic `buildConnectorConfig` line 168 đã có, KHÔNG đụng) | Network payload đúng format |
| **R10** | Nếu user check OK rồi giữ nguyên multi-select default (tất cả collection) → submit RESULTING `collection.include.list` = tất cả collection name explicit (KHÔNG drop về empty) | Verify Kafka Connect config sau create |
| **R11** | URI sanitize trước khi log/error (không leak `user:pass@`) | Backend log grep không có raw password |
| **R12** | Backward compat: behavior cũ với `{host, port}` payload PHẢI tiếp tục work (cho external/automation caller hiện có) | Existing tests pass |

## 3. Non-functional requirements

| ID | Yêu cầu |
|---|---|
| **N1** | Check Connection sync timeout ≤ 10s (đã match existing NATS request-reply timeout) |
| **N2** | KHÔNG cheat DB, KHÔNG patch k8s config |
| **N3** | Build pass FE (`pnpm build`) + BE (`go build ./...`) + Worker (`go build ./...`) |
| **N4** | Lint / vet / tsc pass |
| **N5** | A11y: nút Check có ARIA label, multi-select có ARIA, error message có ARIA `role="alert"` |
| **N6** | i18n: nếu có setup → dùng key; nếu chưa → hardcode tiếng Việt (CLAUDE.md §0) |
| **N7** | Security: URI có password KHÔNG đi vào response payload, không log raw. Sanitize bằng helper đã có. |
| **N8** | Concurrent click Check: debounce hoặc disable button khi đang pending — tránh spam NATS |
| **N9** | Performance: check 1 DB ≤ 5s p95 với DB có ≤ 500 collections (driver `ListCollectionNames`) |

## 4. Out of scope (defer)

| Item | Lý do | Phase đề xuất |
|---|---|---|
| MySQL Check Connection | Worker service chưa có | future phase `connector-check-mysql` |
| Postgres Check Connection | Worker service chưa có | future phase `connector-check-pg` |
| Cross-DB driver interface refactor | Cần refactor lớn | future phase `connector-driver-abstraction` |
| Wizard session persist (resume sau refresh) | UC này stateless, dùng React state là đủ | future phase nếu cần |
| Auto-select connector class theo URI scheme | Logic hiện có ổn, không trong scope | future phase |
| Real-time progress streaming (WebSocket / SSE) | 10s sync không phức tạp đến mức cần stream | future phase |
| List Mongo database trước (chọn DB từ dropdown thay vì gõ tay) | UC user yêu cầu là gõ DB rồi check; pattern này không bắt buộc | future phase enhancement |
| Save check result vào DB | KHÔNG cần (ephemeral) | — |

## 5. Acceptance criteria (Definition of Done)

Phase này được tính DONE khi **TẤT CẢ** các tiêu chí sau PASS:

- [ ] **A1**: BE handler `DiscoverMongoDatabases` + `DiscoverMongoCollections` accept request body có `uri` field (POST `{ "uri":"...", "database":"..." }` HOẶC giữ query params backward compat).
- [ ] **A2**: Worker `HandleDiscoverMongoDatabases/Collections` ưu tiên `uri` từ payload nếu có, fallback `host+port` cũ.
- [ ] **A3**: FE thêm hook `useCheckMongoConnection` + service function `checkMongoConnection({uri, database})`.
- [ ] **A4**: FE SourceConnectors.tsx form Mongo có nút Check + Spin + multi-select. Layout không vỡ horizontal/vertical.
- [ ] **A5**: Smoke E2E: open Create modal → fill valid URI + DB → click Check → progress visible → result multi-select hiện danh sách collections → Create button enabled → submit → connector created với explicit `collection.include.list`. Evidence: video / screenshot.
- [ ] **A6**: Smoke E2E negative `cluster_err`: gõ URI sai → check FAIL → message tiếng Việt "Không kết nối được tới Mongo: <sanitized error>". Create button DISABLED.
- [ ] **A7**: Smoke E2E negative `db_missing`: URI đúng + DB không tồn tại → message "Database <X> không tồn tại. Database có sẵn: <list>". Available DB list tooltip / dropdown visible.
- [ ] **A8**: Smoke E2E negative `empty`: URI + DB hợp lệ nhưng DB chưa có collection nào → message "Database <X> chưa có collection nào. Tạo collection rồi check lại."
- [ ] **A9**: Backward compat: existing caller dùng `?host=&port=` GET vẫn PASS (test cũ không regression).
- [ ] **A10**: Build / lint / typecheck / vet PASS toàn bộ 3 service.
- [ ] **A11**: `/security-agent` review PASS hoặc no HIGH/CRITICAL. Đặc biệt verify: KHÔNG có raw URI password trong log/response.
- [ ] **A12**: `report_check_connection_2026-05-26.md` filled với evidence thực tế, file changed list, screenshot, log snippets.
- [ ] **A13**: `05_progress.md` APPEND đầy đủ audit log mỗi milestone.
- [ ] **A14**: `active_plans.md` update workspace row → DONE.
- [ ] **A15**: Old workspace `feature-connector-default-collections-2026-05-25` được mark SUPERSEDED.

## 6. Risk matrix

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Worker `mongo_introspection.go` parse URI fail với edge format (`+srv`, `?replicaSet=...&authSource=admin&tls=true`) | Low | High | M0 test với 3 URI format thực tế: standalone, replica set, srv |
| BE relay NATS payload schema mismatch worker | Medium | High | M2 ràng buộc DTO shared / có integration test |
| Mongo URL có credentials → log leak | Medium | HIGH (security) | L-3275: dùng `SanitizeMongoDSN` ở mọi log/error path, gate qua `/security-agent` |
| FE state desync: user change URI nhưng multi-select vẫn giữ list cũ | Medium | Medium | R8 + form watch listener |
| Network 10s timeout không đủ cho DB lớn (>1000 collections) | Low | Medium | N9 fallback: nếu >500 collections, warning UI "DB lớn, kết quả có thể không hoàn chỉnh" |
| User bỏ qua check, hack DOM enable Create | Very Low | Low (BE validate vẫn chạy) | BE validate connector.class + config independent of FE state — implicit secure |
| Concurrent check spam NATS | Medium | Low | N8 debounce + disable button when pending |
| Backward-compat test cũ break | Low | Medium | A9 + giữ existing path query string |

## 7. Inverse requirements (CẤM)

| Cấm | Lý do |
|---|---|
| Cấm log raw URI với password | Security leak (L-3275) |
| Cấm save URI sau check (vd vào cache / DB không có encryption) | Out of scope + security |
| Cấm gọi Mongo trực tiếp từ cdc-cms-service (bypass worker) | Violates worker ownership pattern (L-CDC-golden-rule) |
| Cấm hardcode default timeout < 5s hoặc > 30s | Conflict UX |
| Cấm reuse path `/scan-fields` cho UC này | Khác semantic (scan-fields cần registered row) |
| Cấm sửa schema DB cho UC này | Stateless, không cần persist |
| Cấm wizard session table cho UC này | Overkill, dùng React state đủ |
