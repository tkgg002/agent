# 01 — Requirements Phase E: Close-loop fix for residual gaps

**Date**: 2026-05-04 (16:45+07)
**Trigger**: User mandate "lên plan fix nó" sau khi phase `core_flow_hardening_p0_p1` đóng nhưng để lại 9 gap.
**Scope**: 5 task — G5 prune (zero risk) → G4 schedule audit (diagnostic) → G7 multi-tier filter (CODE) → G2 Mongo addtest smoke (E1 validate) → G8 security gate.

---

## FR (Functional Requirements)

### FR-E1 — Multi-tier filter close-loop (G7) — **HIGH**
- Admin-api phải extend tier-cao filter (`database.include.list`/`db.include.list`) đồng thời với tier-thấp (`collection.include.list`/`table.include.list`).
- Idempotent: nếu namespace đã có trong tier-cao → no-op; nếu chưa có → append.
- Per-engine adapter:
  - MongoDB: `database.include.list` (top) + `collection.include.list` (low).
  - PostgreSQL: verify `database.dbname` match (single-DB only) + `schema.include.list`+`table.include.list`.
  - MySQL/MariaDB: `database.include.list` + `table.include.list`.
- Response trả thêm `warnings: [...]` khi vừa add namespace mới ở tier-cao (Debezium cần restart task để snapshot từ namespace mới).

### FR-E2 — Smoke E2E cho Mongo `payment_bills_addtest` (G2)
- Sau khi E1 land, gọi admin-api PUT `payment_bills_addtest` (database = `payment-bill-service` đã có trong tier-cao).
- INSERT 1 doc mới vào source.
- Wait 30s — verify shadow row landed; wait 60s — verify master row landed.

### FR-E3 — Prune V1 legacy seeds (G5)
- Chạy `deployments/sql/cdc/prune_legacy_v1_bindings.sql` (đã tồn tại từ session cũ).
- Idempotent — re-run lần 2 không thay đổi gì.
- Verify count(*) `legacy_*` `is_active=true` → 0.

### FR-E4 — Schedule audit cho `orders_addtest` (G4)
- Diagnose: shadow `src_local_pg_source.orders_addtest` có 11 rows nhưng master = 0 → schedule không enable, hoặc binding không active, hoặc transmute fail.
- Output: 1 file `report_g4_diag_*.md` chỉ rõ root cause + remediation step (nếu cần). KHÔNG auto-fix vì có thể là intentional state.

### FR-E5 — Security gate (G8) — CLAUDE.md §8 mandatory
- Chạy `/security-agent` review trên `cmd/admin-api/`, `internal/admin/`.
- Output: list mitigation cần land, không tự động land code change trong phase này (nếu có fix nhỏ thì OK, ngược lại defer Phase F).

---

## NFR (Non-Functional)

- **NFR-1**: Worker không restart trong toàn bộ phase (validation phụ).
- **NFR-2**: Mọi code change trong E1 phải có unit test ≥1 case happy + ≥1 case idempotent no-op.
- **NFR-3**: Brain prohibition §12 — Brain chỉ Plan + Doc + Verify; Muscle thực thi code.
- **NFR-4**: Memory APPEND-only (§11): tuyệt đối không overwrite `05_progress.md` hoặc `lessons.md`.
- **NFR-5**: Mỗi fix có 1 entry APPEND `05_progress.md` riêng + 1 file `report_*.md` cuối phase.

---

## AC (Acceptance Criteria)

| Task | Acceptance |
|---|---|
| E3 (G5 prune) | `count(*) source_object_registry WHERE object_code LIKE 'legacy\_%' ESCAPE '\' AND is_active=true` = 0; re-run script lần 2 không change row nào |
| E4 (G4 audit) | Có file `report_g4_diag_<ts>.md` xác định root cause + recommendation |
| E1 (G7 code) | `extendDebeziumInclude` xử lý 2-tier; 3 unit test PASS; live: PUT collection mới ở namespace mới qua admin-api → connector config update both `database.include.list` + `collection.include.list` |
| E2 (G2 smoke) | Mongo `payment_bills_addtest` INSERT doc → shadow row landed trong 30s → master row landed trong 90s |
| E5 (G8 sec) | `/security-agent` chạy xong, output list mitigation; không có blocker severity HIGH bỏ ngỏ không có ticket |

---

## Risks

| Risk | Mitigation |
|---|---|
| R1: PUT `database.include.list` trên Debezium connector trigger task restart → mất offset → re-snapshot lại tất cả namespace cũ → spam Kafka | Test trong staging trước; Debezium MongoDB connector restart chỉ re-snapshot namespace MỚI thêm vào (incremental). Verify log `gpay-kafka-connect`. |
| R2: PG `database.dbname` single-DB constraint → nếu user gọi admin-api với engine=PG database khác → reject 400 | E1 helper return error rõ ràng "PG connector locked to single database X, cannot register Y" |
| R3: Prune script chạy lần 2 nhưng đã có row mới với prefix `legacy_*` → false positive | Script idempotent với `WHERE is_active=true` — chỉ deactivate row đang active. Re-run lần 2 không match → 0 row affected. |
| R4: G4 diagnose phát hiện cần code fix → blow up scope | Defer: chỉ ghi report + recommendation, không fix trong phase E |
| R5: /security-agent phát hiện HIGH issue → block deploy | OK — đó là mục đích của gate. Issue HIGH → Phase F1 hardening tách riêng |

---

## Dependencies

- `cmd/admin-api/main.go` đang chạy (Up >5h) — KHÔNG restart trong phase
- `gpay-cdc-worker` Up 31m — KHÔNG restart trong phase
- `gpay-kafka-connect` Up 5h — restart task được chấp nhận khi PUT config (idempotent)
- File SQL `deployments/sql/cdc/prune_legacy_v1_bindings.sql` — đã tồn tại từ session 2026-05-04 trưa

---

## Out of Scope

- D1 Schema Schism (V1 vs V2 PK convention) — long-term, đợi architect.
- G6 orphan shadow tables (`orders_e2e_d_v2/v3/v4`) — defer, low priority.
- G6.1 connector mapping env-based — defer, multi-tenant không cần ngay.
- B3 PG `orders_addtest` include — defer (đã có 11 rows ingested qua path khác, cần audit riêng).

---

## Skills (CLAUDE.md §0)

Read, Write, Edit (APPEND-only), Bash, Agent (Muscle delegate), `/security-agent`, lessons applied (L-three-layer-trust, L-real-data-test, L-runtime-state-verify, L-multi-tier-filter-mirror, L-multi-engine-2, L-cascade-liability).
