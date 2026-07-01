# Context: ShadowTransformAction

## Shadow Transform Action
- Nhu cầu: Tại giao diện `http://localhost:5173/shadow` ở phần "Shadow Actions" hoặc các trang quản lý liên quan, người dùng cần một nút bấm "transform" (Transform Data) để kích hoạt tức thì tiến trình dịch chuyển/transform dữ liệu từ cột `_raw_data` ra các cột mới được thêm vào.
- Hiện tại: Cơ chế transform này chỉ tự động kích hoạt khi chạy job hoặc trong luồng provisioning. Khi cấu trúc schema thay đổi và cột mới được thêm vào, dữ liệu lịch sử ở các dòng cũ trong bảng shadow chưa được điền giá trị cho cột mới trừ khi chạy lại job hoặc transform toàn bộ. Người dùng muốn kích hoạt tiến trình này bằng cách bấm nút "transform" thủ công ngay trên giao diện CMS.

## Mục tiêu
1. Xác định trang frontend tương ứng chứa "Shadow Actions" (dự kiến trong `cdc-cms-web`).
2. Xác định các API endpoint hiện có trong backend (`cdc-cms-service` và/hoặc `centralized-data-service`) thực hiện việc transform/backfill dữ liệu cho shadow table.
3. Nếu chưa có endpoint trực tiếp, tạo endpoint mới để trigger tác vụ transform dữ liệu.
4. Tích hợp nút "Transform" trên giao diện web và gọi API tương ứng.
