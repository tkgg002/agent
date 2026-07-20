# Kế Hoạch Triển Khai: Tối ưu hóa UI Đối Soát & Sửa lỗi Quyền NATS (Reconciliation UI & NATS Permissions Plan)

Dưới đây là kế hoạch chi tiết nhằm cải tiến giao diện đối soát trong ứng dụng quản trị (`cdc-cms-web`), giải quyết lỗi phân quyền NATS (Permissions Violation) gây timeout khi thực hiện Chữa lành đối soát.

## User Review Required

> [!IMPORTANT]
> - **Ẩn Segment Selector:** Giao diện chọn chặng trong modal "Bắt đầu đối soát" sẽ được ẩn đi. modal sẽ hoàn toàn tự động lấy `initialSegment` được truyền từ chặng tương ứng của dòng được click.
> - **Chế độ đối soát mặc định:** Chế độ đối soát sẽ mặc định chọn `Hot Mode (2h)` thay vì `Smoke (7d)` để tăng tốc độ đối soát chặng realtime.
> - **Ẩn Deep Check:** Tùy chọn `Deep Check (Quét toàn collection)` sẽ được ẩn bằng `display: none` trên UI để tránh người dùng click nhầm làm suy giảm hiệu năng DB, nhưng giữ lại code để có thể bật lại sau này khi cần.
> - **Chỉ hiển thị Smoke Check trên Biểu đồ:** Biểu đồ "Biến động số lượng theo phiên recon" sẽ chỉ hiển thị các phiên đối soát thuộc loại `Smoke Check` (lọc `check_type === 'smoke'` hoặc `check_type === 'segment_b_smoke'`).
> - **Sửa Quyền NATS (`_INBOX.>`):** Cấp quyền `publish` tới các topic reply dạng `_INBOX.>` trong NATS ACL cho các user `cdc_worker`, `cms_service` và `debezium` để tránh lỗi Permissions Violation và timeout.

---

## Proposed Changes

### Component: `ConfirmDestructiveModal.tsx`

#### [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
- Cập nhật state `checkMode` mặc định thành `'2h'`.
- Cập nhật `useEffect` khởi tạo state khi modal mở:
  - Thiết lập `checkMode` thành `'2h'`.
  - Thiết lập `customRange` mặc định lùi 2 giờ từ `endTime` (`endTime.subtract(2, 'hour')`).
- Ẩn lựa chọn Deep Check bằng cách gán `style={{ display: 'none' }}` lên Radio component.
- Ẩn khối UI chọn chặng đối soát (`Chặng đối soát (Segment)`) bằng cách bọc điều kiện hoặc ẩn style.

### Component: `ExecuteHealModal.tsx`

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Tận dụng prop `segment` được truyền vào.
- Thực hiện lọc dữ liệu `reports` (tab Phiên chưa xử lý) và `healedReports` (tab Phiên đã xử lý):
  - Với chặng A (`segment === 'source_shadow'`): Lọc `r.segment === 'source_shadow' || !r.segment`.
  - Với chặng B (`segment === 'shadow_master'`): Lọc `r.segment === 'shadow_master'`.
  - Nếu không có segment: Giữ nguyên toàn bộ.

### Component: `ReconPipelineGrid.tsx`

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Lọc `chartData` chỉ hiển thị các phiên có `check_type === 'smoke'` hoặc `check_type === 'segment_b_smoke'`.
- Lọc `yDomain` tương tự để trục Y phản ánh đúng dải giá trị của các phiên Smoke Check.

### Config: `nats-server.conf`

#### [MODIFY] [nats-server.conf](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/deployments/nats/nats-server.conf)
- Thêm `_INBOX.>` vào danh sách `publish` permissions cho các user `cdc_worker`, `cms_service` và `debezium`.

---

## Verification Plan

### Automated Tests
- Chạy linter kiểm tra cấu trúc quy trình:
  ```bash
  python3 agent/tooling/verify_governance.py
  ```
- Kiểm tra tính đúng đắn về kiểu và biên dịch của code React:
  ```bash
  npm run build --prefix cdc-cms-web
  ```

### Manual Verification
- Mở modal "Bắt đầu đối soát" trên UI chặng A/chặng B, kiểm tra xem chặng đối soát selector đã biến mất, Deep Check đã ẩn, và mặc định chọn Hot Mode 2h.
- Mở modal "Chữa lành đối soát" ở từng chặng, kiểm tra danh sách phiên chưa xử lý/đã xử lý có được lọc chính xác theo chặng tương ứng hay không.
- Mở Drawer chi tiết của một pipeline, kiểm tra biểu đồ "Biến động số lượng theo phiên recon" xem đã lọc và chỉ hiển thị các điểm dữ liệu của phiên Smoke Check chưa.
- Thực hiện bấm nút Chữa lành (Heal) trên giao diện CMS để xác nhận không còn xảy ra lỗi timeout hay NATS Permissions Violation trong log của worker.
