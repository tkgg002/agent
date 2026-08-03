# Kế hoạch Triển khai: Xoá Shadow & Xoá Master

## 1. Triển khai API Backend (Go & Fiber)
- Mở rộng `ports.ShadowBindingRepo` thêm 3 method: `GetByID`, `ListMasterBindingIDByShadowID`, và `DeleteShadowBinding`.
- Viết `DeleteShadowBindingCommand` & Handler:
  - Guard: Chặn nếu Shadow binding đang Active.
  - Cascade: Lấy toàn bộ Master bindings có shadow_binding_id khớp, thực hiện xoá master rules rồi xoá master binding.
  - Xoá shadow binding cuối cùng.
- Viết `DeleteMasterBindingCommand` & Handler:
  - Guard: Chặn nếu Master binding có trạng thái Approved.
  - Thực hiện xoá master rules và master binding.
- Đăng ký handlers, commands và routing trong `server.go` và `router.go`.

## 2. Triển khai Giao diện Frontend (React & Antd)
- Trong `TableRegistry.tsx` (Shadow Objects & Bindings):
  - Thêm cột Action với nút Xoá (Disabled nếu `is_active == true`).
  - Gắn modal `ConfirmDestructiveModal` để cảnh báo và thu thập lý do (reason).
  - Gửi `reason` và `Idempotency-Key` qua payload của request DELETE.
- Trong `MasterRegistry.tsx` (Masters):
  - Thêm nút Xoá ở cột Actions (Disabled nếu `schema_status === 'approved'`).
  - Gắn modal `ConfirmDestructiveModal` để cảnh báo và thu thập lý do.
  - Gửi `reason` và `Idempotency-Key` qua payload của request DELETE.
