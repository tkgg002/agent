# Plan: Table Registry UI Enhancement

## 1. Mục tiêu
- Thêm style `padding-bottom: 5px` cho thẻ `div` bao quanh `Switch` (trạng thái hoạt động của Table Registry) tại dòng 857 trong `TableRegistry.tsx`.
- Loại bỏ toàn bộ `ant-space` và cấu trúc `<Space>` trong component `AsyncRowActions`.
- Tích hợp trạng thái chạy và thành công trực tiếp vào nút "Quét field" (Scan fields):
  - Trạng thái đang chạy (`isScanning`): nút có màu xanh dương (primary border/color) và hiển thị icon loading (như hiện tại).
  - Trạng thái hoàn thành (`isSuccess`): nút hiển thị icon dấu check (`<CheckOutlined />`) và border color + text màu xanh lá.
  - Loại bỏ component `<DispatchStatusBadge>` khỏi hiển thị để tránh giao diện rườm rà.

## 2. Các tệp cần chỉnh sửa
- [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx)

## 3. Các bước thực hiện chi tiết
1. Import `CheckOutlined` từ `@ant-design/icons`.
2. Sửa lại `AsyncRowActions`:
   - Xác định trạng thái đang chạy: `isScanning = scan.isPending || scan.state.status === 'dispatching' || scan.state.status === 'accepted' || scan.state.status === 'running'`.
   - Xác định trạng thái hoàn tất: `isSuccess = scan.state.status === 'success'`.
   - Thiết lập style và icon cho nút "Quét field" tương ứng:
     - Nếu `isSuccess`:
       - `style = { borderColor: '#52c41a', color: '#52c41a' }`
       - `icon = <CheckOutlined />`
     - Nếu `isScanning`:
       - `style = { borderColor: '#1677ff', color: '#1677ff' }` (hoặc màu xanh dương mặc định của Antd, ví dụ `#1677ff`)
       - `icon = undefined` (Ant Design Button sẽ tự động chèn loading spinner khi `loading={true}`)
     - Bình thường:
       - `style = {}`
       - `icon = <SearchOutlined />`
   - Loại bỏ các component `<Space>` của Ant Design, thay bằng thẻ `div` với style cơ bản (hoặc Fragment `<>`) để tránh sinh ra class `ant-space`.
   - Ẩn Tag `<DispatchStatusBadge>` để tối giản giao diện.
3. Tìm thẻ `div` bọc `Switch` (khoảng dòng 857) và thêm style `paddingBottom: '5px'`.

## 4. Kế hoạch kiểm thử (Verification Plan)
- Run dev server (`npm run dev`) để xem giao diện có hoạt động đúng và không có lỗi biên dịch.
- Thực hiện kiểm tra thủ công giao diện:
  - Xem Switch đã có padding-bottom 5px chưa.
  - Xem nút Quét field hiển thị thế nào.
  - Click Quét field, xem trạng thái đang chạy (màu xanh dương + loading) và sau khi hoàn tất (màu xanh lá + checkmark).
