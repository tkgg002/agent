# Phân Tích Kỹ Thuật: Nhân Bản Dòng Khi Join master_binding

## 1. Triệu chứng
Khi gọi API `GET /api/activity-log?page=2&page_size=30`, dữ liệu trả về bị lệch. Lượng row nhận được (ví dụ: 34 rows) vượt quá giới hạn `page_size = 30`.

## 2. Phân tích nguyên nhân
Trong file `activity_log_read_repo_gorm.go`, phần `enrichmentFromClause` định nghĩa cách các bản ghi log được liên kết thông tin:
```sql
		LEFT JOIN cdc_system.master_binding mb
		  ON mb.shadow_binding_id = sb.shadow_binding_id
		 AND mb.is_active = TRUE
```
Trong hệ thống, một bảng shadow (`shadow_binding`) có thể được map tới nhiều bảng đích master (`master_binding`) khác nhau (ví dụ: chuyển tiếp sang nhiều db hoặc master tables). Đây là quan hệ 1-N.

Do đó:
- Subquery `innerQuery` lấy ra đúng 30 dòng logs từ `cdc_activity_log`.
- Khi thực hiện join ngoài với `cdc_system.master_binding mb`, nếu có bất kỳ dòng log nào thuộc một `shadow_binding` có nhiều hơn 1 `master_binding` hoạt động, dòng log đó sẽ bị lặp lại tương ứng với số lượng `master_binding`.
- Việc nhân bản này diễn ra sau khi phân trang trong subquery hoàn tất, làm phình to tập kết quả trả về của câu query chính.

## 3. Giải pháp khắc phục
Thay thế phép join thông thường bằng `LEFT JOIN LATERAL` với giới hạn `LIMIT 1`:
```sql
		LEFT JOIN LATERAL (
			SELECT mb.master_schema, mb.master_table
			FROM cdc_system.master_binding mb
			WHERE mb.shadow_binding_id = sb.shadow_binding_id
			  AND mb.is_active = TRUE
			ORDER BY mb.updated_at DESC, mb.id DESC
			LIMIT 1
		) mb ON TRUE
```
Giải pháp này đảm bảo:
- Mỗi dòng `cdc_activity_log` sau khi mapping qua `shadow_binding` chỉ lấy duy nhất 1 bản ghi `master_binding` hoạt động mới nhất.
- Triệt tiêu hoàn toàn sự nhân bản dòng do phép join 1-N gây ra.
- Đảm bảo tính nhất quán của kết quả phân trang (trả về đúng số lượng bản ghi tương ứng với `LIMIT` và `OFFSET`).
