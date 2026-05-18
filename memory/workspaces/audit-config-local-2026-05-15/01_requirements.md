# 01_requirements — Audit config-local.yml

## Yêu cầu chi tiết từ user

1. Kiểm tra `centralized-data-service/config/config-local.yml`.
2. So với flow hiện tại: key nào còn dùng / key nào không dùng.
3. Ràng buộc:
   - Đọc lessons trước.
   - Theo core `/agent` (đọc GEMINI.md).
   - Chỉ làm đúng yêu cầu, KHÔNG cheat DB hoặc sửa config để fit kết quả.
   - Plan rõ ràng, có giải pháp cụ thể.
   - Report dựa trên kết quả tính toán thực tế (grep/read), review cẩn thận, note các file thay đổi.
   - Cuối phải kiểm tra service work.
   - Tạo file `report_*.md`.

## Definition of Done

- [x] Bảng key-by-key xác định: parse được? caller nào? dead/legacy/active?
- [x] Đối chiếu code reference (file:line) cho mỗi kết luận "active".
- [x] Đối chiếu code reference cho mỗi kết luận "dead" (chứng minh KHÔNG có caller).
- [x] File `report_config_local_audit_2026-05-15.md` ở workspace.
- [x] Build verification (cấu hình hiện tại boot được không) — vì đây là audit, không sửa code, nên verification = read-only review.
- [x] Cập nhật `05_progress.md` append-only.

## Constraints

- KHÔNG sửa code Go (.go) — đây là audit, Brain Code Prohibition.
- KHÔNG sửa file YAML (chưa được user authorize).
- Output chỉ là markdown report + workspace docs.
