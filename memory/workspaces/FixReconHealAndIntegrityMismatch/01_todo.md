# TODO - FixReconHealAndIntegrityMismatch

- [ ] Tìm nguyên nhân cụ thể tại sao `failed_sync_logs` đếm ra 465 thay vì 467.
- [ ] Tìm nguyên nhân tại sao Tab tổng quan báo Khớp cho `payment_bills` dù thực tế lệch 1 bản ghi.
- [ ] Tìm nguyên nhân và sửa logic tự chữa lành (Heal) cho `payment_bills` bị `noop`:
  - Kiểm tra xem bản ghi bị thiếu có mốc thời gian cũ hơn 7 ngày hay không.
  - Sửa logic đối soát để tìm chính xác bản ghi thiếu ở shadow mà không bị lệch do sai số `_source_ts` vs `updated_at`.
- [ ] Implement and verify fixes.
- [ ] Chạy các test suite để đảm bảo không phá vỡ code cũ.
