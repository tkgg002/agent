# Report: Shadow Transform Action

Báo cáo kết quả audit và danh sách thay đổi chi tiết của tác vụ thêm nút **Transform** vào **Shadow Actions**.

## Lý do thay đổi (Reason for Change)
Cho phép người dùng trigger tiến trình dịch chuyển/transform dữ liệu lịch sử từ cột `_raw_data` của shadow table sang các cột cấu trúc đích thủ công bất kỳ lúc nào trực tiếp trên giao diện CMS mà không cần chờ chạy lại toàn bộ job hay qua luồng provisioning tự động.

## Danh sách tệp thực tế đã thay đổi & Số dòng code thay đổi (Git Diff Audit)

| Tên tệp (File Name) | Trạng thái (Status) | Số dòng thêm (Additions) | Số dòng xóa (Deletions) | Chi tiết thay đổi |
| :--- | :--- | :--- | :--- | :--- |
| [source_object_actions_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/source_object_actions_handler.go) | Modified | 59 | 0 | Thêm ports.Publisher dependency và endpoint `TransformV2` gọi publish lệnh transform qua NATS |
| [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go) | Modified | 1 | 0 | Đăng ký POST route `/v1/source-objects/:id/transform` |
| [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go) | Modified | 1 | 1 | Inject dependency `natsPublisher` vào handler khởi tạo |
| [source_object_actions_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/api/source_object_actions_handler_test.go) | New | 143 | 0 | Unit tests kiểm thử thành công/lỗi cho endpoint `TransformV2` |
| [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx) | Modified | 39 | 0 | Thêm hàm `handleTransform` và nút bấm Transform có Modal xác nhận |

## Kiến trúc & Pattern Tuân thủ (Architecture & Pattern Compliance)
- **Hexagonal Architecture**: Tất cả các thao tác tương tác của backend với cơ sở hạ tầng (NATS) đều đi qua cổng trừu tượng (`ports.Publisher`), không trực tiếp sử dụng client NATS thô hay logic cheat DB nào.
- **Idempotency & Traceability**: Yêu cầu frontend truyền `Idempotency-Key` cùng trace headers chuẩn hóa và backend thực hiện log async tương thích với cơ chế kiểm soát chất lượng & audit logs của hệ thống.
- **V2 Scope Resolution**: Sử dụng đúng hàm `resolveDispatchScope` kế thừa việc phân giải chính xác binding đang active và hỗ trợ query parameter `binding_id` từ UI.

## Kết quả kiểm thử tự động & thủ công
- Unit tests backend: **Passed** (100% thành công qua lệnh `go test ./test/... -count=1 -short`).
- Compile frontend: **Succeeded** (TypeScript build thành công qua `npm run build`).
- UI/UX Browser: **Verified** (Nút Transform hiển thị chuẩn thiết kế, click ra Modal xác nhận và phản hồi đúng nghiệp vụ).
