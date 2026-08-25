# Sửa Lỗi Phân Trang API Activity Log (Page 2, 3, 4,...)

Khi truy vấn API `/api/activity-log?page=2&page_size=30`, kết quả trả về không đúng số lượng bản ghi hoặc bị trùng lặp do cơ chế join bảng 1-N. Kế hoạch này mô tả cách tối ưu câu SQL join để khắc phục lỗi phân trang.

## User Review Required

> [!IMPORTANT]
> Thay đổi này chỉ sửa đổi cách join bảng metadata `cdc_system.master_binding` (từ `LEFT JOIN` thông thường sang `LEFT JOIN LATERAL ... LIMIT 1`) trong persistence layer để chống nhân bản dòng. Không có thay đổi nào đối với cấu trúc cơ sở dữ liệu hay giao diện API.

## Open Questions

Không có câu hỏi mở nào. Giải pháp đã được xác thực trực tiếp qua script kiểm tra kết nối DB nội bộ.

## Proposed Changes

### cdc-cms-service

#### [MODIFY] [activity_log_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go)

Thay đổi phép join bảng `master_binding` ở hàm `enrichmentFromClause` (dòng 70-72):

```diff
-		LEFT JOIN cdc_system.master_binding mb
-		  ON mb.shadow_binding_id = sb.shadow_binding_id
-		 AND mb.is_active = TRUE
+		LEFT JOIN LATERAL (
+			SELECT mb.master_schema, mb.master_table
+			FROM cdc_system.master_binding mb
+			WHERE mb.shadow_binding_id = sb.shadow_binding_id
+			  AND mb.is_active = TRUE
+			ORDER BY mb.updated_at DESC, mb.id DESC
+			LIMIT 1
+		) mb ON TRUE
```

## Verification Plan

### Automated Tests
- Chạy unit tests để xác nhận thay đổi không gây lỗi biên dịch:
  ```bash
  go test ./test/... -count=1 -short
  ```

### Manual Verification
- Chạy script kiểm thử trực tiếp `/Users/trainguyen/.gemini/antigravity/brain/42b7c576-849f-48b9-b512-f23dfa9ada63/scratch/test_db.go` để xác nhận số dòng của trang 1 và trang 2 đều trả về chính xác 30 dòng (thay vì 34 dòng).
