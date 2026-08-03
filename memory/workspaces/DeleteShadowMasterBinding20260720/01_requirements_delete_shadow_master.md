# Yêu cầu: Xoá Shadow & Xoá Master

## 1. Yêu cầu nghiệp vụ
- Xoá Shadow Binding: Xoá shadow binding và cascade xoá toàn bộ master bindings cùng các mapping rules (cloned rules) đi kèm.
- Xoá Master Binding: Chỉ xoá master binding và mapping rules đi kèm, không ảnh hưởng đến shadow binding.

## 2. Ràng buộc & Cảnh báo an toàn (Guards)
- Shadow Binding: Không được xoá khi đang ở trạng thái active (`is_active = true`). Phải tắt Active trước khi xoá.
- Master Binding: Không được xoá khi đang ở trạng thái approved (`schema_status = 'approved'`). Phải Reject trước khi xoá.
- Cần có xác nhận xác thực và yêu cầu lý do (reason >= 10 ký tự) để audit.
