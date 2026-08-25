# Yêu Cầu: Sửa Lỗi Phân Trang API Activity Log

## 1. Vấn Đề
Khi gọi API `GET /api/activity-log?page=2&page_size=30`, kết quả trả về không chính xác từ trang 2 trở đi.

## 2. Nguyên Nhân
- Trong `activity_log_read_repo_gorm.go`, hàm `ListActivity` thực hiện enrich dữ liệu bằng cách thực hiện `LEFT JOIN` với bảng `cdc_system.master_binding mb` dựa trên `shadow_binding_id`:
  ```sql
  LEFT JOIN cdc_system.master_binding mb
    ON mb.shadow_binding_id = sb.shadow_binding_id
   AND mb.is_active = TRUE
  ```
- Quan hệ giữa `shadow_binding` và `master_binding` là 1-N (một shadow table có thể map tới nhiều master table).
- Việc `LEFT JOIN` thông thường mà không giới hạn số lượng row trả về từ `master_binding` làm nhân bản số dòng (row amplification) trong tập kết quả trả về từ database.
- Do đó, dù subquery `innerQuery` phân trang chính xác bằng `OFFSET` và `LIMIT`, kết quả sau khi join sẽ bị phình to (ví dụ: yêu cầu 30 dòng nhưng database trả về 34 dòng do có 4 dòng bị nhân bản). Điều này gây lỗi hiển thị và phân trang không đồng nhất ở client.

## 3. Yêu Cầu Chi Tiết
- Sửa join với `master_binding` thành `LEFT JOIN LATERAL` kèm `LIMIT 1` để đảm bảo mỗi bản ghi `cdc_activity_log` chỉ tương ứng với tối đa 1 bản ghi `master_binding` hoạt động, triệt tiêu hoàn toàn sự nhân bản dòng.
- Đảm bảo trật tự sắp xếp trong `LATERAL` join lấy bản ghi mới nhất (`ORDER BY mb.updated_at DESC, mb.id DESC LIMIT 1`).
- Chạy unit test để kiểm tra việc biên dịch và hoạt động.
