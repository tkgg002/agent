# Walkthrough: Sửa lỗi trùng lặp (duplicate) dòng pipeline trong ReconPipelineGrid

## Các thay đổi đã thực hiện

### 1. Frontend (`cdc-cms-web`)
#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Bổ sung logic lọc trùng (deduplicate) ở đầu hàm `buildPipelines`. Sử dụng một `Map` theo khóa `${schema}::${table}::${segment}` để giữ lại bản ghi có `checked_at` mới nhất cho mỗi pipeline segment, loại bỏ các bản ghi cũ của phiên bản legacy (chưa có `shadow_schema` stamp).
- Cải tiến logic mapping chéo giữa Segment A (`source_shadow`) và Segment B (`shadow_master`) bằng cách so sánh cả `shadow_schema` và `shadow_table` (nếu có ở cả hai bên) thay vì chỉ so sánh bằng `target_table`, đảm bảo hiển thị đúng đắn kể cả khi có các bảng trùng tên ở nhiều schema shadow khác nhau.

## Kiểm thử & Xác thực
- **Tĩnh**: Đã chạy thành công `npx tsc -b` trong thư mục `cdc-cms-web` để đảm bảo code sạch lỗi cú pháp và Type-safe.
- **Dữ liệu**: Cơ chế lọc trùng này đảm bảo rằng các dòng trùng lặp do dữ liệu cũ trong DB sẽ bị triệt tiêu 100% khi hiển thị trên grid.
