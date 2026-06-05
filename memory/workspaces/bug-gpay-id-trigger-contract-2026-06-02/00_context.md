# 00_context.md — Bug `_gpay_id NULL` Contract Drift

| Field | Value |
|---|---|
| **Workspace** | `bug-gpay-id-trigger-contract-2026-06-02` |
| **Date** | 2026-06-02 |
| **Severity** | 🔴 HIGH — block CDC pipeline prod, data loss potential |
| **Owner Brain** | Antigravity / Claude Opus 4.7 |
| **Scope** | **Brain plan + spec only** (CLAUDE.md §12). Muscle execute sau user approve. |
| **Target service** | `data-hub/centralized-data-service` + `data-hub/cdc-cms-service/migrations/` |

---

## 1. Triệu chứng (User report)

```
flush after batch (enqueued=5000, persisted=784):
  batch upsert chunk failed:
    ERROR: null value in column "_gpay_id" of relation "tokens"
    violates not-null constraint (SQLSTATE 23502)
  (fallback persisted 0 rows)
```

- **Local:** không xảy ra
- **Prod:** xảy ra sau khi fix vòng trước (event_handler PK lookup + batch_buffer remove `effectivePK == "id"` condition)
- Pre-fix: lỗi bị overshadow bởi `ON CONFLICT` mismatch; fix vòng trước đẩy flow đi xa hơn → bóc trần bug ẩn này.

---

## 2. Bản chất bug — **Contract Drift 3 lớp**

Cùng concept "_gpay_id auto-fill" được claim ở 3 nơi nhưng **không nơi nào implement thực tế**:

| Lớp | File / Line | Claim | Reality |
|---|---|---|---|
| **L1 — Comment trong code Go** | `internal/handler/batch_buffer.go:246-250` | *"V2 shadow contract: bootstrap (ShadowAutomator) emits `_gpay_id BIGINT PK` (sonyflake trigger fills)"* | KHÔNG có trigger fill |
| **L2 — Schema DDL Go runtime** | `internal/sinkworker/schema_manager.go:226` | `"_gpay_id" BIGINT PRIMARY KEY` | KHÔNG có `DEFAULT`, KHÔNG có `IDENTITY` |
| **L3 — Migration SQL** | `migrations/schema/ids/018_sonyflake_v125_foundation.sql:130` | `cdc_internal.tg_fencing_guard()` (BEFORE INSERT) | CHỈ check fencing token, **KHÔNG `NEW._gpay_id := ...`** |

### Đối chiếu intent ngược thời gian
- Migration **003 (v1.12)** thiết kế V1 table: `id BIGINT PRIMARY KEY DEFAULT nextval(seq_<table>_id)` — DB-side sequence fill ✅
- Migration **018 (v1.25)** thiết kế V2 shadow: bỏ sequence, comment `"Go Worker will replace with Sonyflake when using pgx.Batch"` — nhưng Go-side **CHƯA bao giờ implement** fill `_gpay_id`.
- → Bug từ thời design V2 shadow contract, **inherit từ v1.25 → hiện tại**.

---

## 3. Vì sao local OK + prod fail

Schema drift do 2 đường tạo table song song:

| Đường tạo table | Local | Prod |
|---|---|---|
| Migration SQL (`make migrate`) | ✅ Dev có thể đã chạy + manual patch DEFAULT khi gặp bug | ❌ Migration mới fresh, không có patch |
| `schema_manager.createShadowTable()` Go runtime | ❌ Có thể không trigger trên local (table đã exist) | ✅ Lazy create lần đầu encounter table |

→ Prod sink lần đầu encounter table V2 shadow `tokens` → Go code tạo table với `_gpay_id BIGINT PRIMARY KEY` no DEFAULT → INSERT fail.

---

## 4. Vì sao fix vòng trước ĐÚNG (không revert)

Bug đã fix:
- `event_handler.go:210` — đã sửa lookup mapping rule lấy PK (thay hardcode `"id"`)
- `batch_buffer.go:252` — đã remove `&& effectivePK == "id"` → luôn remap `_source_id` khi schema có cột này

→ Fix #1 + #2 ĐÚNG về design intent. Chỉ là **đẩy flow vào đến đúng layer test contract `_gpay_id`** → layer này bị thiếu implementation → fail.

---

## 5. Mục tiêu workspace này

1. Align **3 lớp contract** (comment ↔ DDL ↔ migration) về 1 implementation duy nhất.
2. Single Source of Truth cho việc fill `_gpay_id`.
3. Heal existing prod table chưa có DEFAULT (idempotent).
4. Test cover: reproduce bug + regression guard.
5. KHÔNG đụng business logic, KHÔNG đụng schema column khác.

---

## 6. Out-of-Scope
- ❌ Thay đổi `_source_id` UNIQUE / ON CONFLICT logic (fix trước đã xử)
- ❌ Refactor `BatchBuffer` / `BuildBatchUpsertSQLInSchema` ngoài patch tối thiểu
- ❌ Đụng tableV1 (`migrations/003_sonyflake_schema.sql` — vẫn dùng per-table sequence)
- ❌ Migrate sang Snowflake/UUID — giữ sonyflake design intent
- ❌ Workspace hexagonal refactor v2 (track riêng `feature-cdc-cms-hexagonal-refactor-2026-06-01`)

---

## 7. Tham chiếu file đã đọc

| File | Lý do |
|---|---|
| `data-hub/centralized-data-service/internal/handler/batch_buffer.go` (line 246-336) | Caller UPSERT, comment claim sai |
| `data-hub/centralized-data-service/internal/service/schema_adapter.go` (line 378-568) | UPSERT SQL builder, getMetadataInsertCols miss `_gpay_id` |
| `data-hub/centralized-data-service/internal/sinkworker/schema_manager.go` (line 220-301) | createShadowTable DDL miss DEFAULT |
| `data-hub/centralized-data-service/pkgs/idgen/sonyflake.go` | Sonyflake Go implementation (đã có sẵn, chưa được call ở UPSERT path) |
| `data-hub/cdc-cms-service/migrations/schema/ids/003_sonyflake_schema.sql` | V1.12 design — per-table sequence DEFAULT |
| `data-hub/cdc-cms-service/migrations/schema/ids/018_sonyflake_v125_foundation.sql` | V1.25 design — fencing guard, NOT fill _gpay_id |
