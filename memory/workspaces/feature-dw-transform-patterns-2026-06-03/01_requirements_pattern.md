# 01_requirements_pattern.md — Yêu cầu (reframed theo User)

## Yêu cầu cốt lõi (User, 2026-06-03)
1. **Ưu tiên PATTERN + cấu trúc source + system design TRƯỚC**, không làm plan bao quát "xong 1 lần".
2. Mỗi loại sync = **1 file tự chứa**, implement chung interface. Thêm loại mới sau = **copy 1 file + thêm 1 type + khai báo** → tự chạy theo luồng file đó (plugin/registry).
3. Làm **2 loại basic trước**: `1:1` và `trải phẳng (flatten)`.
4. Các loại còn lại (filter/groupby/aggregate/join) = **option lắp sau**; đơn giản thì làm luôn, không thì để sau.

## Diễn giải kỹ thuật
- Đây là **Strategy + Registry pattern** ở tầng `master_binding.transform_type`.
- Khung (framework) là việc làm NGAY; từng loại sync là plugin lắp dần.
- Tiêu chí "đúng": *"Thêm 1 loại sync mới có thực sự chỉ cần thêm 1 file + 1 dòng register không?"* → nếu phải sửa engine thì thiết kế chưa đạt.

## Definition of Done (phase framework)
- [ ] Có interface `Strategy` + registry (mirror `transform_registry.go`).
- [ ] `TransmuterModule.Run/processBatch` dispatch theo `transform_type` (không còn hardcode 1:1).
- [ ] `copy_1_to_1` tách thành 1 file strategy (giữ nguyên hành vi hiện tại — không regression).
- [ ] `flatten` là 1 file strategy mới, chạy được array-explode lên master (xử lý `_source_id` fan-out).
- [ ] Thêm loại mới = copy 1 file + `init()` register + khai báo enum. Có doc hướng dẫn.
- [ ] Build + go vet + test pass; transmute 1:1 cũ không đổi kết quả.
