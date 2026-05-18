# 01 — Requirements: Migration System Refactor

**Phase**: `migration_refactor`
**Workspace**: `InvestigateTransformSchedulerStuck`
**Date**: 2026-05-14 15:32 (Asia/Ho_Chi_Minh)
**Owner**: Muscle (CC CLI)
**Status**: Draft — chờ user duyệt trước khi sang `02_plan`

---

## 1. Vấn đề user xác nhận

CMS đang chạy production. DB target prod là `cdc_cms_database` với user `cdc-cms-user`. Khi CMS start, runtime migrator (`internal/migrate/runner.go`) cố apply migration `005_pg_users.sql` → fail vì:

```
fatal: failed to initialize server
error: apply migrations: migrate: apply 005_pg_users: ERROR: permission denied to create role (SQLSTATE 42501)
```

User feedback (quote):

> đang chạy trên prod, do các file migr đang sét cứng nhiều thông tin quá

Tức là: **migration files hardcode cluster-level resources** (role names, password, DB name) khiến không deploy được trên môi trường khác cấu trúc.

---

## 2. Hardcoded surface — verified inventory

Quét toàn bộ `cdc-cms-service/migrations/*.sql` (53 file). Có **3 file độc hại** đối với DB control plane tách riêng:

| File | Hardcode | Hậu quả trên DB CMS riêng |
|---|---|---|
| `005_pg_users.sql` | `CREATE ROLE cdc_worker/cms_service/cdc_readonly` + password `cdc_worker_2026` / `cms_service_2026` / `readonly_2026` + `GRANT CONNECT ON DATABASE cdc_dw` | Fail CREATEROLE; grant sai DB; password leak vĩnh viễn trong Git |
| `039_set_search_path.sql` | `ALTER ROLE gpay_admin SET search_path = cdc_system, public` | Fail nếu role `gpay_admin` không tồn tại (prod CMS user là `cdc-cms-user`) |
| `042_search_path_with_auth.sql` | Cùng pattern, supersede 039 | Cùng fail |

50 file còn lại (001-004, 006-038, 040-041, 043-053) **portable** — chỉ tạo schema / table / index / function / extension, không động chạm role.

Comment header `-- Target DB: gpay-postgres-cdc (cdc_dw)` ở 051/052/053 là doc-only, body OK.

---

## 3. Phân loại migration (taxonomy đề xuất)

| Loại | Phạm vi | Lifecycle | Ai chạy | Quyền cần |
|---|---|---|---|---|
| **L1 — Cluster bootstrap** | role + cross-DB grant + ALTER ROLE | 1 lần / cluster, không phụ thuộc service | DBA / Ops (psql superuser) | `SUPERUSER` hoặc `CREATEROLE` |
| **L2 — Database bootstrap** | `CREATE DATABASE`, `CREATE EXTENSION` | 1 lần / DB | DBA hoặc app boot first-run | `CREATEDB`, owner |
| **L3 — Schema migration** | schema, table, index, function, trigger | Mỗi lần deploy service | App runtime migrator | `CREATE` trên schema, `OWNER`-equivalent |

`runner.go` hiện tại trộn cả 3 loại → mâu thuẫn kiến trúc đa-DB.

---

## 4. Constraints (user-mandated)

Trích từ rules user đã nêu:

- KHÔNG cheat DB hay đổi config để đạt kết quả.
- KHÔNG hardcode (lessons L590, L755, L933, L953 đã ghi).
- Hướng core systems, không workaround.
- Verify thực tế trước khi báo done (CLAUDE.md §3).
- Memory file APPEND-only (`05_progress.md`).
- Brain không sửa code trực tiếp; Muscle thực thi sau khi user approve plan.
- Prod safety: rollback-friendly, không phá idempotency của các env đã apply.

---

## 5. Non-functional requirements

| NFR | Mục tiêu |
|---|---|
| **Idempotency** | Mỗi migration apply 1 lần / DB; re-run = no-op. Tracker `cdc_system.schema_migrations` đã đảm bảo, refactor KHÔNG được phá. |
| **Backward compat** | Các env đã apply (dev `cdc_dw`) không cần re-migrate. Plan KHÔNG xoá row tracker. |
| **Forward compat** | Env mới (`cdc_cms_database` prod) khởi tạo sạch chạy được, không cần CREATEROLE. |
| **No secret in Git** | Password role không được commit; chuyển sang env var hoặc secret manager. |
| **Auditability** | Mỗi version migration đều có 1 row tracker — kể cả khi bị "skip" vẫn record để duy trì timeline. |
| **Reversibility** | Refactor có rollback path (revert code commit + tracker rows). |

---

## 6. Out of scope (sẽ document, không làm trong phase này)

- Đổi engine migrator (golang-migrate, atlas, sqitch).
- Multi-tenant migration (mỗi tenant 1 DB).
- Online schema change cho table cực lớn (50M+ rows) — lesson L661 đã có.
- Auto-rotate password role.

---

## 7. Definition of Done (cho phase `migration_refactor`)

1. ✅ Migration `005_pg_users` / `039` / `042` được "neutralize" (mark applied mà không exec body) trên `cdc_cms_database` prod **mà không sửa file đã apply trên `cdc_dw`** (giữ git history clean, không phá checksum).
2. ✅ Có cơ chế code-level rõ ràng (skip-list / tag header / split folder) để các deploy mới tự skip 3 file cluster-bootstrap.
3. ✅ Có script bootstrap riêng (psql) cho L1 — DBA chạy trước khi service start lần đầu trên cluster mới.
4. ✅ Password role chuyển sang env var (`${CDC_WORKER_PASSWORD}` v.v.) qua `psql -v`; file cũ trong git mark deprecated, kèm runbook rotate.
5. ✅ CMS prod khởi động thành công, log "migrations done" với `applied_now=0` (đối với 3 file kia) và phần còn lại apply bình thường.
6. ✅ Worker (`cdc_dw` env cũ) KHÔNG bị ảnh hưởng — verify `applied_now=0` sau rebuild.
7. ✅ Lesson được ghi vào `lessons.md` theo Global Pattern (CLAUDE.md §13).
8. ✅ Report `report_*.md` mô tả thay đổi + verification thực tế.
9. ✅ `/security-agent` rà soát code change trước khi báo done (CLAUDE.md §10 Security Auto-Check).

---

## 8. Acceptance criteria (testable)

| AC | Cách verify |
|---|---|
| AC1: CMS prod start được không fail CREATEROLE | Log `"PostgreSQL (control plane) connected"` → `"migrations done"` không có dòng `level":"fatal"` |
| AC2: Tracker prod có 3 version skip với marker | `SELECT version, applied_at, applied_by FROM cdc_system.schema_migrations WHERE version IN ('005_pg_users','039_set_search_path','042_search_path_with_auth')` — cả 3 row tồn tại, có metadata "skipped" |
| AC3: Dev DB (`cdc_dw`) không re-apply 3 file | Sau rebuild + restart CMS dev: log `applied_now=0` cho 3 version đó (tracker row đã có từ trước) |
| AC4: Worker không bị regression | Worker tick scheduler vẫn gap 60s ± 25ms (xem report 1121); 0 FATAL/panic |
| AC5: Password không còn trong file 005 raw | `grep "PASSWORD '" migrations/cluster_bootstrap/005_pg_users.sql` → 0 match (chỉ còn placeholder `:cdc_worker_password`) |
| AC6: README runbook đầy đủ | File `docs/migrations.md` mới: ghi 3 step bootstrap cho env mới |

---

## 9. Risk register

| Risk | Mitigation |
|---|---|
| R1: Sửa runner.go phá idempotency các env đã apply | Skip-list match exact filename string; vẫn check tracker trước; nếu đã applied → log skip thay vì record lại |
| R2: Bootstrap script DBA chạy sai → tạo role thiếu quyền | Script `psql -v ON_ERROR_STOP=1 -1`, kèm verification query cuối |
| R3: Rotate password phá worker đang chạy | Worker đọc password từ env/secret → rolling restart sau khi rotate, không cần migration thay đổi |
| R4: Prod CMS phải downtime để bootstrap | Bootstrap chỉ INSERT tracker + GRANT — không lock table, downtime gần như 0 |
| R5: Lessons không capture → lặp lại 6 tháng sau | Bắt buộc append lesson vào `lessons.md` theo CLAUDE.md §13 sau khi merge |

---

## 10. Stakeholders

- **User (decider)**: duyệt plan, cấp DBA window cho prod.
- **DBA / Ops**: chạy `scripts/bootstrap_cluster.sql` trên prod cluster.
- **Brain (Antigravity)**: review plan, không sửa code.
- **Muscle (CC CLI)**: thực thi code change sau khi user approve `02_plan` + `08_tasks` + `09_tasks_solution`.
- **Security agent**: rà soát code/secret rotation trước khi báo done.
