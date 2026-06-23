# Workspace Tasks: Hermes Closed Learning Loop

## Task: Phase 1 - Infrastructure Setup
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: agent-learning-loop
- **Mô tả**: Thiết lập thư mục và tệp mẫu chỉ dẫn kỹ năng SKILL_TEMPLATE.md
- **Trạng thái**: [x] DONE

### [Context]
- Current state: Khởi tạo hạ tầng cho việc lưu trữ skills và tooling.
- Dependencies: None
- ADR liên quan: Decision 1 (Skills Folder Structure)

### [Definition of Done]
- [x] Thư mục `agent/skills/` được khởi tạo.
- [x] Tệp `agent/skills/SKILL_TEMPLATE.md` được tạo và tuân thủ định dạng agentskills.io.
- [x] Thư mục `agent/tooling/learning_loop/` được tạo.
- [x] Ghi nhận tiến độ vào `05_progress.md`.

---

## Task: Phase 2 - SOP Patcher Implementation
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: agent-learning-loop
- **Mô tả**: Phát triển bộ vá quy trình tự động `sop_patcher.py`
- **Trạng thái**: [x] DONE

### [Context]
- Current state: SOP Patcher dùng để tự động vá các workflow khi có chỉ thị lỗi.
- Dependencies: agent/workflows/
- ADR liên quan: Decision 2, Decision 3 (Semantic Patching)

### [Definition of Done]
- [x] Script `agent/tooling/learning_loop/sop_patcher.py` hoạt động chính xác.
- [x] Hỗ trợ lệnh `[SOP_PATCH] workflow_name.md | TargetContent | ReplacementContent`.
- [x] Có mock test kiểm chứng thành công.
- [x] Ghi nhận tiến độ vào `05_progress.md`.

---

## Task: Phase 3 - Skill Miner Implementation
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: agent-learning-loop
- **Mô tả**: Phát triển bộ khai thác kỹ năng tự động `skill_miner.py`
- **Trạng thái**: [x] DONE

### [Context]
- Current state: Skill Miner trích xuất kỹ năng từ progress logs của các tasks phức tạp.
- Dependencies: agent/memory/workspaces/
- ADR liên quan: Decision 1, Decision 2

### [Definition of Done]
- [x] Script `agent/tooling/learning_loop/skill_miner.py` trích xuất thành công `SKILL.md` từ workspace logs.
- [x] `SKILL.md` đầu ra tuân thủ agentskills.io và lưu đúng thư mục.
- [x] Có mock test kiểm chứng thành công.
- [x] Ghi nhận tiến độ vào `05_progress.md`.

---

## Task: Phase 4 - Rule Promoter Implementation
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: agent-learning-loop
- **Mô tả**: Phát triển bộ tự tiến hóa luật hiến pháp `rule_promoter.py`
- **Trạng thái**: [x] DONE

### [Context]
- Current state: Rule Promoter phân tích lessons.md để chèn luật mới vào GEMINI.md và đồng bộ sang CLAUDE.md.
- Dependencies: agent/memory/global/lessons.md, agent/GEMINI.md, agent/CLAUDE.md
- ADR liên quan: Decision 2, Rule 17 (Constitution Sync)

### [Definition of Done]
- [x] Script `agent/tooling/learning_loop/rule_promoter.py` phát hiện các tag/pattern lỗi lặp lại >= 3 lần.
- [x] Tự động chèn luật mới vào `agent/GEMINI.md` dưới phần luật bổ sung và sao lưu (.bak).
- [x] Đồng bộ chính xác sang `agent/CLAUDE.md`.
- [x] Có mock test kiểm chứng thành công.
- [x] Ghi nhận tiến độ vào `05_progress.md`.

---

## Task: Phase 5 - Integration & Triggers
- **Phase**: GĐ0
- **Service Group**: Utilities
- **Service(s)**: agent-learning-loop
- **Mô tả**: Tạo script wrapup.sh và workflow trigger để kích hoạt learning loop ở cuối mỗi session.
- **Trạng thái**: [x] DONE

### [Context]
- Current state: Tích hợp các công cụ vào quy trình làm việc chuẩn của agent.
- Dependencies: agent/scripts/, agent/workflows/

### [Definition of Done]
- [x] Script `agent/scripts/wrapup.sh` hoạt động và chạy tuần tự các công cụ học tập.
- [x] Tệp `agent/workflows/learning-loop-trigger.md` hướng dẫn chi tiết cách chạy.
- [x] Chạy `/security-agent` rà soát mã nguồn thành công.
- [x] Ghi nhận tiến độ vào `05_progress.md`.
