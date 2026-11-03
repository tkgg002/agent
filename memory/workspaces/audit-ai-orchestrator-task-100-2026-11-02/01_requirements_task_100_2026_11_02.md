---
tier: T2
mode: propose-only
target_project: ai-orchestrator
suffix: task_100_2026_11_02
created_at: 2026-11-02T16:12:00+07:00
---

# 01. Requirements — Điều tra lỗi S1 / S2 / S2.5 / S3 của `TASK-100` trên `ai-orchestrator`

## 1. Yêu cầu của User
- Kiểm tra lý do vì sao khi chạy `TASK-100` trên `ai-orchestrator` với đề bài:
  `Hiện tại repo đang ở branch feat/auto-recon-export , kiểm tra xem các issue đang mắc phải.`
- Kết quả ở các bước `S1 (Đặc Tả Yêu Cầu & Phân Rã Sub-Tasks)`, `S2`, `S2.5`, `S3` đều bị sai lệch hoàn toàn và tự ý lôi cả `cdc` repo vào phân tích không liên quan.

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L013 (`LS-GLOBAL-002-agnostic-core`):** Cấm hardcode từ khóa công nghệ, tên repo/bảng dự án con hoặc fallback sang repo mặc định (`cdc` / `data-hub` / v.v.) khi user chỉ định repo khác hoặc không nhắc tới `cdc`.
- **L016 (`LS-GLOBAL-005-single-conv-session`):** Mọi khâu `S1`, `S2`, `S2.5`, `S3` phải gọi đúng ngữ cảnh thực tế của `TargetRepo` được chọn (hoặc truyền đúng repo context) thay vì trộn lẫn multi-repo / `cdc` repo hoặc sinh văn mẫu sai bản chất.
- **L012 (`LS-GLOBAL-001-propose-only-gate`):** Giữ chế độ Read-Only (`Propose-Only`), phân tích gốc rễ từng bước `S1`, `S2`, `S2.5`, `S3` kèm bằng chứng `file:line`, trình bày **Code Demo** trực tiếp trên khung chat và dừng chờ `APPROVE`.
- **L029 & L030:** Mở đầu bằng **Direct Summary** 2–4 câu trả lời trực diện; ánh xạ Giải pháp 1-1 theo từng phát hiện `F1..Fn`; giữ thái độ cầu thị, lễ phép (`em - anh`).
