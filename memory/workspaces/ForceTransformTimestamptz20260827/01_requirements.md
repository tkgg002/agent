# 01 Requirements — Force Transform + Timestamp Bug Fix

## Task 1 — Bug Fix: Transform in-progress sai trên child binding table

**Triệu chứng:** Click Transform trên `payment_bills` → cột Transform hiện "Đang chạy" trên cả `payment_bills` lẫn `payment_bills_1`

**Root cause:** `activeTransformJobs` trong `TableRegistry.tsx` dùng `source_object_id` làm key. Child binding tables có cùng `source_object_id` → đọc nhầm job của parent.

**DoD:**
- [ ] Child binding table dùng key riêng (ví dụ `b:${shadow_binding_id}`)
- [ ] Trigger transform trên `payment_bills` → `payment_bills_1` KHÔNG hiện in-progress

---

## Task 2 — Feature: Force Transform Mode

**Yêu cầu:**
- User đổi field type từ `timestamp` → `timestamptz` qua "sync fields to shadow"
- Field đã có giá trị (sai do ALTER TYPE) → batch transform thông thường bỏ qua (vì `field IS NULL = false`)
- Cần re-extract field đó từ `_raw_data` mà KHÔNG cần NULL trước

**Design đã approve:**
- Thêm `force: bool` + `force_fields: []string` vào `BatchTransformPayload`
- Khi `force=true`:
  - SET clause: CHỈ các field trong `force_fields` (không đụng field khác)
  - WHERE clause: `_raw_data IS NOT NULL` (bỏ `IS NULL` check)
- Khi `force=false`: giữ nguyên behavior cũ

**Fix đi kèm: `BuildCastExpr` ELSE branch cho `timestamptz`**
- Tách case `timestamptz` / `timestamp with time zone` riêng
- ELSE: `(NULLIF(_raw_data->>'field', ''))::TIMESTAMPTZ` (thay vì `::TIMESTAMP`)

**DoD:**
- [ ] `force=true` + `force_fields=["X"]` chỉ overwrite field X, các field khác không đổi
- [ ] WHERE: `_raw_data IS NOT NULL` → process toàn bộ 100M records dạng chunked CTE
- [ ] `BuildCastExpr` cho `timestamptz` dùng `::TIMESTAMPTZ` ở ELSE branch
- [ ] FE có thể trigger force transform từ UI (sync fields modal hoặc button riêng)
