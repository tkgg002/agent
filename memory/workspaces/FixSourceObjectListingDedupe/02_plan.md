# 02_plan — FixSourceObjectListingDedupe

## Approach
**Option A elegant variant**: thay vì GROUP BY + array_agg (đổi response shape), convert `LEFT JOIN cdc_table_registry tr ON ...` → `LEFT JOIN LATERAL (SELECT ... FROM tr WHERE ... ORDER BY ... LIMIT 1) tr ON TRUE`. Vừa scope theo `source_connection_id` (semantic correct) vừa đảm bảo deterministic 1 row.

## Steps
1. Sửa `listBaseFromWhere` constant trong `source_object_read_repo_gorm.go`:
   - Thay `LEFT JOIN cdc_system.cdc_table_registry tr ON ...` bằng LATERAL subquery.
   - Subquery scope thêm: `(tr.source_connection_id = so.source_connection_id OR tr.source_connection_id IS NULL)`.
   - Order: `(tr.source_connection_id IS NULL) ASC, tr.id ASC` → exact match thắng legacy NULL; trong cùng nhóm thì id nhỏ hơn thắng.
   - `LIMIT 1`.
2. SELECT clause giữ nguyên — vẫn dùng `tr.id`, `tr.sync_interval`, `tr.priority`, `tr.timestamp_field`, `tr.notes`, `tr.is_table_created`, `tr.updated_at`. LATERAL subquery phải SELECT đủ các cột này.
3. COUNT(*) tự động đúng vì cùng FROM clause.
4. (Optional) Apply cùng pattern cho `GetMappingContextByRegistryID` nếu nó cũng có issue tương tự — nhưng query đó FROM tr nên cardinality khác, defer ra ngoài scope.

## Risks
- R-1: Legacy row với `source_connection_id IS NULL` vẫn có thể match nhiều so → fallback OR clause kéo NULL row vào. Mitigation: ORDER + LIMIT 1.
- R-2: PG planner có thể chọn nested loop với LATERAL trên large dataset. Mitigation: index `idx_ctr_source_connection` đã có.
- R-3: COUNT(*) trước đây = 6 (sai), giờ thành 4. FE đang dựa vào số đó? Check — đây là fix bug, FE nên hiển thị đúng.

## Out of scope
- Không backfill thêm dữ liệu vào `cdc_table_registry.source_connection_id` (đã được 055 lo).
- Không động `GetMappingContextByRegistryID` (defer — pattern khác, dùng `tr.id = ?` đã unique).
- Không thay đổi response wire shape.
