# 10_gap_analysis — Cleanup `_gpay_source_id` + `_gpay_deleted`

## GAP-1 — Path A preview handler DRIFT
- **Symptom**: `cdc-cms-service/internal/api/mapping_preview_handler.go:63-69` đọc `_gpay_id`, `_gpay_source_id` từ shadow tables Path A.
- **Reality**: `cdc-cms-service/internal/infra/persistence/shadow_automator.go:78-90` tạo shadow tables với `id BIGINT PK`, `source_id VARCHAR(200)`, KHÔNG có `_gpay_*`.
- **Impact**: Preview API broken trên Path A shadow tables (DB error "column _gpay_source_id does not exist").
- **Bug #2 fix yesterday accidental masking**: Path B (`command_handler.go` CREATE) thêm `_gpay_source_id` → preview "work" được khi user tạo shadow qua flow command_handler, nhưng vẫn broken cho flow shadow_automator.
- **Resolution**:
  - Nếu chọn Option A: GAP-1 không address; mở issue tracker riêng.
  - Nếu chọn Option B: GAP-1 fix luôn (patch B.2).
  - Nếu chọn Option C: GAP-1 fix luôn.

## GAP-2 — Path C V2 master naming convention dual-path
- **Symptom**: V2 master + sinkworker + transmuter dùng `_gpay_source_id`/`_gpay_deleted` exclusively; FE shadow path-A dùng `source_id`/`_deleted`. 2 naming convention song song trong cùng codebase.
- **Impact**: Confusion cho dev mới; future code có thể chọn sai convention.
- **Resolution**:
  - Option A: KHÔNG address (V2 master giữ nguyên).
  - Option B: KHÔNG address (chỉ unify FE shadow).
  - Option C: Address (REJECTED phase này — yêu cầu workspace riêng `refactor-master-naming-2026-06-XX`).

## GAP-3 — Test fixtures + UI placeholder
- **Symptom**: 
  - `centralized-data-service/test/internal/service/schema_adapter_ordering_test.go` dùng `_gpay_source_id` làm PK test fixture.
  - `centralized-data-service/test/internal/service/schema_adapter_test.go` test V2 schema branch.
  - `cdc-cms-web/src/pages/MasterRegistry.tsx:68,425` placeholder spec `'{"pk":"_gpay_source_id"}'`.
- **Impact**: 
  - Test fixtures tied to V2 master schema → Option C breaking.
  - FE placeholder hint user nhập `_gpay_source_id` → confuse khi backend đổi.
- **Resolution**:
  - Option A: KHÔNG address (test + UI giữ nguyên).
  - Option B: Update FE placeholder (patch B.4). Test fixtures giữ vì V2 master không đổi.
  - Option C: Address full (REJECTED).

## GAP-4 — Bug #2 e2e test reference
- **Symptom**: `cdc-cms-service/internal/app/commands/approve_schema_proposal_e2e_test.go:100` dùng `_gpay_source_id` trong setup CREATE TABLE expectation.
- **Impact**: Option A revert làm test này vẫn pass (test setup tạo bảng riêng với `_gpay_source_id`). Option B sửa preview handler → test setup phải update để dùng `source_id`.
- **Resolution**:
  - Option A: Không sửa.
  - Option B: Phải sửa setup test dùng `source_id` (nhưng cần verify destinationsemantics).

## GAP-5 — `04_decisions.md` deferred Option C 
- **Symptom**: Option C bị defer. Naming convention dual-path không được giải quyết.
- **Action plan**: Sau khi Option A/B apply → đánh giá lại nhu cầu unify master. Nếu prod stable + có 1 sprint capacity → kick off workspace `refactor-master-naming-2026-06-XX`.

## Action items cho từng option (nếu user verb)

| Option | GAP-1 | GAP-2 | GAP-3 | GAP-4 | GAP-5 |
|---|---|---|---|---|---|
| A | OPEN | OPEN | OPEN | N/A | OPEN |
| B | FIXED | OPEN | PARTIAL (FE only) | FIXED | OPEN |
| C | FIXED | FIXED | FIXED | FIXED | CLOSED |

---

## REVISED 2026-05-28 — Unified RENAME plan
Option A/B/C retire. Plan mới = single unified RENAME (`09_tasks_solution_cleanup.md`).

| GAP | Status sau RENAME plan |
|---|---|
| GAP-1 Path A preview drift | FIXED (patch A.1 đổi SELECT `_gpay_*` → `id`/`source_id`) |
| GAP-2 V2 master dual naming | FIXED (rename `_gpay_source_id` → `source_id`, `_gpay_deleted` → `_deleted` cả Path C) |
| GAP-3 Test fixtures + UI placeholder | FIXED (patch C.5, D.1, D.2, D.3) |
| GAP-4 Bug #2 e2e test | FIXED (patch A.2) |
| GAP-5 Defer Option C | CLOSED (đã merge vào unified plan) |

## NEW GAP — Production data migration
- **GAP-6**: Production master/shadow tables đã có data. Migration SQL phải apply trước/đồng thời với code deploy.
- **Resolution**: `09_tasks_solution_cleanup.md` section "DB MIGRATION SQL" + "Deploy order" lay out 5 bước drain → migrate → deploy → restart → smoke.
- **Risk**: DROP `_gpay_deleted` Path B → mất giá trị soft-delete column riêng. Mitigation: Path B đã có `_deleted` chứa giá trị tương đương (cùng được set TRUE trong tombstone INSERT `event_handler.go:236`).
