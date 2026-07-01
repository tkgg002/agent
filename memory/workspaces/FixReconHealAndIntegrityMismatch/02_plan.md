# Plan - FixReconHealAndIntegrityMismatch

## Giai đoạn 1: Khảo sát & Research

3. **Phân tích lỗi Heal `noop` của `payment_bills`**:
   - Tìm bản ghi bị lệch cụ thể bằng cách query trực tiếp MongoDB source và PostgreSQL shadow.
   - Xem giá trị `_id`, `updated_at` của bản ghi bị lệch.
   - Xác định xem tại sao `RunTier2` không phát hiện ra missing ID này. Có phải do lệch trường so sánh thời gian (`_source_ts` vs `updated_at`) hay do mốc thời gian của nó nằm ngoài window scan hay do kiểu dữ liệu của `_id`?

## Giai đoạn 2: Lập implementation_plan.md chi tiết
- Đề xuất phương án sửa lỗi chi tiết cho từng vấn đề.
- Tạo `implementation_plan.md` artifact và xin ý kiến user.
