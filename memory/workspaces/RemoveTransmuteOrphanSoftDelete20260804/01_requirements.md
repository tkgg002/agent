# 01 — Requirements: Thay Soft-Delete thành Hard Delete cho Orphan trên Transmute Pipeline

**Ngày:** 2026-08-04  
**Người yêu cầu:** User  
**Workspace:** RemoveTransmuteOrphanSoftDelete20260804

## Cập nhật yêu cầu (confirmed)
1. **Hard delete** thay soft-delete — cả 2 luồng
2. **Comment code cũ lại**, không xóa
3. **Fix N+1 query** luồng 2 Flatten Shrink — gom chunk luôn trong cùng task

---

## Yêu cầu đã được làm rõ

> "Orphan trên luồng transmute không soft-delete nữa" = **Xóa vật lý (hard DELETE) khỏi Master table**

Thay vì `UPDATE ... SET _deleted = true`, phải dùng `DELETE FROM ... WHERE`.

---

## Scope thay đổi

- **File duy nhất:** `centralized-data-service/internal/service/master/transmuter.go`
- **Không động vào:** strategy files, các module khác

---

## Chi tiết 2 luồng cần thay đổi

### Luồng 1 (L286–324): Orphan chung — Incremental Sync
**Hiện tại:**
```sql
UPDATE <master_table>
SET _deleted = true, _source_ts = ?, _updated_at = NOW()
WHERE _source_id IN (?)
```
**Thay bằng:**
```sql
DELETE FROM <master_table>
WHERE _source_id IN (?)
```

### Luồng 2 (L344–402): Flatten Array Shrink Orphan
**Hiện tại:**
```sql
-- Bước 1: SELECT để tìm orphan
SELECT _gpay_id FROM <master_table>
WHERE _gpay_id IN (?) AND _deleted = false

-- Bước 2: Soft-delete
UPDATE <master_table>
SET _deleted = true, _source_ts = ?, _updated_at = NOW()
WHERE _gpay_id IN (?)
```
**Thay bằng:**
```sql
-- Bước 1: SELECT để tìm orphan (vẫn giữ, nhưng bỏ filter _deleted = false vì hard delete)
SELECT _gpay_id FROM <master_table>
WHERE _gpay_id IN (?)

-- Bước 2: Hard delete
DELETE FROM <master_table>
WHERE _gpay_id IN (?)
```
