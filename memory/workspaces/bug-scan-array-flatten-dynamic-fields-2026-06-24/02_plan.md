# Plan: Scan Array Flatten Dynamic Fields Bug

## Objectives
1. Phân tích code trong `scan_service.go` và các file liên quan để tìm cơ chế quét và làm phẳng (flatten) array/JSON fields.
2. Xác định nguyên nhân tại sao khi scan array nó lại tạo thẳng vào `mapping_rule_v2` với key động hoặc representation của array.
3. Thiết kế giải pháp bỏ qua việc quét (scan) các child field động của array hoặc chặn tạo mapping rule tĩnh cho các index array `[0,1,2,3,4]`.
4. Triển khai giải pháp.
5. Viết unit test để kiểm chứng giải pháp.

## Checklist
- [ ] Phân tích `scan_service.go` và tìm logic scan flatten array.
- [ ] Tìm hiểu cách hệ thống xử lý các array fields (đặc biệt là array of objects).
- [ ] Thiết lập Implementation Plan chi tiết để trình User duyệt.
- [ ] Cập nhật code để bỏ qua việc sinh mapping rule tĩnh cho dynamic array fields.
- [ ] Viết unit test bảo vệ logic mới.
- [ ] Chạy verify test và log output.
