# Báo cáo Walkthrough - Tối ưu hóa Transmuter & Hạn chế Timeout

Chúng ta đã hoàn thành việc tái cấu trúc và tối ưu hóa hệ thống `TransmuterModule` để giải quyết triệt để lỗi `context deadline exceeded` (300s timeout) khi đồng bộ hóa (transmute) các bảng dữ liệu cực lớn (100M+ records) từ Shadow sang Master DB.

## Các Thay đổi Đã Thực hiện

### 1. Tối ưu hóa câu lệnh fetchShadowBatch
- **Bối cảnh**: Ban đầu, các câu lệnh chữa lành (heal) hoặc incremental sync chỉ truyền một danh sách nhỏ các `_source_id` nhưng lại bị ép dùng phân trang `_gpay_id` kèm theo `ORDER BY _gpay_id LIMIT 2000`. Điều này bắt PostgreSQL phải quét toàn bộ Index PK của bảng lớn.
- **Giải pháp**: Tách biệt logic truy vấn khi nhận danh sách `SourceIDs`. Sử dụng trực tiếp `WHERE _source_id IN (...)` không phân trang, loại bỏ `ORDER BY` và `LIMIT`, giúp truy vấn đạt độ phức tạp $O(1)$ nhờ chỉ mục index.

### 2. Tự động hóa tạo Chỉ mục (Auto-Indexing CONCURRENTLY)
- **Giải pháp**: Khi `fetchShadowBatch` được gọi, hệ thống kiểm tra sự tồn tại của chỉ mục `idx_<tablename>_source_id` trên cột `_source_id` của bảng Shadow. Nếu chưa có, kích hoạt goroutine chạy ngầm thực thi lệnh non-blocking:
  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_<tablename>_source_id ON <schema>.<tablename> (_source_id)`
  Giúp đảm bảo hiệu năng tối đa cho các câu lệnh healing / incremental sync.

### 3. Cơ chế checkpoint/resume (Lưu giữ trạng thái Full Sync)
- **Bối cảnh**: Tiến trình Full Sync chạy quét tuần tự hàng triệu dòng dữ liệu dễ bị đứt gãy do kết nối mạng, khởi động lại pod, hoặc timeout.
- **Giải pháp**:
  - Khi bắt đầu Full Sync, tải checkpoint `last_cursor_json` (chứa `last_gpay_id`) từ bảng `cdc_system.sync_runtime_state`.
  - Lưu checkpoint mới sau mỗi lô 2000 dòng được xử lý thành công.
  - Reset checkpoint về `{}` khi hoàn thành toàn bộ bảng.
  - Nếu xảy ra sự cố, lần chạy tiếp theo sẽ tiếp tục từ checkpoint mà không cần quét lại từ đầu.

### 4. Xử lý Bất đồng bộ ở tầng Handler (Dynamic Timeout)
- **Giải pháp**:
  - Chuyển tiến trình `svc.Run` trong `HandleTransmute` chạy dưới dạng background goroutine bất đồng bộ.
  - Gán dynamic timeout context riêng biệt:
    - **Incremental / Heal Sync**: Timeout là 30 phút.
    - **Full Sync**: Timeout là 24 giờ.
  - Đảm bảo OTel trace spans và activity logs (`cdc_activity_log`) được quản lý và ghi nhận đầy đủ trạng thái (running, complete, fail).

### 5. Cập nhật và mở rộng UI đối soát (`cdc-cms-web`)
- **Tùy chỉnh khoảng thời gian đối soát**: Cập nhật giá trị mặc định của custom date range thành 30 ngày gần nhất (thay vì 7 ngày) khi mở modal đối soát.
- **Tích hợp Tab đối soát đã xử lý**:
  - Cải tiến hook `useTableHistory` hỗ trợ custom `pageSize` (mặc định 30, truyền vào 100).
  - Tách đôi giao diện của Modal `ExecuteHealModal.tsx` thành component `Tabs` từ `antd`:
    - **Tab 1: Phiên chưa xử lý**: Hiển thị danh sách unhealed như ban đầu.
    - **Tab 2: Phiên đã xử lý**: Lọc từ table history với điều kiện `healed_at != null` (các phiên đã được heal thành công). Hiển thị chi tiết số lượng dòng đã heal, đã dọn dẹp và thời gian xử lý cụ thể.

## Kết quả Kiểm tra & Biên dịch

- **Backend**: Toàn bộ dự án `centralized-data-service` biên dịch thành công (`make build` pass). Chạy `go test -v ./internal/service/master/...` thành công 100%.
- **Frontend**: Dự án `cdc-cms-web` biên dịch thành công (`npm run build` pass). Không phát sinh bất kỳ lỗi cú pháp hay kiểu dữ liệu (TypeScript) nào.
