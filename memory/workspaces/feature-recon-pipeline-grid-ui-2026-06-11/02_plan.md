# Plan: Recon Pipeline Grid UI Enhancement

## 1. Mục tiêu
- Loại bỏ cột Pipeline gộp chung và các cột rowSpan đơn lẻ.
- Tách thành 3 cột riêng biệt trong Ant Design Table: **Source**, **Shadow**, **Master** (hàng trên hiển thị table name, hàng dưới hiển thị schema name).
- Gom nhóm (Grouping) các hàng theo `Connector` & `Source DB` thành một cột duy nhất đặt lên đầu: **Source Connection & DB**.
- Biến đổi dữ liệu sang cấu trúc Tree Data (`children`) để hỗ trợ đóng/mở rộng nhóm:
  - Dòng cha (Group Header) hiển thị thông tin DB name, Connector, và số lượng tables con. Dòng cha gộp tất cả các cột thông tin bằng `colSpan`.
  - Mặc định các nhóm được thu gọn (ẩn các dòng con). Click nút mở rộng để hiển thị các dòng con (các pipeline chi tiết).

## 2. Các tệp cần chỉnh sửa
- [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)

## 3. Các bước thực hiện chi tiết
1. Định nghĩa kiểu dữ liệu `PipelineTableDataType` kế thừa `PipelineRow` và có thêm các thuộc tính `isGroupHeader`, `connector`, `db`, `tableCount`, `children`.
2. Trong component `ReconPipelineGrid`, sử dụng `useMemo` gom nhóm danh sách `pipelines` theo `sourceConnector` & `sourceDb`.
   - Mỗi nhóm tạo ra 1 dòng cha có `isGroupHeader: true` và mảng `children` chứa các dòng con có `children: undefined`.
3. Cấu trúc lại các cột (`columns`) của bảng:
   - Cột `Source Connection & DB`:
     - Nếu dòng cha: render DB Name + Connector Tag + Badge số lượng tables. Đặt `colSpan` bằng 10 để chiếm toàn bộ chiều rộng hàng.
     - Nếu dòng con: render trống.
   - Các cột khác (`Source`, `Shadow`, `Master`, records counts, Drift, Recon cuối lúc, Trạng thái):
     - Nếu dòng cha: trả về `colSpan: 0` trong `onCell`.
     - Nếu dòng con: render nội dung chi tiết.
4. Thêm prop `expandable={{ defaultExpandAllRows: false }}` cho component `Table` để mặc định ẩn các nhóm khi load trang.

## 4. Kế hoạch kiểm thử (Verification Plan)
- Chạy lệnh `npm run build` trên `cdc-cms-web` để kiểm tra biên dịch.
- Xác nhận giao diện hiển thị gọn gàng, nhóm Connector & DB hoạt động collapse/expand mượt mà, chính xác.

## 5. Bổ sung: Hiển thị trạng thái Onstream (Shadow) và Sync (Master)
- **Shadow**:
  - Dùng `useQuery` load danh sách `/api/v1/source-objects` để tra cứu trạng thái `is_active` (nếu chưa binding) hoặc `shadow_binding_is_active`.
  - Hiển thị Tag `Onstream: Bật` (màu `green`) hoặc `Onstream: Tắt` (màu `default`) tương ứng.
- **Master**:
  - Dùng `useQuery` load danh sách `/api/v1/masters` để lấy thông tin master registry và `is_active`.
  - Dùng `useQuery` load danh sách `/api/v1/schedules` để check cấu hình schedule (`post_ingest` hay `cron`).
  - Đối chiếu tên table đích để hiển thị Tag trạng thái tương ứng:
    - Bật sync realtime (`post_ingest` enabled): `Sync: Realtime` (màu `green`).
    - Bật sync hẹn giờ (`cron` enabled): `Sync: Hẹn giờ (cron_expr)` (màu `blue`).
    - Sync thủ công (chỉ chạy bằng tay): `Sync: Manual` (màu `orange`).
    - Tắt sync / Chưa được kích hoạt hoạt động: `Sync: Tắt` (màu `default`).

