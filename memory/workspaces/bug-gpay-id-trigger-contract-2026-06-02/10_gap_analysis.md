# 10_gap_analysis.md — Contract Drift Evidence

> Đo thực tế bằng `Read` + `Grep` trên codebase. Mỗi gap có file:line evidence.

---

## Gap matrix

| Gap | Severity | Layer | File:Line | Loại drift |
|---|---|---|---|---|
| G1 | 🔴 HIGH | Comment | `internal/handler/batch_buffer.go:246-250` | Documentation lies |
| G2 | 🔴 HIGH | Go DDL | `internal/sinkworker/schema_manager.go:226` | Missing constraint |
| G3 | 🔴 HIGH | Migration SQL | `migrations/schema/ids/018_sonyflake_v125_foundation.sql:130-159` | Missing implementation |
| G4 | 🟡 MED | Builder | `internal/service/schema_adapter.go:getMetadataInsertCols` | Missing column in INSERT |
| G5 | 🟡 MED | Idgen util | `pkgs/idgen/sonyflake.go` | Existing util not called |
| G6 | 🟢 LOW | Reverse intent | `migrations/schema/ids/003_sonyflake_schema.sql:1-80` | V1 had DEFAULT, V2 dropped |
| G7 | 🟢 LOW | Local-prod drift | (env) | Schema drift between envs |

---

## G1 — Comment trong code Go nói dối

### Evidence
```go
// internal/handler/batch_buffer.go:246-250
// V2 shadow contract: bootstrap (ShadowAutomator) emits `_gpay_id BIGINT
// PK` (sonyflake trigger fills) + `_source_id TEXT NOT NULL` partial
// UNIQUE WHERE NOT _deleted (ON CONFLICT anchor).
effectivePK := first.PrimaryKeyField
```

### Reality
- `grep -rn "sonyflake trigger" data-hub/cdc-cms-service/migrations/` → **0 match**
- `grep -rn "NEW._gpay_id" data-hub/cdc-cms-service/migrations/` → **0 match**

### Tác động
Dev mới đọc comment tin rằng có trigger → không phát hiện bug khi onboard / debug.

---

## G2 — Go DDL khai báo cột thiếu DEFAULT

### Evidence
```go
// internal/sinkworker/schema_manager.go:226
cols := []string{
    `"_gpay_id" BIGINT PRIMARY KEY`,   // ← NO DEFAULT, NO IDENTITY
    `"_source_id" TEXT NOT NULL`,
    ...
}
```

### Tác động
Khi `createShadowTable()` chạy lần đầu trên prod (lazy create) → table sinh ra **không có cơ chế auto-fill** → mọi INSERT không chỉ định `_gpay_id` raise `23502`.

---

## G3 — Migration trigger CHỈ check fencing, KHÔNG fill

### Evidence
```sql
-- migrations/schema/ids/018_sonyflake_v125_foundation.sql:130-159
CREATE OR REPLACE FUNCTION cdc_internal.tg_fencing_guard()
RETURNS TRIGGER AS $$
DECLARE
  v_session_machine INTEGER;
  v_session_token   BIGINT;
  v_current_token   BIGINT;
BEGIN
  v_session_machine := current_setting('app.fencing_machine_id', false)::INTEGER;
  v_session_token   := current_setting('app.fencing_token', false)::BIGINT;
  -- ... chỉ check token match, RAISE EXCEPTION nếu mismatch
  RETURN NEW;
  -- KHÔNG có "NEW._gpay_id := ..." dòng nào
END;
$$ LANGUAGE plpgsql;
```

### Tác động
Trigger được attach trong `schema_manager.go:286-291` cho mọi V2 shadow table — nhưng chỉ phục vụ zombie-pod protection, **KHÔNG** sinh ID.

---

## G4 — Builder UPSERT bỏ qua cột `_gpay_id`

### Evidence
```go
// internal/service/schema_adapter.go (getMetadataInsertCols)
func getMetadataInsertCols(schema *TableSchema, pkField string) []string {
    var cols []string
    if _, ok := schema.Columns["_raw_data"]; ok { cols = append(cols, `"_raw_data"`) }
    if _, ok := schema.Columns["_source"]; ok { cols = append(cols, `"_source"`) }
    if _, ok := schema.Columns["_synced_at"]; ok { cols = append(cols, `"_synced_at"`) }
    if _, ok := schema.Columns["_version"]; ok { cols = append(cols, `"_version"`) }
    if _, ok := schema.Columns["_hash"]; ok { cols = append(cols, `"_hash"`) }
    if _, ok := schema.Columns["_source_id"]; ok && pkField != "_source_id" {
        cols = append(cols, `"_source_id"`)
    }
    if _, ok := schema.Columns["_source_ts"]; ok { cols = append(cols, `"_source_ts"`) }
    return cols
    // KHÔNG có "_gpay_id" dòng nào
}
```

### Tác động
Builder không thêm `_gpay_id` vào INSERT statement → nếu DB không có DEFAULT → NULL violation.

---

## G5 — Util Sonyflake Go đã có nhưng không dùng

### Evidence
- File `pkgs/idgen/sonyflake.go` tồn tại, export `Next() uint64`.
- `grep -rn "idgen.Next\|idgen.NextSonyflake" data-hub/centralized-data-service/internal/` → match ở nơi khác (event publish?), **KHÔNG match** trong UPSERT path (`handler/batch_buffer.go` / `service/schema_adapter.go`).

### Tác động
Implementation Go-side đã sẵn sàng, chỉ thiếu wire vào builder. Low-hanging fruit cho Option A.

---

## G6 — V1 từng có DEFAULT, V2 cố ý bỏ nhưng không thay thế

### Evidence
```sql
-- migrations/schema/ids/003_sonyflake_schema.sql (v1.12)
v_sql := format(
    'CREATE TABLE %I (
        id BIGINT PRIMARY KEY DEFAULT nextval(%L),  -- ← V1 có DEFAULT
        source_id VARCHAR(200) NOT NULL,
        ...
    )', p_target_table, v_seq_name);
```
```sql
-- migrations/schema/ids/018_sonyflake_v125_foundation.sql (v1.25)
-- Comment: "Go Worker will replace with Sonyflake when using pgx.Batch"
```

### Conclusion
Design v1.25 **cố ý** chuyển trách nhiệm về Go-side, nhưng Go-side **CHƯA bao giờ** implement. Bug intent-vs-implementation gap kéo dài từ v1.25 → hiện tại.

---

## G7 — Schema drift local vs prod

### Evidence (suy luận, không có file)
- Local: Dev có thể đã chạy `ALTER TABLE ... SET DEFAULT` manual khi gặp lỗi từ rất lâu — hoặc local table được tạo từ migration 003 (V1 style có DEFAULT).
- Prod: Migration mới fresh + sink mới tạo V2 shadow lần đầu → không có patch.

### Mitigation
Migration `019_sonyflake_default_fill.sql` sẽ idempotent ALTER cả 2 môi trường → eliminate drift.

---

## Tổng kết

| Lớp | Trạng thái hiện tại | Sau fix |
|---|---|---|
| Comment | Nói dối (claim trigger fills) | Khớp với function thật |
| Go DDL | Thiếu DEFAULT | Có DEFAULT `sf_nextval()` |
| Migration | Chỉ fencing | Thêm `sf_nextval()` + ALTER existing tables |
| Builder | Skip `_gpay_id` | (Tùy chọn) thêm Go-side fill defense-in-depth |
| Util `idgen` | Có nhưng không dùng | Dùng trong builder (Option A) |
| Env drift | Local OK, prod fail | Cả 2 đồng nhất sau migration |

→ **3 layer align** = 1 root + 2 defense lớp.
