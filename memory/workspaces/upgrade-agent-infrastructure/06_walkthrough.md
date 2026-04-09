# Walkthrough: Nâng cấp hạ tầng Agent (Agentic Core v1.10.0)

Tôi đã hoàn tất việc nâng cấp hạ tầng cho Agent, xác lập thư mục `agent/` là **Agentic Core** (Hạt nhân điều phối) và `.agent/` là **Infrastructure Harness** (Hạ tầng công cụ).

## Các thay đổi chính

### 1. Nâng cấp Muscle (Hạ tầng công cụ)
- **Backup**: Đã sao lưu thư mục `.agent` cũ sang `.agent_backup_20260406_113610`.
- **Rules**: Cập nhật 15 bộ quy tắc ngôn ngữ mới (Go, TS, Python, Rust, C++, Java, v.v.) vào `[.agent/rules](file:///Users/trainguyen/Documents/work/.agent/rules)`.
- **Skills**: Đồng bộ 181 kỹ năng mới nhất của phiên bản v1.10.0 vào thư viện toàn cục `[~/.gemini/antigravity/skills/](file:///Users/trainguyen/.gemini/antigravity/skills/)`.
- **Workflows**: Cập nhật 79 lệnh kỹ thuật (commands) vào `[.agent/workflows](file:///Users/trainguyen/Documents/work/.agent/workflows)`.

### 2. Thiết lập Agentic Core (Quy trình quản trị)
- **Authority Hierarchy**: Cập nhật file `[GEMINI.md](file:///Users/trainguyen/Documents/work/agent/GEMINI.md)` với **Quy tắc #10**.
- **Cấu trúc Ưu tiên**: Khẳng định `agent/` luôn ghi đè (Override) lên `.agent/`. Khi có xung đột giữa quy trình mặc định (như `/plan`) và quy trình dự án (như `/brain-delegate`), Agent sẽ luôn chọn quy trình dự án.

## Kết quả xác minh

- [x] **Tính toàn vẹn**: File `GEMINI.md` đã được cập nhật chính xác nội dung Quy tắc #10.
- [x] **Tính sẵn sàng**: Các skills mới như `brand-voice` và `social-graph-ranker` đã có mặt trong hệ thống.
- [x] **Tính ổn định**: Toàn bộ `agent/memory` (Trí nhớ dự án) được giữ nguyên vẹn, không bị ảnh hưởng bởi quá trình nâng cấp hạ tầng.

## Hướng dẫn cho Agent (Brain) trong tương lai
Từ nay, khi thực hiện bất kỳ feature nào, Agent phải:
1. Đọc `agent/GEMINI.md` để nắm bắt quy trình Brain/Muscle.
2. Luôn sử dụng workflows trong `agent/workflows/` làm ưu tiên 1.
3. Sử dụng các kỹ năng và rules mới cập nhật trong `.agent/` làm công cụ hỗ trợ thực thi cho Muscle.

> [!TIP]
> Việc tách biệt **Hạt nhân Agentic (`agent/`)** và **Hạ tầng Công cụ (`.agent/`)** giúp bạn có thể nâng cấp framework thoải mái mà không lo làm hỏng "Bộ não" riêng của dự án.
