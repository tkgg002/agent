# Kế hoạch Triển khai - Sửa lỗi sai lệch Count hiển thị trên Dashboard (ListLatest)

## 1. Phân tích nguyên nhân
Hàm `ListLatest` truy vấn `listLatestPrimary` thực hiện gộp kết quả từ `cdc_reconciliation_report` (quét cửa sổ thời gian giới hạn) và `cdc_recon_smoke_result` (quét toàn bộ bảng). Do `DISTINCT ON` lấy bản ghi mới nhất, khi vừa chạy Full Search xong, bản ghi từ `cdc_reconciliation_report` sẽ đè lên smoke check. Tuy nhiên, counts (`source_count`, `dest_count`) của Full Search chỉ nằm trong khoảng thời gian phân tích (ví dụ: 8 bản ghi), gây ra hiện tượng lệch ảo trên Dashboard (Source: 8, Shadow: 8, Master: 457).

## 2. Giải pháp đề xuất
Thay vì lấy trực tiếp cột `source_count`/`dest_count` từ `cdc_reconciliation_report` cho phần query leg tương ứng, ta sẽ thực hiện `LEFT JOIN LATERAL` với `cdc_recon_smoke_result` để lấy kết quả counts thực tế mới nhất của table/schema/segment đó.
Nếu không tồn tại bản ghi smoke test nào, ta sẽ fallback về counts của chính báo cáo.

### Chi tiết thay đổi trong SQL query `listLatestPrimary` của `recon_read_repo_gorm.go`:
```sql
			SELECT
				r.id,
				r.run_id,
				NULL::bigint AS cycle_id,
				r.segment,
				NULL::text AS source_type,
				NULL::text AS source_host,
				r.source_db,
				COALESCE(s.source_total, r.total_source_count) AS source_total,
				COALESCE(s.source_active, r.source_count) AS source_active,
				COALESCE(s.shadow_total, r.total_dest_count) AS shadow_total,
				COALESCE(s.shadow_active, r.dest_count) AS shadow_active,
				r.master_schema,
				r.master_table,
				s.master_total AS master_total,
				s.master_active AS master_active,
				r.diff,
				r.status,
				r.error_message,
				r.duration_ms,
				r.checked_at,
				r.shadow_schema,
				r.shadow_table,
				NULL::text AS source_table,
				COALESCE(
					CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.shadow_active, 0) ELSE COALESCE(s.source_active, 0) END,
					r.source_count
				) AS source_count,
				COALESCE(
					CASE WHEN r.segment = 'shadow_master' THEN COALESCE(s.master_active, 0) ELSE COALESCE(s.shadow_active, 0) END,
					r.dest_count
				) AS dest_count
			FROM cdc_system.cdc_reconciliation_report r
			LEFT JOIN LATERAL (
				SELECT source_total, source_active, shadow_total, shadow_active, master_total, master_active
				FROM cdc_system.cdc_recon_smoke_result
				WHERE shadow_schema = r.shadow_schema
				  AND shadow_table = r.shadow_table
				  AND (master_schema IS NOT DISTINCT FROM r.master_schema)
				  AND (master_table IS NOT DISTINCT FROM r.master_table)
				  AND segment = r.segment
				ORDER BY checked_at DESC
				LIMIT 1
			) s ON TRUE
```

## 3. Kế hoạch xác minh

### Kiểm thử Tự động
- Build dự án `cdc-cms-service`.
- Chạy unit tests của `queries` package để xác nhận không lỗi biên dịch/chạy query.

### Kiểm thử Thư mục
- Dùng `cURL` gọi endpoint `http://localhost:8083/api/reconciliation/report` (ListLatest) và kiểm tra xem pipeline `export_jobs` có còn bị lệch count (ví dụ: hiện 8 thay vì 457) hay không.
- Dashboard của `export_jobs` phải hiển thị chính xác counts từ smoke result mới nhất.
