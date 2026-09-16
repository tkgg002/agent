# Yêu cầu Chi tiết (Requirements): Sắp xếp Schedules theo Last Run

## 1. Bối cảnh
Trang `http://localhost:5173/schedules` (`TransmuteSchedules.tsx`) hiển thị danh sách các lịch đồng bộ (transmute schedules).
Người dùng muốn danh sách này tự động sắp xếp (sort) theo thời gian chạy gần nhất (`last_run` / `last_run_at`) với thứ tự giảm dần (mới nhất lên trên), đồng thời bổ sung tính năng sorter cho cột `Last run` trên giao diện bảng.

## 2. Tiêu chí Hoàn thành (Definition of Done)
1. Danh sách `schedules` được mặc định sắp xếp giảm dần theo `last_run_at` (bản ghi vừa chạy gần đây nhất xếp đầu tiên, bản ghi chưa bao giờ chạy xếp cuối cùng).
2. Cột `Last run` trên Antd Table có tính năng sorter cho phép người dùng click đổi thứ tự tăng/giảm dần linh hoạt.
3. Build check `npm run build` thành công 100% không văng lỗi TypeScript.
