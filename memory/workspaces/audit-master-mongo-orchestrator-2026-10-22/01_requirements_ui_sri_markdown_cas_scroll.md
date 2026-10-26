# 01 Requirements: Sửa lỗi hiển thị `.sri-markdown-content` chỉ chiếm 1/2 `.sri-markdown-pane` và `.cas-messages-list` không có thanh cuộn (Scroll)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `ui_sri_markdown_cas_scroll`
- **tier:** T2
- **Project:** `ai-orchestrator` (`/Users/trainguyen/Documents/work/ai-orchestrator`)
- **Ngày tạo:** 2026-10-22

---

## 1. Yêu cầu của Anh (User Request)
> `"sri-markdown-content đang chỉ có 1 nửa cái sri-markdown-pane, còn cái cas-messages-list sao ko có scroll."`

## 2. Phạm vi & Ràng buộc Bất biến
1. **Ranh giới Repo (`L003`):** Chỉ làm việc trên `/Users/trainguyen/Documents/work/ai-orchestrator` (`internal/webui/static/app.css` và test liên quan). Tuyệt đối không chạm vào `/Users/trainguyen/Documents/work/data-hub/*`.
2. **Mục tiêu Kỹ thuật:**
   - **Vấn đề 1 (`#sri-markdown-content` / `.sri-pre-block` trong `.sri-markdown-pane`):** Làm cho `#sri-markdown-content` lấp đầy toàn bộ chiều cao khả dụng của khung `.sri-markdown-pane` (thay vì bị bóp cứng ở `max-height: 235px` khiến nửa dưới `.sri-markdown-pane` bị trống đen khi cột phải `.sri-side-pane` cao ~460px–480px) và có thanh cuộn dọc mượt mà.
   - **Vấn đề 2 (`.cas-messages-list` / `#co-architect-chat-messages` trong `#co-architect-studio`):** Khắc phục lỗi lệch tên class CSS giữa `index.html:380` (`.cas-messages-list`), `app.js:966-981` (`.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`, `.cas-msg-author`, `.cas-msg-meta`, ...) và `app.css:2286-2442` (đang viết `.cas-messages-box` thay vì `.cas-messages-list`), đảm bảo `.cas-messages-list` có `max-height`, `overflow-y: auto`, scrollbar rõ ràng và tự động cuộn xuống tin nhắn mới nhất.
