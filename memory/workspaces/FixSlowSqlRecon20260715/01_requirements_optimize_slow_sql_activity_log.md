# Yêu cầu Tối ưu hóa SQL cdc_activity_log (SLOW SQL)

## 1. Vấn đề
Màn hình nhật ký hoạt động (`cdc_activity_log`) xuất hiện cảnh báo `SLOW SQL >= 200ms` khi thực hiện truy vấn đếm tổng số bản ghi (`SELECT COUNT(*)`). 
Nguyên nhân là do câu lệnh đếm hiện tại đang sử dụng chung mệnh đề `FROM` + `LEFT JOIN LATERAL` của câu lệnh truy vấn dữ liệu chi tiết, dẫn đến việc thực thi các phép JOIN không cần thiết và tốn nhiều chi phí khi quét toàn bộ bảng.

## 2. Mục tiêu
- Tối ưu hóa hiệu năng câu lệnh `SELECT COUNT(*)` trong hàm `ListActivity` của `activity_log_read_repo_gorm.go`.
- Rút ngắn thời gian truy vấn đếm xuống dưới 200ms.
- Bảo toàn logic nghiệp vụ: Chỉ thực hiện `JOIN` khi các điều kiện lọc yêu cầu dữ liệu từ bảng liên kết (`shadow_binding` hoặc `source_object_registry`). Nếu không lọc theo các bảng này, thực hiện đếm trực tiếp trên bảng `cdc_activity_log`.

## 3. Các tiêu chí hoàn thành (Definition of Done)
- [ ] Tách biệt câu countQuery thành phiên bản tối giản khi không có bộ lọc bảng liên kết.
- [ ] Chạy thành công toàn bộ test suite hiện tại.
- [ ] Đo đạc và ghi nhận thời gian thực thi (Cũ vs Mới).
