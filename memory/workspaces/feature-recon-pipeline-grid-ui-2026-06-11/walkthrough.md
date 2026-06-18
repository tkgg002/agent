# Walkthrough: Recon Pipeline Grid UI Enhancement

## Các thay đổi chính

### Frontend: [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Thêm hàm helper `splitFQN` để phân tách Fully Qualified Name thành database/schema và table name.
- **Tách cột Pipeline thành 3 cột riêng biệt**:
  - **Source**: Hàng trên hiển thị table name của Source (in đậm), hàng dưới hiển thị schema name của Source.
  - **Shadow**: Hàng trên hiển thị table name của Shadow (code block), hàng dưới hiển thị schema name của Shadow.
  - **Master**: Hàng trên hiển thị table name của Master (code block hoặc `—`), hàng dưới hiển thị schema name của Master.
- **Gom nhóm bằng Tree Data và bổ sung tính năng Expand/Collapse**:
  - Gộp thông tin Connector và Source DB thành 1 cột duy nhất đặt lên đầu: **Source Connection & DB**.
  - Dòng cha (Group Header) sử dụng `colSpan={10}` để trải dài toàn bộ chiều rộng bảng, hiển thị tên DB, tag Connector và Badge số lượng tables con.
  - Tích hợp tính năng đóng/mở rộng nhóm thông qua prop `expandable` của Antd Table.
  - Mặc định (auto) tất cả các nhóm là ẩn (collapsed).
  - Tùy biến `onRow`: Click vào dòng cha (Group Header) để ẩn/mở rộng nhanh nhóm đó mà không làm mở Drawer drill-down; click vào dòng con (các pipeline chi tiết) mới hiển thị Drawer chi tiết.
- **Hiển thị trạng thái Onstream & Sync**:
  - **Shadow**: Tìm source object Registry tương ứng bằng `shadowName` và đối chiếu trạng thái active. Hiển thị Tag `on` (màu xanh lá) nếu active và `off` (màu xám) nếu inactive.
  - **Master**: Tìm master Registry tương ứng bằng `masterName` và tra cứu config active cùng với các schedules đang bật. Hiển thị Tag:
    - Realtime post-ingest: `Sync: Realtime` (màu xanh lá).
    - Hẹn giờ (cron expression): `Sync: Hẹn giờ (<cron>)` (màu xanh dương).
    - Sync thủ công: `Sync: Manual` (màu cam).
    - Sync bị tắt: `Sync: Tắt` (màu xám).

## Kết quả kiểm thử
- Đã chạy kiểm thử biên dịch thành công thông qua lệnh:
  ```bash
  npm run build
  ```
  Quá trình build diễn ra hoàn hảo không có lỗi TypeScript hay lỗi biên dịch frontend.
- **Cập nhật ngày 2026-06-12 (Sửa lỗi rớt dòng con khi đóng/mở group)**:
  - Loại bỏ hoàn toàn Tree Data mặc định của Antd Table (loại bỏ trường `children` trong data source) để tránh bug layout kinh điển của thư viện core `rc-table` khi kết hợp tree layout với `colSpan` động của dòng cha.
  - Sử dụng state `expandedKeys` tự quản lý để phẳng hóa dữ liệu (Flat Data). Khi một group được mở rộng, các child rows được chèn trực tiếp vào mảng phẳng phía sau dòng cha. Khi đóng group, chúng được lọc bỏ hoàn toàn khỏi mảng phẳng.
  - Tự render icon đóng/mở (`RightOutlined` / `DownOutlined`) ở cột đầu tiên của Group Header.
  - Cập nhật `onRow` click handler để toggle đóng/mở một cách nhất quán (không bị double event trigger do click bubble từ nút mũi tên cũ).
  - Đã xác thực biên dịch TypeScript bằng `npx tsc -b` thành công 100%.

