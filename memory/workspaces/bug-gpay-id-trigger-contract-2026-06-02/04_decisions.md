# 04_decisions.md — Architecture Decision Records

> Mỗi ADR gồm: Context · Options · Decision · Consequences. Brain plan only.

---

## ADR-01 — Single Source of Truth cho fill `_gpay_id`: **DB-side DEFAULT + Go-side fallback (Option A+C combo)**

### Context
Hiện tại có 3 chỗ "claim fill _gpay_id" nhưng 0 chỗ thực sự fill. Cần chốt 1 implementation chính + cơ chế dự phòng.

### Options
| Opt | Mô tả | Pros | Cons |
|---|---|---|---|
| A | Go-side: builder thêm `_gpay_id` value từ `idgen.Next()` vào INSERT | Control hoàn toàn từ app, dễ test | Mỗi caller phải nhớ fill; nếu builder mới quên → tái diễn bug |
| B | DB-side: `DEFAULT cdc_internal.sf_nextval()` trên cột | Bullet-proof; mọi INSERT (kể cả ad-hoc) đều an toàn | Cần sync machine ID giữa Go pod và DB function |
| C | BEFORE INSERT trigger: `IF NEW._gpay_id IS NULL THEN NEW._gpay_id := ...` | Backward-compat tốt, kết hợp được với trigger fencing hiện hữu | Trigger overhead per row; harder to reason about |
| **A+B** | **Go fill + DB DEFAULT fallback** | Defense-in-depth, idempotent, an toàn cả khi 1 lớp bypass | Phải maintain 2 nguồn ID — chấp nhận vì cùng `idgen`/`sf_nextval` decode tương thích |

### Decision
**Option A + B combo**:
- **Primary (B):** Migration mới `019_sonyflake_default_fill.sql` tạo function `cdc_internal.sf_nextval()` + ALTER TABLE SET DEFAULT cho mọi V2 shadow table.
- **Secondary (A):** Go DDL `schema_manager.go` emit `DEFAULT cdc_internal.sf_nextval()` ngay khi `createShadowTable()`. Builder UPSERT KHÔNG cần thay đổi (DB lo) — tiết kiệm patch.
- **C bỏ qua:** vì DEFAULT cover tất cả case, trigger chỉ thêm overhead.

### Consequences
- ✅ Idempotent — chạy migration nhiều lần OK
- ✅ Mọi đường tạo table (Go runtime / migration / manual psql) đều có DEFAULT
- ✅ Builder Go không thay đổi → giảm rủi ro regression
- ⚠ Phải đồng bộ machine ID giữa pod và DB — xử lý qua session var `app.fencing_machine_id` (đã có sẵn từ fencing system)

---

## ADR-02 — Machine ID source cho `sf_nextval()`: **dùng `app.fencing_machine_id` session var**

### Context
Sonyflake cần `machine_id` (16-bit) để distinguish nguồn ID. DB function chạy server-side, không tự biết pod nào gọi.

### Options
| Opt | Mô tả |
|---|---|
| A | Hard-code `machine_id = 0` cho DB-side | Cons: tất cả pod cùng machine ID → collision |
| B | Đọc từ `current_setting('app.fencing_machine_id')` — đã set bởi connection bootstrap | Pros: reuse infra hiện hữu (fencing trigger đã đọc cùng var) |
| C | Tạo connection pool riêng cho mỗi machine, dùng `SET LOCAL` | Cons: phức tạp, ít ROI |

### Decision
**Option B.** Fencing system đã set `app.fencing_machine_id` mỗi connection (xem `migrations/schema/ids/018_sonyflake_v125_foundation.sql` line 100-120). `sf_nextval()` chỉ cần đọc cùng var → 0 thay đổi infra.

### Consequences
- ✅ Reuse 100% infra fencing
- ✅ Machine ID consistency giữa fencing và sonyflake gen
- ⚠ Connection KHÔNG set `app.fencing_machine_id` (ad-hoc psql) sẽ lỗi → đúng intent (chỉ Go sink mới được INSERT V2 shadow)

---

## ADR-03 — Heal existing tables: **`ALTER TABLE ... SET DEFAULT` (metadata-only)**

### Context
Prod có table V2 shadow đã tạo sẵn, không có DEFAULT. Cần update không downtime.

### Options
| Opt | Mô tả |
|---|---|
| A | `ALTER TABLE ... SET DEFAULT cdc_internal.sf_nextval()` | Metadata-only, lock ngắn (millisecond) |
| B | DROP + CREATE TABLE migration | Cons: mất data, không khả thi |
| C | Trigger BEFORE INSERT cho retrofit | Cons: phải attach trigger từng table; trigger conflict với fencing chain |

### Decision
**Option A.** PostgreSQL `SET DEFAULT` chỉ update catalog `pg_attribute.atthasdef` — không rewrite row. Lock `AccessExclusiveLock` < 1ms cho table empty/active.

### Consequences
- ✅ Zero rewrite, zero data movement
- ✅ Existing rows giữ nguyên `_gpay_id` cũ (nếu đã có)
- ⚠ Migration phải iterate qua `information_schema.tables` để tìm V2 shadow table — wrap trong `DO $$ ... $$` block

---

## ADR-04 — Migration file naming: **`019_sonyflake_default_fill.sql`** (kế tiếp 018)

### Context
Migration `018_sonyflake_v125_foundation.sql` là gốc bug. Patch tiếp theo nên cùng namespace `schema/ids/`.

### Decision
- File: `data-hub/cdc-cms-service/migrations/schema/ids/019_sonyflake_default_fill.sql`
- Số 019 = liên tiếp 018
- Subdir `ids/` (giống 003, 018) thay vì `core/` hay `ops/`

### Consequences
- ✅ Continuity với history
- ✅ Reviewer nhận diện ngay scope (sonyflake ID)

---

## ADR-05 — Comment fix: **rewrite, không xóa**

### Context
Comment hiện tại nói dối. Có thể (1) xóa, (2) rewrite chính xác, (3) thêm TODO.

### Decision
**Rewrite chính xác**, link tới function thật:
```go
// V2 shadow contract: `_gpay_id BIGINT PRIMARY KEY DEFAULT cdc_internal.sf_nextval()`
// (xem migrations/schema/ids/019_sonyflake_default_fill.sql) + `_source_id TEXT
// NOT NULL` partial UNIQUE WHERE NOT _deleted (ON CONFLICT anchor).
```

### Consequences
- ✅ Dev mới đọc comment → tìm được function thật
- ✅ Audit trail rõ: comment ↔ migration cùng nhắc nhau

---

## ADR-06 — KHÔNG sửa builder `getMetadataInsertCols` (skip Option A của ADR-01)

### Context
ADR-01 chốt A+B. Tuy nhiên nếu B (DB DEFAULT) đủ, A có dư thừa.

### Decision
**Chốt CHỈ B** (đảo ngược ý ban đầu sau khi cân nhắc): KHÔNG thêm `_gpay_id` vào `getMetadataInsertCols`. Lý do:
1. DB DEFAULT cover 100% case INSERT không chỉ định cột → đủ.
2. Thêm Go-side fill = 2 nguồn ID = phải đồng bộ machine ID twice = thêm bề mặt bug.
3. Patch tối thiểu (NFR-7).

→ Update ADR-01 Decision: **CHỈ B** (DB-side DEFAULT). A bỏ. C bỏ. 1 lớp duy nhất, nhưng cover mọi đường INSERT.

### Consequences
- ✅ Patch nhỏ hơn (chỉ 1 migration + 1 dòng Go DDL + 1 comment)
- ✅ Single source of truth thực sự
- ⚠ Nếu ai đó tắt DEFAULT (ALTER ... DROP DEFAULT) → tái bug. Mitigation: CI test AC-1 + AC-2 chạy thường xuyên

---

## ADR-07 — Test: **integration test với Postgres thật** thay vì mock

### Context
Bug chỉ xảy ra khi DB thực thi DEFAULT — mock không bắt được.

### Decision
- Test mới `internal/handler/batch_buffer_v2shadow_test.go` chạy với `testcontainers-go/postgres` (đã có ở repo).
- Setup: chạy migration `019` → tạo V2 shadow table → INSERT không chỉ định `_gpay_id` → assert row có ID.

### Consequences
- ✅ Bắt được regression sớm
- ⚠ Test chậm hơn (~5s startup container) — chấp nhận vì bug đắt hơn

---

## ADR-08 — Brain scope: **KHÔNG đụng V1 path** (migration 003)

### Context
V1 dùng per-table sequence DEFAULT, hoạt động ổn. Có cám dỗ migrate sang sonyflake luôn.

### Decision
**KHÔNG.** Out-of-scope. Lý do:
- V1 không lỗi → không sửa (CLAUDE.md §6 Simplicity First).
- Migrate V1 → V2 là feature riêng, không phải bug fix.
- Risk lớn (V1 đang prod stable, đụng = downtime).

### Consequences
- ✅ Scope nhỏ, dễ review
- ✅ Không introducedẻ unintended change

---

## Summary

| ADR | Quyết định |
|---|---|
| 01 | Single source = DB DEFAULT `sf_nextval()` |
| 02 | Machine ID từ session var `app.fencing_machine_id` |
| 03 | Heal table existing = `ALTER TABLE SET DEFAULT` (metadata-only) |
| 04 | Migration file `019_sonyflake_default_fill.sql` |
| 05 | Comment rewrite chính xác, link tới migration |
| 06 | KHÔNG sửa Go builder (B đủ, A dư) |
| 07 | Integration test với testcontainers Postgres |
| 08 | KHÔNG đụng V1 path (migration 003) |
