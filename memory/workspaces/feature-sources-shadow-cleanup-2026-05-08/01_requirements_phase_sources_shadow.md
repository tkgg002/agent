# Requirements — Sources / Shadow Cleanup

## Mục tiêu

1. Liệt kê các DB/schema trong `cdc_system` đang được sử dụng bởi CMS/worker flow hiện tại.
2. Ẩn `flow1` khỏi UI.
3. Đổi tên tab/route hiển thị từ `registry` sang `shadow`.
4. Trang `sources` phải cho tạo connection mới cho 3 loại DB:
   - MongoDB
   - MySQL
   - PostgreSQL
5. Mỗi loại DB có form cấu hình riêng:
   - URL/host/port/database
   - username/password khi phù hợp
6. Sau khi thêm phải hiển thị danh sách connection/source.
7. Có chức năng edit config và update lại để connector active/update theo config mới.
8. Tự test trên browser các chức năng chính của 3 tab liên quan.

## Definition of Done

- `npm run build` pass cho FE.
- `go test ./...` pass cho CMS service hoặc ít nhất không gây regression mới trong scope sửa.
- Browser test xác nhận:
  - tab Sources mở được
  - tạo form cho 3 loại DB hiện đúng
  - edit/update config hoạt động theo UI/response
  - tab Shadow mở được
  - `flow1` không còn xuất hiện trong menu chính
