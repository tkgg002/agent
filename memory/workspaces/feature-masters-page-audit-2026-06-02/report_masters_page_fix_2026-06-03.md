# Report: Master Mapping Registry Isolation & Schedule Fix — 2026-06-03

Báo cáo chi tiết các file đã thay đổi, các chức năng được sửa đổi và kết quả nghiệm nghiệm trên Master Registry mapping infrastructure.

## Những File Đã Thay Đổi & Số Dòng Code
### 1. Backend (cdc-cms-service)
- [MODIFY] [transmute_schedule_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/transmute_schedule_handler.go) (+6, -2 lines)
- [MODIFY] [master_mapping_rule_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master_mapping_rule_handler.go) (+40, -18 lines)
- [MODIFY] [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go) (+2, -2 lines)

### 2. Frontend (cdc-cms-web)
- [NEW] [MasterMappingFieldsPage.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx) (+250 lines)
- [MODIFY] [MasterRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx) (+15, -2 lines)
- [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx) (+2, -2 lines)
- [MODIFY] [App.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/App.tsx) (+5, -1 lines)

*Tổng số dòng thay đổi trong phiên này:* ~320 lines mới (chủ yếu là trang quản lý Mapping Master độc lập và cơ chế lọc/truy vấn sạch từ Shadow DB).

---

## Chi Tiết Thay Đổi
### 1. Khắc Phục Lỗi Transmute Schedule Create (500 Internal Error)
- **Vấn đề**: Khi tạo schedule cho master table chưa tồn tại (chưa được approve / chưa sync), backend ném lỗi 500 `internal_error` do không check sự tồn tại của master binding trong control plane.
- **Giải Pháp**: Thêm kiểm tra binding tồn tại trước khi tạo schedule. Nếu master binding không tồn tại, trả về HTTP 404 `master table does not exist` thay vì 500.
- **Kết Quả**: API POST `/api/v1/schedules` hoạt động ổn định và trả về đúng mã lỗi client-side (404) khi master table không tồn tại.

### 2. Cô Lập Master Mapping Rules Khỏi Cột Hệ Thống
- **Vấn đề**: Khi operator tạo thủ công các mapping rule hoặc chạy cơ chế `Flatten` trên các cột JSON/JSONB, các cột hệ thống của Shadow table (ví dụ: `_id`, `progress`, `status`, `params`, `error`, `_raw_data`, `_synced_at`, ...) bị leak sang Master Registry (`mapping_rule_master`).
- **Giải Pháp**:
  - Tích hợp hàm blacklist helper `isSystemColumn` vào cả 2 luồng: `Save` (thao tác lưu thủ công) và `Flatten` (thao tác tự động bóc tách schema JSON).
  - Trực tiếp loại bỏ các cột hệ thống ra khỏi danh sách kết quả trả về của API `Flatten`, đảm bảo schema master luôn "sạch nghiệp vụ".
- **Kết Quả**: Khi gọi API `/api/v1/master-mapping-rules/flatten` cho cột `params`, kết quả trả về chỉ gồm 3 cột nghiệp vụ thực sự (`params_exporttype`, `params_datefr`, `params_dateto`), toàn bộ cột hệ thống bị loại bỏ hoàn toàn.

### 3. Sửa Lỗi SQL Syntax & Sai Database Connection Trong Flatten API
- **Vấn đề 1 (SQL Syntax 42601)**: Khi chạy API `Flatten`, query dữ liệu mẫu sử dụng `fmt.Sprintf` với `gorm.Expr` dẫn đến việc convert struct `gorm.Expr` sang chuỗi lỗi cú pháp.
- **Giải Pháp 1**: Viết hàm `sanitizeIdentifier` để chuẩn hóa và lọc sạch tên schema/table/field, sau đó dùng truy vấn parameterized dạng placeholder an toàn.
- **Vấn đề 2 (Sai DB Connection)**: Query sample data của shadow table được thực thi nhầm trên database control plane `h.db` (`cdc_dw` cổng 5433) thay vì database shadow data plane (`cdc_shadow` cổng 5436), dẫn đến lỗi `relation does not exist` (42P01).
- **Giải Pháp 2**: Inject `shadowDB` connection vào `MasterMappingRuleHandler` (thông qua `server.go`) và dùng `h.shadowDB` để query mẫu trực tiếp từ shadow table.

### 4. Triển Khai Master Mapping Rules Management UI
- **Giao Diện Mới**: Tạo trang `MasterMappingFieldsPage.tsx` độc lập hỗ trợ đầy đủ các tính năng:
  - Xem danh sách mapping của Master table (Source Field, Target Column, Rule Type, Status...).
  - Thêm mới mapping rule thủ công qua Modal Form.
  - Tự động tách nhỏ JSON (Flatten) với nút bấm trực quan.
  - Xóa/cập nhật trạng thái các rules.
- **Tích Hợp Điều Hướng**: Thêm nút bấm `Mappings` trong cột Action của bảng `MasterRegistry.tsx` dẫn trực tiếp tới trang quản lý mapping tương ứng.

---

## Kết Quả Kiểm Chứng & Build
1. **CMS Service (Backend)**:
   - Chạy `go build ./...` thành công.
   - Server restart mượt mà. Test API flatten trả về kết quả chính xác, sạch sẽ.
2. **CMS Web (Frontend)**:
   - `npm run build` và `tsc -b` hoàn thành 100% không có lỗi TypeScript compiler.
