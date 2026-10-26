# 01 Requirements: Nâng cấp Workflow `ai-orchestrator` — Cổng Hook Audit Phản Biện `S5` Chuẩn `/agent` (`S5 -> S3 Re-Plan`), Nhịp Loading `~3s` Giữa Các Bước & Phản hồi Trực tiếp vào Co-Architect Studio (`approve_and_run`)

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `co_architect_approve_run_reply`
- **Tier:** `T2`
- **Project:** `ai-orchestrator`
- **Ngày tạo:** 2026-10-22

---

## 1. Bối cảnh & Yêu cầu của User
1. **Phản hồi trực tiếp vào Co-Architect Studio & Thực thi cô lập `S4–S5` khi bấm `✓ APPROVE Giải Pháp & Chạy S4–S5`:**
   - Lúc `13:24:22`, anh dán Báo cáo Audit vào khung **Co-Architect Studio** (`#co-architect-studio`) của `TASK-11` trên `http://127.0.0.1:8090`, sau đó lúc `13:25:27` bấm **`✓ APPROVE Giải Pháp & Chạy S4–S5`** (`runTaskAction('', 'approve_and_run')` tại `index.html:390`) và chất vấn: `"mẹ mày, tao vừa apply, vẫn chưa thấy trả lời. mày làm chưa."`.
2. **Nâng cấp Workflow hiện tại với Cổng Hook Audit Phản Biện `S5` chuẩn `/agent` & Vòng lặp `S5 -> S3 Re-Plan`:**
   - Anh chỉ đạo rõ:
     > *"nâng cấp work flow hiện tại, có bước audit lại như /agent của tao đang setup hook, với cái promt kiểu này: 'Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. Kiểm tra các xem có suy diễn, báo cáo láo không. Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop). Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng Lưu toàn bộ vào file audit_report_*.md trong workspace của task. Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.' => nếu phát hiện vấn đề thì quay lại bước lên plan lại."*
3. **Nhịp Loading Trực Quan `~3s` Giữa Các Bước (`Paced Step-by-Step Progress`):**
   - Anh yêu cầu:
     > *"với lại cái progress sao mà nhấn 1 cái chạy cái vèo vậy, sleep loading giữa các bước tầm 3s đi, để cho có hiêu ứng và tao biết đang chạy ko phải láo"*
4. **Nối Telegram vào bước `S3`, Nhận `APPROVE` trên Telegram tự chạy `S4–S5` & Nút `✈️ Setup Telegram` cạnh `⚙ Setup CLI & Model`:**
   - Anh chỉ đạo:
     > *"1 .nối telegram vào s3 đi. và khi a gửi approve trên tele thì tự chạy. bên cnahj cái setup CLI & model thì có thêm cái setup telegram."*
5. **Giải thích & Ẩn 2 Cụm Nhãn Cố Định Trên Topbar (`ai-orchestrator · feat/...` & `🛡️ Kernel: /agent`):**
   - Anh yêu cầu:
     > *"2 . trên top của app, có cái : ai-orchestrator · feat/modular-queue-agent-kernel (1111216) 🛡️ Kernel: /agent (279 Lessons · H1..H7) . nó để có mục đích gì. nếu nó là core ko thay đổi sang cái khác đc thì ẩn đi cho đỡ rồi."*
6. **Bố Cục Phân Cấp 3 Tầng Trực Quan `Epic -> Task -> Sub-Task` trên Màn Hình `2. Wave 0..N Dependency DAG`:**
   - Anh chỉ đạo:
     > *"3. ở 2 wave 0...N, dependency DAG, có 3 tab subtab, task, epic. epic hiện có tác dụng gì. ko phải 1 epic là chứa nhiều task, 1 task chứa nhiều subtask sao. và epic là 1 chủ đề lớn đúng ko, vậy nó riêng lẻ nhau thì có line nối với nhau làm gì. theo anh nghĩ. thì cái này ko chia tab là tách làm 3 luôn, khi chọn epic thì show task ở dưới, khi chọn task thì show subtask ở dưới, vậy nó trực quan hơn."*
7. **Khắc phục lỗi nhấn gửi trên Modal `✈️ Setup Telegram` không hoạt động ("ko thấy bot spam") dù đã nhập đúng `Bot Token`, `Chat ID` và `Bot Username`:**
   - Anh phản hồi:
     > *"tao đã setup token chatid và bot user name. nhưng nhân gưi thì ko hoạt động, ko thấy bot spam"*

## 2. Nguyên nhân gốc rễ đã xác minh (`VERIFIED` `file:line`)
1. **Thiếu tin nhắn phản hồi trong `#co-architect-chat-messages` khi bấm `✓ APPROVE Giải Pháp & Chạy S4–S5` (`server.go:1072-1418`, `app.js:92-135`).**
2. **Gán sai đường dẫn `TargetFiles` trong `ingestAuditReportAndRemediation` (`task_chat.go:1013-1017`).**
3. **Bước `qa` (`server.go:1260-1418`) chưa tích hợp Cổng Hook Audit Phản Biện `S5` nguyên văn chuẩn `/agent`, chưa lưu `audit_report_<taskKey>.md`, và chưa có cơ chế tự động kích hoạt `S5 -> S3 Re-Plan` khi phát hiện vấn đề.**
4. **Thanh Progress & các bước quy trình chạy vèo `< 10ms` (`server.go:983-1018`, `app.js:92-135`).**
5. **Chưa nối Telegram Bot vào cổng `S3 (NEEDS_YOU)` + chưa có Poller/Webhook nhận lệnh `APPROVE` từ Telegram + chưa có nút `✈️ Setup Telegram` cạnh `⚙ Setup CLI & Model` (`index.html:47-50`).**
6. **Topbar (`index.html:18-21`) đang hiển thị `#hdr-project-branch` (`ai-orchestrator · feat/...`) và `#hdr-kernel-badge` (`🛡️ Kernel: /agent`) vốn là thông tin core cố định không chuyển đổi được, gây rối thanh điều hướng trên cùng.**
7. **Màn hình `2. Wave 0..N Dependency DAG` (`index.html:436-440`, `app.js:1214-1495`) đang chia 3 tab loại trừ nhau (`Sub-Tasks | Tasks | Epics`) và vẽ dây nối Bezier giữa các Epic độc lập, thay vì tách thành 3 tầng phân cấp hiển thị đồng thời từ trên xuống dưới (`Tầng 1: Epics` -> bấm Epic hiện `Tầng 2: Tasks` -> bấm Task hiện `Tầng 3: Sub-Tasks Dependency DAG`).**
8. **Lệch tên trường JSON giữa Frontend (`notify_on_s3_plan` tại `app.js:2355, 2408`) và Backend (`notify_on_s3` tại `telegram_bridge.go:59, 76, 204`) làm tắt sạch thông báo Telegram sau khi bấm Lưu:**
   - Khi anh nhập `Bot Token` (`539329***YBU0`), `Chat ID` (`-1004390214071`), `Bot Username` (`@tk001sea_bot`) và bấm **"Lưu Cấu Hình Telegram"**, `submitTelegramSettings` (`app.js:2408`) gửi `"notify_on_s3_plan": true`, nhưng `TelegramSettingsDTO` (`telegram_bridge.go:59`) lại unmarshal theo `json:"notify_on_s3"`. Kết quả là `req.NotifyOnS3` nhận giá trị `false` và ghi đè `"notify_on_s3": false` vào `.orchestrator/telegram_settings.json:6`!
   - Ngay sau đó, mọi lần gọi `notifyTelegramOnS3PlanReady` (`telegram_bridge.go:412`) đều chạm phải `if !cfg.Enabled || !cfg.NotifyOnS3 { return "DISABLED" }` và dừng ngay lập tức mà **không bao giờ gọi `sendTelegramBotMessage`**!
9. **Nút `"📤 Gửi Thử Thông Báo S3 Plan"` (`testTelegramS3Notification` tại `app.js:2439-2461`) không tự lấy/lưu các ô input đang nhập trên form trước khi gửi, và đọc sai trường JSON response (`data.settings` thay vì `data.delivery_status` / `data.telegram_settings`):**
   - Lúc `17:24:56` -> `17:35:33`, khi anh gõ Token/ChatID vào form rồi bấm luôn nút **"📤 Gửi Thử Thông Báo S3 Plan"**, hàm `testTelegramS3Notification()` chỉ gửi `{ task_key }` mà không gửi kèm `bot_token`, `chat_id`, `bot_username` anh vừa gõ, nên backend đọc file cấu hình cũ báo `QUEUED_LOCAL_BRIDGE`.
   - Khi trả kết quả về, `app.js:2451` đọc `data.settings?.last_status || "OK"` (trong khi backend trả `data.delivery_status` và `data.telegram_settings`), làm dòng trạng thái trên UI luôn báo `OK` giả tạo dù thực tế là `DISABLED` hoặc `QUEUED_LOCAL_BRIDGE`.
10. **Cần tự động gửi tin nhắn xác nhận kết nối ngay khi bấm `"Lưu Cấu Hình Telegram"`, hỗ trợ Fallback plain-text nếu `parse_mode: "Markdown"` gặp ký tự `_` lẻ (`telegram_bridge.go:476-498`), và gắn nút bấm Inline Keyboard (`✅ APPROVE & Chạy S4–S5 Ngay`) để vượt qua Group Privacy Mode (`can_read_all_group_messages: False`) của `@tk001sea_bot` trong Supergroup `-1004390214071`.**

## 3. Ranh giới Phạm vi Tuyệt đối (`L003` / `L005`)
- **TUYỆT ĐỐI CẤM** chạm vào `/Users/trainguyen/Documents/work/data-hub/*`.
- Chỉ cập nhật mã nguồn và tạo tác bên trong `/Users/trainguyen/Documents/work/ai-orchestrator` và `/Users/trainguyen/Documents/work/agent`.


