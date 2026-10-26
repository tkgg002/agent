---
tier: T2
suffix: fix_repo_task_queue_sidebar_ui_ux
workspace: audit-master-mongo-orchestrator-2026-10-22
project: ai-orchestrator
created_at: 2026-10-26T14:14:00+07:00
updated_at: 2026-10-26T14:25:30+07:00
---

# 01. Requirements — Luồng Thực Thi Fractal `N` Sub-Tasks (`Approve All` vs `Approve Từng Sub-Task` qua Telegram/WebUI) & Khắc phục UI/UX `Hàng Đợi Task (Theo Repo)` (`✈️ /task`)

## 1. Bối cảnh & Yêu cầu của User
1. **Yêu cầu 1 (UI/UX Sidebar & nút `✈️ /task`):**
   > *"Hàng Đợi Task (Theo Repo) / ✈️ /task : ui/ux đang lỗi rồi"*
2. **Yêu cầu 2 (Cốt lõi — Luồng hoàn thành 1 Task có $N$ Sub-Tasks & Điều khiển tinh tế qua Telegram):**
   > *"và thật sự là mày vẫn chưa hoàn thành xong 1 flow task. tao ví dụ nha, tao thêm 1 task, và task nó có 10sub task thì mày chạy như thế nào. subtask ko cần review approve à. mày phải note qua tele cực kì tinh tế vào. ko đùa đc. ở task khi tơia bước implement chạy 10 task, thì tao đã bảo là 10 subtask nó cũng vận hành như task vậy, thì các wave của nó cũng phải tương tự, phải cần approve, hoặc khi tới bước implement thì phải cho tai 2 chức năng, aprove all, hoăc tao sẽ aprove tứng sub task. sau khi xong hết 10 cái subtassk thì mày mới tiếp tục các bước wave tiếp theo của task. mày hiểu không"*

## 2. Yêu cầu Chức năng & Kiến trúc Hệ thống (Functional & Architectural Requirements)

### R1 — Hai Chế độ Duyệt & Chạy Sub-Tasks tại Bước Implement (`Approve All` vs `Approve Từng Sub-Task`)
Khi một Task cha (`Parent Task`, ví dụ có `10 Sub-Tasks` chia theo `Wave 0, Wave 1, Wave 2`) hoàn tất `S3 Plan` và bước vào giai đoạn **Implement (`S4`)**:
1. **Chế độ 1 — `⚡ Approve All Sub-Tasks` (`APPROVE_ALL #<ID>` / `approve_all`):**
   - Duyệt giải pháp S3 của Task cha và cho phép chạy tự động toàn bộ `10/10` Sub-Tasks tuần tự theo đúng thứ tự Wave (`Wave 0 -> Wave 1 -> Wave 2`).
   - Mỗi Sub-Task vẫn đi qua đầy đủ chu trình (`Sub-Task Plan -> Sub-Task Approve -> Sub-Task S4 Implement -> Sub-Task S5 5A/5B QA -> DONE`).
   - Gửi thông báo tiến độ tổng hợp tinh tế qua Telegram (`⚡ [WAVE 0..W SUB-TASKS EXECUTION LEDGER]`) và **chỉ sau khi cả `10/10` Sub-Tasks đạt `DONE`** mới tự động chuyển sang các bước Wave tiếp theo của Task cha (`Parent S5 Adversarial Audit -> Laya DG3 -> S6 DONE`).
2. **Chế độ 2 — `🧩 Approve Từng Sub-Task (Step-by-Step 1..10)` (`STEP_SUBTASKS #<ID>` -> `APPROVE_SUB #<ID> <SubTaskID>`):**
   - Mỗi Sub-Task (`1/10 .. 10/10`) vận hành độc lập như một Task hoàn chỉnh theo Wave (`Wave 0`, `Wave 1`, `Wave 2`), bắt buộc qua cổng Review & Approve riêng của Sub-Task đó.
   - Khi bấm **`🧩 Approve Từng Sub-Task (1/10 -> 10/10)`**, hệ thống mở khóa cổng S3 của Task cha để bước vào pha **`S4_SUBTASKS`**, dừng ở Sub-Task đầu tiên chưa xong (`[1/10]` thuộc `Wave 0`) với trạng thái `NEEDS_YOU` (`⏸ CHỜ DUYỆT SUB-TASK 1/10`), và bắn ngay **Thẻ Review Sub-Task Tinh Tế** lên Telegram.
   - Khi User bấm **`[ ✅ Approve Sub-Task k/10 (<SubTaskID>) & Chạy ]`** trên Telegram:
     - Hệ thống chỉ thực thi `S4 Implement -> S5 5A/5B QA & Laya DG3` cho đúng Sub-Task `k/10` đó.
     - Nếu `k < 10` (chưa xong hết 10 Sub-Tasks): Báo cáo hoàn tất Sub-Task `k/10` (`✅ DONE k/10`) và **tự động gửi ngay Thẻ Review của Sub-Task `(k+1)/10` tiếp theo** (tuân thủ thứ tự Wave) lên Telegram để User duyệt tiếp.
     - Hỗ trợ nút **`[ ✏️ Sửa Plan Sub-Task k/10 ]`** (`REVISE_SUB_PROMPT #<ID> <SubTaskID>`) để User phản hồi sửa riêng giải pháp của Sub-Task `k/10`, và nút **`[ ⚡ Approve All Còn Lại ]`** nếu User muốn duyệt chạy tự động nốt các Sub-Task còn lại.

### R2 — Rào chắn Cứng (`Parent S4 -> S5 Wave Barrier`): Xong hết `10/10` Sub-Tasks mới chạy tiếp các Wave sau của Task cha
- Tuyệt đối **CẤM** cho phép Task cha chạy sang các bước Wave sau bước Implement (`Parent S5 Mandatory Adversarial Audit Hook`, `Parent Laya DG3`, `Parent S6 Close / DONE`) khi số Sub-Tasks hoàn thành `doneCount < totalSubTasks` (`< 10/10`).
- Chỉ khi Sub-Task cuối cùng (`10/10`) đạt `DONE` (`doneCount == totalSubTasks`), hệ thống mới tự động kích hoạt các bước Wave tiếp theo của Task cha (`Parent S4 Final Diff Merge -> Parent S5 Adversarial Audit -> Parent Laya DG3 -> Parent S6 DONE`) và bắn thông báo hoàn tất toàn trình lên Telegram!

### R3 — Tinh tế hóa Thông báo Telegram (`Telegram UX Refinement`)
- Bản tin `⏸ [APPROVAL REQUIRED · HỒ SƠ DUYỆT GIẢI PHÁP S3]` và `/task` hiển thị rõ sơ đồ phân bổ Sub-Tasks theo Wave (`Wave 0`, `Wave 1`, `Wave 2`) và cung cấp đầy đủ 2 nút chế độ duyệt:
  - `[ ⚡ Approve All (Chạy 10/10 Sub-Tasks -> S5) ]`
  - `[ 🧩 Duyệt Từng Sub-Task (1/10 -> 10/10) ]`
- Bản tin `🧩 [SUB-TASK REVIEW · BƯỚC k/10 · WAVE W]` hiển thị thanh tiến độ trực quan (`[■■■□□□□□□□] 3/10`), Repo, Role/Model, điểm chạm `file:line`, Giải pháp & Code Demo của riêng Sub-Task `k`, tiêu chuẩn DoD, và bộ nút bấm điều khiển riêng cho Sub-Task `k`.

### R4 — Khắc phục triệt để lỗi UI/UX khu vực `Hàng Đợi Task (Theo Repo)` và nút `✈️ /task` trên Sidebar WebUI (`:8090`)
- Mở rộng Sidebar trái từ `248px` lên `310px`, chống gãy dòng tiêu đề `Hàng Đợi Task (Theo Repo)` và nút `✈️ /task` (`.mini-tg-btn` màu xanh Telegram, có phản hồi trực tiếp `⏳ Đang gửi... -> ✓ Đã đẩy Tele` ngay trên nút).
- Thêm `flex-shrink: 0` (và bỏ `overflow: hidden` trên `.sb-repo-group`) để danh sách nhóm Repo không bị Flexbox bóp chiều cao; hợp nhất thanh bộ lọc + bộ đếm; hiển thị trên mỗi thẻ Task cả 2 nút nhanh (`⚡ All` / `🧩 Từng Sub`) khi Task đang chờ duyệt.
