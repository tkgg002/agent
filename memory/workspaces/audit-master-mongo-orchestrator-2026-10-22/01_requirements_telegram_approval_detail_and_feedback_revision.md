---
tier: T2
suffix: telegram_approval_detail_and_feedback_revision
workspace: audit-master-mongo-orchestrator-2026-10-22
status: PROPOSE_ONLY_WAITING_APPROVE
created_at: 2026-10-26T13:27:00+07:00
---

# 01. Requirements: Hiển Thị Đầy Đủ Hồ Sơ Duyệt Giải Pháp S3 (`Nguyên Nhân` + `Giải Pháp` + `Những Thay Đổi`) & Luồng Phản Hồi Chỉnh Sửa Giải Pháp (`Revise Solution`) Trực Tiếp Trên Telegram

## 1. Bối cảnh & Yêu cầu của User
User phản ánh khi dùng lệnh `/task` trên Telegram hoặc khi có Task ở trạng thái `⏸ CHỜ APPROVE` (`NEEDS_YOU`), Bot chỉ gửi một dòng rút gọn cụt lủn:
`• tích hợp Master MongoDB Destination [⏸ CHỜ APPROVE (#1)] — tích hợp Master MongoDB Destination (0/10 Sub-Tasks · 📝 19 files diff)`
rồi ở dưới ghi *"👇 Bấm trực tiếp vào nút Task bên dưới để kích hoạt chạy ngay trên Orchestrator"*.

Yêu cầu bắt buộc của User:
1. **Đầy đủ 3 phần nội dung cốt lõi khi Task ở trạng thái `⏸ CHỜ APPROVE`:**
   - **🔍 Nguyên nhân (`Root Cause Analysis` & `file:line`):** Trích xuất trực tiếp từ `09_tasks_solution.md` (`## 1. Root Cause...`) và `13_analysis.md`.
   - **💡 Giải pháp kỹ thuật & Code Demo (`Solution & Code Preview`):** Trích xuất trực tiếp từ `09_tasks_solution.md` (`## 2. Giải pháp Kỹ thuật...` & `Code Demo`).
   - **📋 Những thay đổi (`Changes`: Danh sách Sub-Tasks `1..N` + Danh sách cụ thể các Files bị ảnh hưởng/diff + Đường dẫn file vật lý trên Mac):**
     - Liệt kê chi tiết từng Sub-Task (`1..N`): `[SubTaskID] (Repo) Title — Context: file:line`.
     - Liệt kê danh sách tên file cụ thể trong `19 files bị ảnh hưởng` (từ `08_tasks.md` / `diff.patch` / `git status`).
     - Hiển thị rõ đường dẫn file vật lý trên đĩa (`.orchestrator/tasks/<safeKey>/09_tasks_solution.md`, `08_tasks.md`, `13_analysis.md`, `diff.patch`).
2. **Hiển thị trực quan trong cả bảng `/task` lẫn Bản tin Chi tiết `⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]`:**
   - Trong bảng `/task`: Bung tóm tắt Nguyên nhân, Giải pháp, Sub-Tasks và Files bị ảnh hưởng ngay dưới dòng `[⏸ CHỜ APPROVE (#N)]`, kèm cặp nút **`[ 👁 Xem Đủ Plan & Duyệt ]`** (`VIEW_PLAN <TaskRef>`) và **`[ ✏️ Sửa Giải Pháp ]`** (`REVISE_PROMPT <TaskRef>`).
   - Khi Task vừa chuyển sang `⏸ CHỜ APPROVE` (hoặc khi User bấm `[ 👁 Xem Đủ Plan & Duyệt ]` / gõ `/plan <Task>`): Bắn bản tin chi tiết **`⏸ [APPROVAL REQUIRED — HỒ SƠ DUYỆT GIẢI PHÁP S3]`** chứa trọn vẹn Nguyên nhân + Giải pháp + Code Demo + Đủ `N` Sub-Tasks + Đủ `M` Files thay đổi, kèm 3 nút Inline Keyboard:
     - **`[ ✅ Approve & Chạy S4–S5 Ngay ]`** (`APPROVE <TaskRef>`)
     - **`[ ✏️ Phản hồi / Sửa Giải Pháp ]`** (`REVISE_PROMPT <TaskRef>`)
     - **`[ 📋 Mở Hàng Đợi Task (/task) ]`** (`/task`)
3. **Luồng Phản hồi / Chỉnh sửa Giải pháp trực tiếp trên Telegram khi User KHÔNG Approve (*"ngoài ra nếu tao ko approve, tao muốn phản hổi để fix giải pháp thì sao"*):**
   - Hỗ trợ đầy đủ 3 cách tương tác trên Telegram:
     1. **Bấm nút `[ ✏️ Phản hồi / Sửa Giải Pháp ]` (`REVISE_PROMPT <TaskRef>`):** Bot bật chế độ chờ phản hồi cho Task đó (`PendingRevisionTaskKey` trong `telegram_settings.json` + `force_reply: true` của Telegram) và hướng dẫn User nhắn trực tiếp yêu cầu sửa giải pháp ở tin nhắn tiếp theo.
     2. **Reply (Trả lời) trực tiếp vào tin nhắn Plan của Bot trên Telegram:** Tự động nhận diện Task từ nội dung tin nhắn được Reply và áp dụng nội dung góp ý vào Giải pháp S3.
     3. **Gõ lệnh trực tiếp `/revise <Task> <nội dung>` (hoặc `/feedback <Task> <nội dung>`, `SỬA <Task>: <nội dung>`):** Cập nhật ngay giải pháp trong 1 bước.
   - Khi nhận được phản hồi sửa giải pháp: Hệ thống tự động gọi bộ máy Co-Architect (`applyTelegramSolutionRevision` nối với `revise_solution` / `update_subtasks`), cập nhật trực tiếp vào `.orchestrator/tasks/<safeKey>/09_tasks_solution.md` & `08_tasks.md` & `chat_history.json`, thu hồi `.user-approved` cũ (giữ ở `NEEDS_YOU`), và **bắn ngay bản tin `⏸ [APPROVAL REQUIRED — ĐÃ CẬP NHẬT THEO PHẢN HỒI CỦA BẠN]` mới nhất** lên Telegram để User đọc lại và bấm **`[ ✅ Approve & Chạy S4–S5 Ngay ]`**!

## 2. Tiêu chí Nghiệm thu (Acceptance Criteria — Gates G1–G8)
- **AC1:** Gửi `/task` trên Telegram hiển thị đầy đủ thông tin cho các Task đang `⏸ CHỜ APPROVE` (File Plan, Nguyên nhân tóm tắt, Giải pháp tóm tắt, danh sách Sub-Tasks chính, danh sách tên files bị ảnh hưởng) và nút `[ 👁 Xem Đủ Plan & Duyệt ]` + `[ ✏️ Sửa Giải Pháp ]`.
- **AC2:** Khi Task vào `⏸ CHỜ APPROVE` hoặc khi bấm `[ 👁 Xem Đủ Plan & Duyệt ]` (`VIEW_PLAN`) / gõ `/plan`, Bot gửi đầy đủ hồ sơ **Nguyên nhân + Giải pháp + Code Demo + Đủ `1..N` Sub-Tasks + Đủ danh sách Files thay đổi + Đường dẫn file vật lý** (tự động chia chunk an toàn nếu vượt 3,800 ký tự để không bị cắt cụt).
- **AC3:** Khi User không Approve mà bấm `[ ✏️ Phản hồi / Sửa Giải Pháp ]` rồi nhắn tin góp ý (hoặc Reply tin nhắn / gõ `/revise`), `09_tasks_solution.md` (và `08_tasks.md` nếu thêm/sửa Sub-Task) được cập nhật ngay trên đĩa, đồng bộ vào Co-Architect Studio trên WebUI `:8090`, và gửi lại bản Plan đã cập nhật lên Telegram kèm nút `[ ✅ Approve & Chạy S4–S5 Ngay ]`.
- **AC4:** Hỗ trợ định danh chính xác mọi loại `TaskKey` trên Telegram (kể cả `CDC-RECON-01` nhiều dấu gạch ngang hoặc `tích hợp Master MongoDB Destination` có khoảng trắng tiếng Việt, thông qua cả `TaskKey` lẫn mã ngắn `#<ID>` an toàn dưới giới hạn 64 bytes của Telegram `callback_data`).
