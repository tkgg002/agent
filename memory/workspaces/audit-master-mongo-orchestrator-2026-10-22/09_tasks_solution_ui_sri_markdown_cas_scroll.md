# 09 Tasks Solution: Khắc phục `#sri-markdown-content` chỉ chiếm 1/2 `.sri-markdown-pane` & `.cas-messages-list` mất Thanh cuộn (Scroll)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `ui_sri_markdown_cas_scroll`
- **Tier:** `T2`
- **Project:** `ai-orchestrator` (`/Users/trainguyen/Documents/work/ai-orchestrator`)

---

## 1. Phân tích Nguyên nhân Gốc rễ (`S2 Investigate` — `VERIFIED` kèm `file:line`)

### 1.1. Lỗi 1: Vì sao `#sri-markdown-content` (`.sri-pre-block`) chỉ chiếm đúng 1 nửa khung `.sri-markdown-pane`?
- **Bằng chứng `file:line` (`VERIFIED`):**
  - Tại [index.html](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html#L288-L320) và [app.css](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1776-L1858):
    - Khung cha `.sri-body` (`app.css:1776-1780`) dùng `display: grid; grid-template-columns: 1.15fr 1fr; gap: 10px;`. Theo mặc định của CSS Grid (`align-items: stretch`), cột trái `.sri-markdown-pane` tự động giãn chiều cao bằng với cột phải `.sri-side-pane`.
    - Cột phải `.sri-side-pane` chứa cả `#sri-code-demo-content.sri-code-block` (`max-height: 145px` tại `app.css:1822`) **cộng thêm** khối `.sri-subtasks-box` (`app.css:1836-1858`, trong đó `.sri-subtasks-list` có `max-height: 220px` + thanh filter repo + form thêm Sub-Task), khiến tổng chiều cao thực tế của `.sri-side-pane` (và `.sri-markdown-pane`) lên tới **~460px–480px**.
    - Thế nhưng bên trong cột trái `.sri-markdown-pane` (`display: flex; flex-direction: column;` tại `app.css:1781-1790`), thẻ `<pre id="sri-markdown-content" class="sri-pre-block">` (`app.css:1801-1810`) lại bị khóa cứng:
      ```css
      .sri-pre-block {
        ...
        max-height: 235px;
        overflow-y: auto;
      }
      ```
      và **thiếu `flex: 1; min-height: 0;`**! Vì `235px` chỉ bằng đúng ~1/2 của `470px`, phần tử `#sri-markdown-content` bị cắt cụt ở nửa trên của `.sri-markdown-pane`, để lại nửa dưới của `.sri-markdown-pane` là một khoảng trống màu đen vô nghĩa.

### 1.2. Lỗi 2: Vì sao `.cas-messages-list` (`#co-architect-chat-messages`) không có thanh cuộn (scroll)?
- **Bằng chứng `file:line` (`VERIFIED`):**
  - Tại [index.html](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html#L380):
    ```html
    <div class="cas-messages-list" id="co-architect-chat-messages"></div>
    ```
  - Nhưng tại [app.css](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L2348-L2358), rule CSS quy định chiều cao tối đa và thanh cuộn (`max-height: 185px; overflow-y: auto;`) lại đặt tên selector là **`.cas-messages-box`** thay vì **`.cas-messages-list`**:
    ```css
    .cas-messages-box {
      background: #060911;
      border: 1px solid rgba(148, 163, 184, 0.18);
      border-radius: 6px;
      padding: 7px 9px;
      max-height: 185px;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    ```
  - Toàn bộ file `app.css` hoàn toàn không có selector `.cas-messages-list` (hay `#co-architect-chat-messages`)! Hậu quả là `<div class="cas-messages-list" id="co-architect-chat-messages">` không hề nhận được `max-height` và `overflow-y: auto`, dẫn đến danh sách tin nhắn tràn dài vô tận không có thanh cuộn dọc, và lệnh `listEl.scrollTop = listEl.scrollHeight` tại [app.js](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js#L985) không có tác dụng.
  - Ngoài ra, đối chiếu toàn bộ cây DOM của `#co-architect-studio` trong [index.html](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html#L326-L395) và [app.js](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.js#L966-L982) với [app.css](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L2286-L2442) còn phát hiện các selector bị lệch tên tương tự:
    - `index.html:328,330,332,333`: `.cas-title-group`, `.cas-sub-hint`, `.cas-controls`, `.cas-select-label` (trong `app.css` đang viết `.cas-title-left`, `.cas-sub`, `.cas-controls-right`, `.cas-label`).
    - `app.js:966-978`: `.cas-msg-card`, `.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`, `.cas-msg-author`, `.cas-slot-pill`, `.cas-skill-pill`, `.cas-model-pill`, `.cas-msg-meta`, `.cas-mode-pill`, `.cas-applied-pill`, `.cas-time` (trong `app.css` đang viết `.cas-msg`, `.cas-msg-user`, `.cas-msg-agent`, `.cas-msg-model`, `.cas-msg-file`, `.cas-msg-time`).
    - `index.html:384-393`: `.cas-input-form textarea`, `.cas-form-actions`, `.btn-cas-send`, `.btn-cas-approve` (trong `app.css` chỉ viết `.cas-input-form input`, `.cas-submit-btns`, `.btn-cas-discuss`).

---

## 2. Phương án Khắc phục Tối ưu Duy nhất (`S3 Single-Best Solution` & Concrete Code Demo)

### 2.1. Thiết kế Giải pháp (1 Phương án duy nhất trên `internal/webui/static/app.css`)
1. **Đối với `.sri-markdown-pane` và `#sri-markdown-content` (`.sri-pre-block`) tại `app.css:1781-1810`:**
   - Gắn `flex: 1; min-height: 260px; max-height: 465px; margin: 0;` (và `#sri-markdown-content { flex: 1; min-height: 260px; max-height: 465px; overflow-y: auto; }`) để khung nội dung báo cáo Markdown tự động giãnเต็ม 100% chiều cao bên trong `.sri-markdown-pane`, khớp khít với tổng chiều cao của cột `.sri-side-pane` bên phải (`Code Demo` + `Sub-Tasks`) và cuộn dọc mượt mà khi báo cáo dài.
2. **Đối với `.cas-messages-list` (`#co-architect-chat-messages`) và các phần tử con trong `.co-architect-studio` tại `app.css:2286-2443`:**
   - Khai báo đầy đủ selector `.cas-messages-list, .cas-messages-box, #co-architect-chat-messages` với:
     - `max-height: 260px; min-height: 110px; overflow-y: auto; overflow-x: hidden; overscroll-behavior: contain;`
     - Style thanh cuộn (`scrollbar-width: thin`, `::-webkit-scrollbar`) rõ nét để anh dễ dàng kéo xem lịch sử hội thoại dài (như báo cáo Audit 12 mục trong `TASK-11`).
   - Đồng bộ toàn bộ selector thực tế đang dùng trong `index.html:326-395` và `app.js:966-982` (`.cas-title-group`, `.cas-sub-hint`, `.cas-controls`, `.cas-select-label`, `.cas-msg-card`, `.cas-msg-card.user-msg`, `.cas-msg-card.agent-msg`, `.cas-msg-author`, `.cas-slot-pill`, `.cas-skill-pill`, `.cas-model-pill`, `.cas-msg-meta`, `.cas-mode-pill`, `.cas-applied-pill`, `.cas-time`, `.cas-input-form textarea`, `.cas-form-actions`, `.btn-cas-send`, `.btn-cas-approve`).

### 2.2. Concrete Code Demo (`internal/webui/static/app.css`)

```css
/* 1. Fix #sri-markdown-content (.sri-pre-block) filling 100% of .sri-markdown-pane (app.css:1781-1810) */
.sri-markdown-pane, .sri-side-pane {
  background: #080b12;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 7px;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  min-height: 0;
}
.sri-pre-block, #sri-markdown-content {
  font-family: "JetBrains Mono", Menlo, monospace;
  font-size: 10.5px;
  line-height: 1.45;
  color: #e2e8f0;
  background: #05080e;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 5px;
  padding: 8px 10px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  flex: 1 1 auto;
  min-height: 260px;
  max-height: 465px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

/* 2. Fix .cas-messages-list (#co-architect-chat-messages) scroll & Co-Architect Studio selectors (app.css:2286-2443) */
.cas-title-left, .cas-title-group, .cas-controls-right, .cas-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.cas-sub, .cas-sub-hint {
  font-size: 10px;
  color: #94a3b8;
}
.cas-sub code, .cas-sub-hint code {
  color: #67e8f9;
}
.cas-label, .cas-select-label {
  font-size: 10px;
  color: #cbd5e1;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.cas-messages-list, .cas-messages-box, #co-architect-chat-messages {
  background: #060911;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 6px;
  padding: 8px 10px;
  min-height: 110px;
  max-height: 260px;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: rgba(56, 189, 248, 0.45) rgba(15, 23, 42, 0.6);
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.cas-messages-list::-webkit-scrollbar {
  width: 7px;
}
.cas-messages-list::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.6);
  border-radius: 4px;
}
.cas-messages-list::-webkit-scrollbar-thumb {
  background: rgba(56, 189, 248, 0.45);
  border-radius: 4px;
}
.cas-messages-list::-webkit-scrollbar-thumb:hover {
  background: rgba(56, 189, 248, 0.75);
}
.cas-msg, .cas-msg-card {
  padding: 7px 9px;
  border-radius: 6px;
  font-size: 10.5px;
  line-height: 1.42;
  flex-shrink: 0;
}
.cas-msg-user, .cas-msg-card.user-msg {
  background: rgba(56, 189, 248, 0.13);
  border: 1px solid rgba(56, 189, 248, 0.35);
}
.cas-msg-agent, .cas-msg-card.agent-msg {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
}
.cas-msg-hdr {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 4px;
  font-size: 9.5px;
}
.cas-msg-author, .cas-msg-meta {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
}
.cas-slot-pill, .cas-skill-pill, .cas-mode-pill {
  font-family: "JetBrains Mono", Menlo, monospace;
  font-size: 9px;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(15, 23, 42, 0.85);
  color: #cbd5e1;
  border: 1px solid rgba(148, 163, 184, 0.2);
}
.cas-msg-model, .cas-model-pill {
  font-family: "JetBrains Mono", Menlo, monospace;
  color: #67e8f9;
}
.cas-msg-file, .cas-applied-pill {
  font-family: "JetBrains Mono", Menlo, monospace;
  color: #fde68a;
  background: rgba(245, 158, 11, 0.16);
  padding: 1px 5px;
  border-radius: 3px;
}
.cas-msg-time, .cas-time {
  color: #64748b;
  margin-left: auto;
}
.cas-msg-body {
  font-family: "JetBrains Mono", Menlo, monospace;
  font-size: 10px;
  color: #f1f5f9;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
.cas-input-form {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cas-input-form input, .cas-input-form textarea {
  width: 100%;
  background: #070b12;
  color: #f8fafc;
  border: 1px solid rgba(245, 158, 11, 0.42);
  border-radius: 5px;
  padding: 6px 9px;
  font-size: 10.5px;
  font-family: "JetBrains Mono", Menlo, monospace;
  resize: vertical;
}
.cas-submit-btns, .cas-form-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.btn-cas-discuss, .btn-cas-send {
  background: #0284c7;
  color: #fff;
  border: none;
  border-radius: 5px;
  padding: 5px 10px;
  font-size: 10px;
  font-weight: 800;
  cursor: pointer;
  white-space: nowrap;
}
.btn-cas-approve {
  background: #10b981;
  color: #04130a;
  border: none;
  border-radius: 5px;
  padding: 5px 10px;
  font-size: 10px;
  font-weight: 800;
  cursor: pointer;
  white-space: nowrap;
}
```
