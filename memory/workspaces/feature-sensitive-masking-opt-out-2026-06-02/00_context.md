# 00_context.md — Context & Scope (Updated for JSON masking)

## Goal
Điều chỉnh cơ chế Sensitive Masking trên cả Backend (mã hoá nested JSON) và Frontend (Khôi phục cột Mask Strategy).

1. **Backend (Nested JSON Masking)**:
   - Thêm thuật toán `json_mask` cho các cột lưu trữ dữ liệu dạng JSON string hoặc JSON array (như `passwordHistory`).
   - Khi áp dụng thuật toán này, CDC Worker sẽ duyệt đệ quy qua các key-value của tài liệu JSON.
   - Nếu bất kỳ key con nào thuộc danh sách nhạy cảm (`sensitive_fields` hoặc default keywords), worker tự động mã hoá key đó bằng thuật toán tương ứng (ví dụ: `password` -> `hmac`).
   - Xử lý cả trong `_raw_data` và business columns.

2. **Frontend (UI Restore)**:
   - Khôi phục cột **Mask Strategy** bên cạnh cột **Sensitivity** (`is_sensitive_field`).
   - Hiển thị Switch cho **Sensitivity**. Khi tắt, cột **Mask Strategy** bị disabled và hiển thị `none`.
   - Khi bật, cột **Mask Strategy** mở ra với các option: `hmac`, `aes_gcm`, `json_mask`. Không được để rỗng/null, mặc định sẽ là `hmac` (hoặc cấu hình global).

## Scope
- Dịch vụ Backend: `centralized-data-service` (nơi xử lý raw_data walker) và `cdc-cms-service`.
- Giao diện Frontend: `cdc-cms-web`.

## Governance Check
- Tuân thủ quy trình Workspace-First. Không phát hiện lỗi vi phạm quy trình Governance nào.
