---
description: Kiểm tra việc tuân thủ các quy tắc quản trị (Rule 7) và duy trì Bộ não dự án.
---

# Governance Audit Workflow (Rule 7)

Workflow này đảm bảo Agent luôn tuân thủ **Quy tắc số 7 (Rule 7)** về việc duy trì "Bộ não dự án".

## 📋 Checklist Kiểm tra (Mandatory)

### 1. Metadata Integrity
- [ ] `05_progress.md` sử dụng định dạng: `[YYYY-MM-DD HH:mm] [Agent:Model] Action`.
- [ ] KHÔNG có Metadata giả mạo hoặc thiếu thông tin phiên làm việc.

### 2. Prefix Registry (00-10)
- [ ] Đã xác nhận file tương ứng với mã số hiện tại (VD: `03` cho Tech Design, `08` cho Tasks).
- [ ] Xóa bỏ phụ thuộc vào Shadow Documents (Artifacts, Chat context).

### 3. Immutable Log Policy
- [ ] Cấm dùng `Overwrite: true` cho Memory files.
- [ ] Mọi hành động sửa chữa lỗi quy trình đều đi kèm RCA trong `05_progress.md`.

### 4. Global Patterns
- [ ] Cập nhật `lessons.md` theo format: `Global Pattern [A does B to X] → Result Y. Đúng: [Flow]`.

## 🚀 Thực thi
Chạy workflow này:
1. Đọc 10 dòng cuối `05_progress.md` để verify format.
2. Kiểm tra danh sách file trong workspace folder.
3. Báo cáo Governance Status: 🟢 (Pass) / 🔴 (Fail - Require RCA).
