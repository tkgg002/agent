# Validation Plan: Phase 1.8 - CMS Frontend

Kế hoạch kiểm thử và xác minh tính đúng đắn của giao diện quản trị CDC.

## 1. Automated Tests (Unit & Build)

- `[ ]` **Build Check**: `npm run build` trong `cdc-cms-web`.
    - Kỳ vọng: Build thành công, không có lỗi TypeScript.
- `[ ]` **Lint Check**: `npm run lint`.
    - Kỳ vọng: Tuân thủ coding standard.

## 2. Manual Verification Scenarios

| ID | Scenario | Steps | Expected Result |
| :--- | :--- | :--- | :--- |
| `TC-1.8.1` | Login Flow | Đăng nhập với admin/admin. | Chuyển hướng Dashboard, lưu token đúng. |
| `TC-1.8.2` | API Port | Kiểm tra Network tab. | Các request bắt đầu bằng `localhost:8080` (thay vì 8090). |
| `TC-1.8.3` | Schema Approval | Thêm field vào Mongo -> Approve drift. | Postgres chạy `ALTER TABLE`, Worker nạp mapping mới. |
| `TC-1.8.4` | Table Register | Register table mới từ Airbyte. | Tạo thành công bảng trong registry và nạp schema. |
| `TC-1.8.5` | Standardize | Click nút Standardize. | CMS Publish NATS command, Worker thực thi (Check logs). |

## 3. Governance Audit (Rule 7)

- `[ ]` Kiểm tra `05_progress.md` có đầy đủ log hành động không.
- `[ ]` Kiểm tra `.env` có được loại bỏ khỏi `git` (nếu có dùng git).
