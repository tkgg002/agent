# 09_tasks_solution_security_2026-06-05.md — GAP-01 RLS + GAP-02 OCC (giải pháp concrete)

> **Agent**: Muscle:Claude-Opus-4.8 | 2 hạng mục security/concurrency — làm pass RIÊNG có verify, KHÔNG rush ("không sửa mù"). Doc này để execute chính xác.

---

## GAP-01 — RLS cho mọi master schema (prod-blocker)

### Root cause (verify file:line)
- `master_ddl_generator.go:220`: chỉ `enable_master_rls` khi `MasterSchema=="public"` → schema khác (vd `dw_centrallized_export_service`) **bỏ RLS hoàn toàn**.
- `038_*.sql:171-193` `cdc_system.enable_master_rls(p_table)`: **hardcode `public.%I`** + policy `USING(true) WITH CHECK(true)` (full-access, không cô lập).
- Function nằm ở `cdc_system` (control plane 5433) nhưng generator gọi trên **dest DB (5434)** (`db=GetMasterDB`). Nhiều khả năng dest KHÔNG có function này → call fail thầm (`if err==nil`) → RLS không áp **kể cả public**. Live: b2 `rls_applied:false`.

### Giải pháp (emit DDL trực tiếp trên dest, bỏ function cross-DB)
Sửa `master_ddl_generator.go` (Apply, thay block 220-224): emit RLS DDL trong CÙNG transaction chạy trên dest, cho MỌI schema:
```go
rlsSQL := []string{
  fmt.Sprintf(`ALTER TABLE %s ENABLE ROW LEVEL SECURITY;`, quoteDDLQualified(reg.MasterSchema, reg.MasterTable)),
  // PHASE-1: chưa FORCE để owner (worker gpay_admin) vẫn ghi được; policy tách phase-2.
}
// chạy rlsSQL trong tx dest (db.Exec). res.RLSApplied = true.
```
### ⚠️ QUYẾT ĐỊNH cần chốt (phase-2 policy) — KHÔNG đoán
Master table CHƯA có cột tenant → không thể viết policy cô lập per-row ngay. Chọn 1:
- **(A)** Chỉ ENABLE RLS (không FORCE) + KHÔNG policy permissive → role non-owner bị DENY (cô lập khỏi truy cập ngoài), worker (owner) bypass → an toàn, không lockout. **Khuyến nghị phase-1**.
- **(B)** Thêm cột `tenant_id` vào master + policy `USING (tenant_id = current_setting('app.tenant'))` → cô lập thật (cần thay đổi schema + pipeline set tenant). Phase-2, cần ADR.
- KHÔNG dùng lại `USING(true)` (vô nghĩa về bảo mật).
### Verify bắt buộc sau khi sửa
1. Apply master → `\d+ schema.table` thấy "Row security enabled".
2. **Worker upsert vẫn chạy** (transmute ghi master OK — không lockout). 3. Role non-owner SELECT bị chặn (nếu chọn A).

---

## GAP-02 — OCC out-of-order trên master upsert (MED)

### Root cause
- Master upsert `ON CONFLICT (_source_id) DO UPDATE ... WHERE _hash IS DISTINCT FROM EXCLUDED._hash` → KHÔNG so `_source_ts` → event CŨ (source_ts nhỏ hơn) có hash khác → **đè bản mới hơn**.
- ⚠️ CHỈ sửa master upsert (`transmuter.go upsertMaster`). **TUYỆT ĐỐI không đụng `upsert.go` shadow-side** (source→shadow).

### Bước 1 — TEST reproduce TRƯỚC (không sửa mù)
Test (worker `internal/service`): cùng `_source_id`, gửi 2 lần với `_source_ts` GIẢM dần (event2 cũ hơn event1) → assert master giữ bản `_source_ts` LỚN nhất. Hiện tại sẽ FAIL (event cũ đè).

### Bước 2 — Fix (sau khi reproduce)
Thêm guard vào ON CONFLICT của `upsertMaster`:
```sql
ON CONFLICT (_source_id) DO UPDATE SET ...
WHERE <t>._hash IS DISTINCT FROM EXCLUDED._hash
  AND EXCLUDED._source_ts >= <t>._source_ts   -- chặn event cũ đè bản mới
```
### Verify
Re-run test → PASS (master luôn giữ bản source_ts mới nhất); transmute bình thường không regression.

---

## Trạng thái các issue/gap khác (verify turn này)
- ✅ I2 (master bỏ _raw_data) LIVE · GAP-04 (menu) · GAP-03 (close-loop run-now) · GAP-05 (Delete schedule) · I6 (In Master dùng flag worker, bỏ endpoint control-plane).
- ✅ Gemini đã làm: I1, I10, I4 (source_data_type), I8 (manual mapping), GAP-06 (transform_fn), I3 (shadow_status + in_shadow + gate disabled), I9 (scan-array modal review/promote).
- Còn cần pass riêng: **GAP-01 (quyết định policy A/B), GAP-02 (test-first)**.
