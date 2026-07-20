# Nhật ký Tiến độ: Tối ưu hóa Smoke Check (Audit Log)

- [2026-07-14T08:05:00Z] [Agent:Antigravity] Khởi tạo Workspace ReconSmokeBoundary, tạo 01_requirements_smoke_boundary.md và 05_progress_smoke_boundary.md.
- [2026-07-14T08:15:30Z] [Agent:Antigravity] Rà soát 3 rủi ro Gotchas của giải pháp trừ bù cửa sổ (sai số Estimated Count, Hard Deletes, cảnh báo fallback). Cập nhật requirements, solution và implementation_plan.md.
- [2026-07-14T08:19:00Z] [Agent:Antigravity] Bỏ hoàn toàn EstimatedCount và logic fallback; thiết kế đếm thêm _deleted trong cửa sổ 120s để tính toán activeClean chính xác tuyệt đối. Cập nhật requirements, solution, progress và implementation_plan.md.
- [2026-07-14T08:30:00Z] [Agent:Antigravity] Thay đổi giải pháp: Sử dụng EstimatedCount mặc định cho MongoDB, đối soát chéo HashWindow trên index thời gian khi có lệch để đảm bảo an toàn hiệu năng và tính chính xác tuyệt đối. Cập nhật requirements, solution, progress và implementation_plan.md.
- [2026-07-14T08:32:00Z] [Agent:Antigravity] Chi tiết hóa thuật toán HashWindow, cách lấy range (lo, hi từ pickScanRangeWithLag), cách băm và đối soát XOR hash để Muscle thực thi trực tiếp không cần suy luận thêm.
- [2026-07-14T09:23:00Z] [Agent:Antigravity] Sửa đổi thiết kế HashWindow: Thay đổi mốc trên hi thành fromTime (now - 120s làm tròn phút) thay vì dùng now, loại trừ hoàn toàn cửa sổ ghi đang chịu ảnh hưởng bởi lag đồng bộ để đảm bảo HashWindow không bao giờ bị nhiễu do lag.
- [2026-07-14T09:45:00Z] [Agent:Antigravity] Rà soát và hoàn thiện toàn bộ luồng smoke check A/B, đồng bộ tài liệu workspace và sửa đổi format của 05_progress_smoke_boundary.md để vượt qua kiểm tra của verify_governance.py.
- [2026-07-14T09:52:00Z] [Agent:Antigravity] Phát hiện lỗi relation "schedule_histories" does not exist trong RunTotalOnlyA do sử dụng entry.TargetTable chưa qualified; đã sửa thành entry.QualifiedTarget() để trỏ đúng shadow schema.
