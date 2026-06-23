# Context: Hermes Closed Learning Loop for /agent

## Overview
Dự án `/agent` (bộ khung quy tắc, quy trình, tri thức và kỹ năng cho Gemini/Claude CLI Agent) hiện tại chỉ có một cơ chế tự học thụ động thông qua việc ghi chép thủ công các lỗi sai vào `lessons.md`. Khi có lỗi xảy ra hoặc khi tìm ra giải pháp tối ưu hơn, hệ thống chưa có khả năng tự động cập nhật các quy trình tĩnh (`agent/workflows/`), tự đúc kết kỹ năng mới (`agent/skills/`) hay tự tiến hóa hiến pháp (`agent/GEMINI.md` và `CLAUDE.md`).

Nhiệm vụ của chúng ta là hiện thực hóa cơ chế **Closed Learning Loop** (lấy cảm hứng từ Hermes Agent của Nous Research) giúp `/agent` tự hoàn thiện liên tục thông qua 3 trụ cột cốt lõi:
1. **Workflow Self-Optimization**: Tự động vá lỗi và tối ưu hóa quy trình (SOP) tĩnh.
2. **Autonomous Skills Generation**: Tự động đúc kết và tự tiến hóa các kỹ năng (Skills) đặc thù dự án.
3. **Rules & Constitution Self-Evolution**: Định kỳ rà soát, thúc đẩy bài học kinh nghiệm lặp lại thành Quy tắc cứng và đồng bộ hiến pháp.

## Objectives
- Thiết lập thư mục `agent/skills/` và định nghĩa định dạng tiêu chuẩn `SKILL.md` tương thích *agentskills.io*.
- Viết bộ kịch bản tự động hóa (bằng Python/Shell Script) chạy ngầm hoặc qua lệnh kích hoạt để:
  - Tự động vá lỗi SOP tại `agent/workflows/` khi có cải tiến hoặc khi tìm thấy root cause.
  - Tự động khai thác (mine) và đóng gói các kĩ năng mới từ lịch sử chạy thực tế (`05_progress.md` hoặc git diff).
  - Tự động đồng bộ bài học kinh nghiệm lặp lại từ `lessons.md` lên `GEMINI.md` và `CLAUDE.md`.
- Thiết lập các điểm kích hoạt (Learning Triggers): Mid-session và Wrap-up session.

## System Impact
- Toàn bộ các quy tắc, quy trình và kỹ năng của agent sẽ tự động hóa tiến hóa theo thời gian và bối cảnh thực tế mà không cần sự can thiệp thủ công liên tục của con người.
- Giảm thiểu đáng kể thời gian "hand-holding" từ phía người dùng khi hệ thống gặp lỗi lặp lại.
