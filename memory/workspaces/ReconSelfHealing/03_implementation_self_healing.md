# Thiết kế Kỹ thuật - ReconSelfHealing

## 1. Thiết kế Luồng Xử lý trong Transmuter
```mermaid
graph TD
    A[Nhận onlySourceIDs] --> B[Quét Shadow rows matching onlySourceIDs]
    B --> C[Lọc các ID bị đánh dấu xóa _deleted=true trong Shadow]
    B --> D[Xác định ID mồ côi vật lý: onlySourceIDs không tồn tại trong Shadow]
    C --> E[Gộp danh sách ID soft-delete]
    D --> E
    E --> F[Thực thi UPDATE Master SET _deleted=true, _source_ts=NOW()]
```

## 2. Câu lệnh SQL Cập nhật
```sql
UPDATE "master_table" 
SET _deleted = true, _source_ts = ?, _updated_at = NOW() 
WHERE _source_id = ANY(?)
```

## 3. SQLite Dialect Adapter (cho Unit Test)
GORM Query/Exec callbacks dùng regex để thay thế dynamically:
- `= ANY(?)` -> `IN (?)`
- `NOW()` -> `datetime('now')`
- `::bigint` -> ``
- `::jsonb` -> ``
