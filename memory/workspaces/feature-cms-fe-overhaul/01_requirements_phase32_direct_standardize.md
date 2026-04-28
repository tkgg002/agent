# Phase 32 Requirements — Direct Standardize

## Mục tiêu
- Tiếp tục giảm phụ thuộc `registry_id` cho operator-flow.
- Ưu tiên direct-V2 hóa action `standardize` vì worker payload hiện chỉ cần `target_table`.

## Audit findings
- `standardize` worker path dùng `target_table` là chính; `registry_id` chỉ là metadata phụ cho result/audit.
- `create-default-columns` vẫn chưa direct-V2 hóa an toàn vì còn phụ thuộc metadata legacy sâu hơn và ảnh hưởng `is_table_created`.
- FE hiện chặn row V2-only ở nút `Tạo Field MĐ` một cách không cần thiết.

## Yêu cầu kèm theo
- Update swagger annotations cùng phase.
- Verify bằng `go test ./...` và `npm run build`.
