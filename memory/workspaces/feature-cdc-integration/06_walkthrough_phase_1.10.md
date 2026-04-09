# Walkthrough: Phase 1.10 — CDC Schema Scan & UI Stabilization

Tôi đã hoàn tất các hạng mục trong Phase 1.10, tập trung vào việc tự động hóa đăng ký bảng/field và khắc phục các lỗi UI nghiêm trọng.

## Các thay đổi chính

---

### 1. Backend: Tự động hóa quét Schema & Fields

Tôi đã triển khai hai tính năng quan trọng để loại bỏ việc nhập liệu thủ công:

- **Scan Source**: Tự động lấy danh sách bảng từ Airbyte Source và đăng ký vào Registry nếu chưa có.
  - Endpoint: `POST /api/registry/scan-source?source_id=xxx`
- **Scan Fields**: Quét toàn bộ fields của một bảng từ Airbyte Discovery API.
  - Endpoint: `POST /api/registry/:id/scan-fields`
  - Tự động phân loại: Các trường hệ thống (`_raw_data`, `_source`, v.v.) được đặt trạng thái `approved`, các trường nghiệp vụ mới được đặt `pending` để User duyệt.

> [!TIP]
> Hệ thống hiện tại đã bỏ hoàn toàn bảng `pending_fields`. Mọi logic duyệt field hiện nay nằm trực tiếp trong `cdc_mapping_rules`, giúp quản lý tập trung và nhất quán.

---

### 2. Frontend: Khắc phục lỗi và Nâng cấp UI

#### [FIX] Queue Monitoring Crash
- Khắc phục lỗi màn hình đen (crash) khi truy cập dashboard mà Worker stats đang rỗng hoặc `pool_size = 0`.
- Thêm các guard check phép chia cho không tại [QueueMonitoring.tsx](file:///Users/trainguyen/Documents/work/cdc-cms-web/src/pages/QueueMonitoring.tsx).

#### [UPDATE] Table Registry & Schema Changes
- **Table Registry**: 
  - Tiêu đề căn lề trái.
  - Bổ sung cột: `Source DB`, `Connection ID`, `Created At`.
  - Thêm nút **Scan Fields** trực tiếp trên từng dòng để quét nhanh cấu trúc bảng.
- **Schema Changes**:
  - Chuyển sang lấy dữ liệu `pending` trực tiếp từ bộ quy tắc mapping rules.
  - Cập nhật giao diện duyệt field đồng bộ với cấu trúc dữ liệu mới.

---

## Kết quả kiểm tra

### Backend Build
```bash
# Đã chạy build thành công
cd cdc-cms-service && go build ./...
# Result: Success (No errors)
```

### UI Verification
- [x] `/registry`: Hiển thị đúng các cột mới, nút Scan Fields hoạt động tốt.
- [x] `/queue`: Truy cập bình thường kể cả khi stats chưa có dữ liệu.
- [x] `/schema-changes`: Hiển thị danh sách field chờ duyệt từ database mapping rules.

---

## Hướng dẫn tiếp theo

Bây giờ bạn có thể:
1. Truy cập **Table Registry**, chọn một Database Source để **Scan Source**.
2. Với các bảng mới, bấm **Scan Fields** để hệ thống tự nhận diện cấu trúc.
3. Vào **Schema Changes** để duyệt các field mới phát hiện.

Lưu ý: Mọi tiến trình đã được cập nhật vào [05_progress.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-integration/05_progress.md) và [task_1.10.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/feature-cdc-integration/task_1.10.md).
