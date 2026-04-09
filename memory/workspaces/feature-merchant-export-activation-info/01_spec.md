# Specification: Merchant Export Activation Info

## 01_spec.md - Đặc tả yêu cầu

### Các cột bổ sung vào file Excel
1. **Mã Đối tác (Merchant Code)**.
2. **Ngày kích hoạt**: Định dạng theo hệ thống.
3. **Người kích hoạt**: Tên hoặc ID người dùng thực hiện kích hoạt cuối cùng.

### Logic nghiệp vụ (Business Logic)
- Nếu Merchant chưa từng bị Inactive: `Activation Date = createdAt`.
- Nếu Merchant đã từng Inactive và được Active lại:
  - Truy vấn bảng History giao dịch/thay đổi của Merchant.
  - Tìm bản ghi "Activate" gần nhất.
  - `Activation Date = history.createdAt`.
  - `Activator = history.updatedBy` (hoặc trường tương đương).
