# Plan: Implement Hermes Closed Learning Loop for /agent

## Phases of Implementation

### Phase 1: Setup Infrastructure & Directory Structure
- [ ] Khởi tạo thư mục kỹ năng đặc thù dự án `agent/skills/`.
- [ ] Tạo file mẫu chỉ dẫn kỹ năng `agent/skills/SKILL_TEMPLATE.md` tuân thủ tiêu chuẩn *agentskills.io*.
- [ ] Tạo thư mục tooling chứa các script tự động hóa: `agent/tooling/learning_loop/`.

### Phase 2: Workflow Self-Optimization Engine (SOP Patcher)
- [ ] Viết script `agent/tooling/learning_loop/sop_patcher.py`:
  - Đọc log lỗi hoặc `05_progress.md` hiện tại.
  - Xác định quy trình (SOP) nào trong `agent/workflows/` vừa thực hiện bị lỗi hoặc được tối ưu.
  - Áp dụng kỹ thuật `patch` (tìm và thay thế chuỗi chính xác) để cập nhật SOP một cách tự động.

### Phase 3: Autonomous Skills Generator (Skill Miner)
- [ ] Viết script `agent/tooling/learning_loop/skill_miner.py`:
  - Phân tích `05_progress.md` hoặc vết Git Diff của phiên làm việc hiện tại sau khi hoàn thành Task phức tạp (> 5 bước).
  - Trích xuất: Tên kỹ năng, bối cảnh, các bước thực hiện, giải pháp xử lý lỗi và kịch bản kiểm thử (Verification).
  - Sinh ra tệp `SKILL.md` hoàn chỉnh lưu vào `agent/skills/<skill-name>/SKILL.md`.

### Phase 4: Rules & Constitution Self-Evolution (Rule Promoter)
- [ ] Viết script `agent/tooling/learning_loop/rule_promoter.py`:
  - Phân tích tệp `agent/memory/global/lessons.md`.
  - Phát hiện các Pattern lỗi bị lặp lại nhiều lần (ngưỡng mặc định: >= 3 lần vi phạm).
  - Sinh đề xuất nâng cấp thành Rule cứng gửi vào tệp đề xuất luật hoặc tự động chèn vào `agent/GEMINI.md` dưới phần Luật bổ sung.
  - Đồng bộ hóa các quy tắc mới sang `CLAUDE.md` để đảm bảo tính thống nhất của hiến pháp.

### Phase 5: Learning Triggers Integration (Mid-Session & Wrap-Up)
- [ ] Viết kịch bản tích hợp và kích hoạt `agent/scripts/wrapup.sh`:
  - Kích hoạt chạy tự động tất cả các công cụ học tập cuối phiên (Skill Miner, SOP Patcher, Rule Promoter).
  - Cập nhật tiến độ học tập vào `agent/memory/global/active_plans.md` và sinh báo cáo tiến hóa tri thức.
- [ ] Tạo workflow hướng dẫn agent gọi kịch bản học tập: `agent/workflows/learning-loop-trigger.md`.

## Verification & Testing Strategy
- Viết các file kịch bản giả lập (Mock logs, mock lessons, mock progress log) để chạy kiểm thử cục bộ cho 3 script Python trên.
- Đảm bảo các script Python không làm hỏng cấu trúc tệp markdown, giữ nguyên tính đúng đắn của cú pháp yamlfrontmatter nếu có.
- Kiểm tra tính toàn vẹn của hiến pháp sau khi tự động đồng bộ.
