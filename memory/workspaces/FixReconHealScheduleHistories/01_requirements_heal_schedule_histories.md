# Yêu cầu Hotfix: Sửa lỗi tính năng Heal cho bảng schedule_histories (Segment A)

## 1. Bối cảnh
Khi kích hoạt hành động "Heal" từ giao diện quản trị (`ExecuteHealModal`) cho bảng `schedule_histories` trong Segment A (`source_shadow`), hệ thống phản hồi thành công nhưng số lượng bản ghi được xử lý (`rows_affected`) luôn bằng 0. Dữ liệu lệch trong báo cáo đối soát vẫn tồn tại.

## 2. Nguyên nhân gốc rễ (Root Cause)
- Khi xử lý báo cáo đối soát thuộc Segment A, handler `ExecuteHealHandler` thực hiện gán `rpt.TargetTable = rpt.ShadowSchema + "." + rpt.ShadowTable` (ví dụ: `shadow_testss.schedule_histories`).
- Hàm `resolveTargetTableConfig` trong `recon_base_handler.go` nhận tham số `targetTable` chứa đầy đủ schema và thực hiện tra cứu trực tiếp cấu hình từ metadata và database registry qua `GetTableConfig(targetTable)` hoặc `GetByTargetTable(ctx, targetTable)`.
- Trong cơ sở dữ liệu và registry cấu hình, bảng đích được đăng ký chỉ với tên bảng thuần túy: `schedule_histories` (không kèm schema). Do đó, phép so khớp trực tiếp thất bại.
- Logic kiểm tra prefix `ShadowPrefix` (`shadow_`) cũng không khớp vì schema thực tế là `shadow_testss.` (chứa tiền tố khác). Hàm trả về `nil`, dẫn tới logic heal thoát sớm mà không thực thi xử lý bản ghi nào.

## 3. Yêu cầu chi tiết
- **H1**: Cập nhật hàm `resolveTargetTableConfig` trong `centralized-data-service/internal/handler/recon/recon_base_handler.go`.
- **H2**: Trích xuất tên bảng thuần túy (loại bỏ schema, ví dụ: từ `shadow_testss.schedule_histories` lấy `schedule_histories`) trước khi thực hiện tra cứu metadata và database registry.
- **H3**: Đảm bảo logic fallback tra cứu bằng tên bảng thuần túy nếu tra cứu bằng tên đầy đủ thất bại để đảm bảo tương thích ngược cho cả hai định dạng (có schema và không schema).
- **H4**: Tiến hành chạy build và kiểm tra độ chính xác của logic sửa đổi bằng cách chạy các integration tests có sẵn.
