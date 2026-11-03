---
tier: T2
suffix: fix_coarchitect_chat_answering_root_cause_and_solution
---

# 13. Phân tích Chuyên sâu (S2 Investigate): Nguyên Nhân Gốc Rễ Của `CDC-RECON-01` (`Shadow -> Master`) & Lỗi Không Phản Hồi Trong `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`)

## PHẦN I: VÌ SAO KHI CHẠY RECON SHADOW -> MASTER (`CDC-RECON-01`) KẾT QUẢ KHÔNG ĐÚNG? (`VERIFIED file:line`)

Qua điều tra trực tiếp mã nguồn trên 3 repositories (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) và dữ liệu runtime thực tế (đã ghi nhận tại [`13_analysis_shadow_master_update_drop.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-shadow-master-update-drop-2026-10-20/13_analysis_shadow_master_update_drop.md) và [`CDC-RECON-01/09_tasks_solution.md`](file:///Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/CDC-RECON-01/09_tasks_solution.md)), có **8 sơ hở gốc rễ (`F1` -> `F8`)** làm kết quả đồng bộ & đối soát `Shadow -> Master` bị sai lệch:

1. **🔴 Sơ hở `F1` (`centralized-data-service` — Rớt `DELETE` do lỗi SQL `42703` & `binarySearchSplit` vứt bỏ nguyên mẻ `UPDATE`):**
   - **Điểm chạm:** [`master_ddl_generator.go:137-143, 411`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L137-L143), [`transmuter.go:672-688, 429-431`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L672-L688), [`transmute_handler.go:691-712`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L691-L712).
   - **Cơ chế lỗi:** DDL bảng Master luôn chạy `DROP COLUMN IF EXISTS "_source_id"`, nhưng `TransmuterModule.RunEvents()` và `Run()` vẫn hardcode `DELETE FROM %s WHERE _source_id = ?`, gây nổ lỗi `ERROR: column "_source_id" does not exist (SQLSTATE 42703)`. Khi lỗi rơi vào `TransmuteHandler.binarySearchSplit`, vì mỗi chu kỳ flush thường chỉ gửi `len(batch) == 1` NATS message chứa mảng nhiều `Events`, hàm `binarySearchSplit` không chia đôi mảng `Events` bên trong message đó mà đánh dấu nguyên message là `transmute_poison_pill` rồi ACK bỏ luôn, làm **rớt cả các sự kiện `UPDATE` hợp lệ đi chung mẻ**!

2. **🔴 Sơ hở `F2` (`centralized-data-service` — Split-Brain `_gpay_id` giữa `Run()` dùng Sonyflake và `RunEvents()` dùng SHA-256 làm mù Recon & Heal Segment B):**
   - **Điểm chạm:** [`transmuter.go:720, 1174, 1439`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L720), [`recon_stream_bucket_engine.go:715-748`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go#L715-L748), [`recon_execute_heal_handler.go:653-734`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L653-L734).
   - **Cơ chế lỗi:**
     - Đường batch `Run()` lấy `_gpay_id` là số **Sonyflake** từ bảng Shadow (`deterministicGpayID`), trong khi đường sự kiện NATS `RunEvents()` lại tính `_gpay_id` bằng **băm SHA-256 của `_source_id`** (`deterministicSourceGpayID`).
     - Khi `bulkUpsertMaster` chạy `UPDATE`, nó lại bỏ qua không cập nhật cột `_gpay_id` (`if k == conflictTarget || k == "_gpay_id" { continue }`).
     - **Hậu quả khi chạy Recon Shadow -> Master (Segment B):** Bộ đối soát `resolveFieldsB` ưu tiên so sánh theo `pkCol = "_gpay_id"`, nên dù cùng 1 bản ghi `_id` đã có trên cả Shadow và Master nhưng một bên mang `_gpay_id` Sonyflake, một bên mang `_gpay_id` SHA-256 -> báo lệch giả hàng nghìn dòng (`missing_in_shadow` + `missing_in_master`: `1,482` dòng trên `payment_bills`, `520` dòng trên `export_jobs`).
     - **Hậu quả khi chạy Execute Heal Segment B:** `resolveSourceIDsForSegmentB` lấy `_gpay_id` (SHA-256) trên Master đem truy vấn `SELECT "_source_id" FROM shadow WHERE "_gpay_id" IN (...)` (vốn lưu Sonyflake) -> trả về **0 dòng**, khiến Heal báo `completed` nhưng số dòng lệch không giảm!

3. **🟠 Sơ hở `F3` (`centralized-data-service` — Quét Incremental `_gpay_id > lastGpayID` bỏ sót 100% `UPDATE` của dòng cũ & Heal Segment A không kích hoạt Transmute sang Master):**
   - **Điểm chạm:** [`transmuter.go:330-364, 978-984`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L330-L364), [`recon_heal.go:548-570, 679-737`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L548-L570).
   - **Cơ chế lỗi:** Trên bảng Shadow, `_gpay_id` chỉ sinh khi `INSERT` và giữ nguyên khi `UPDATE`. Nhưng `TransmuterModule.Run()` ở chế độ incremental chỉ lọc `WHERE ("_gpay_id")::bigint > currentLastID`, nên mọi bản ghi cũ được `UPDATE` (`_gpay_id <= currentLastID`) đều bị bỏ sót. Đồng thời khi `ReconHealer.HealMissingIDs` / `HealWindow` (Segment A: Mongo -> Shadow) ghi trực tiếp vào Shadow, nó không bắn NATS `cdc.cmd.transmute-shadow` sang Master.

4. **🟠 Sơ hở `F4` & `F5` (`centralized-data-service` — Đảo thứ tự `UPDATE` trong cùng cửa sổ Flush & Rớt cột khi `records[0]` thiếu key):**
   - **Điểm chạm:** [`batch_buffer.go:728-753`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L728-L753), [`transmuter.go:1412`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412).
   - **Cơ chế lỗi:** `BatchBuffer.batchUpsert` chia nhóm theo `getSignature(r)` trước khi khử trùng lặp theo `PrimaryKeyValue` và duyệt Go `map` ngẫu nhiên; `bulkUpsertMaster` chỉ lấy danh sách cột từ `sortedKeysAny(records[0])` làm rớt cột của các bản ghi sau nếu `records[0]` không có cột đó.

5. **🟠 Sơ hở `F6`, `F7`, `F8` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web` — Cộng nhầm `DELETE` vào `Updated`, thiếu cột `Delete(d)` của Master và bỏ sót `transmute_poison_pill` trên bảng Đối soát):**
   - **Điểm chạm:** [`transmuter.go:708-726, 449, 1409`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L708-L726), [`daily_flow.go:36-83`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/system/daily_flow.go#L36-L83), [`activity_log_read_repo_gorm.go:420, 444-550`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go#L420), [`ReconPipelineGrid.tsx:840-844, 1309-1361`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx#L840-L844).
   - **Cơ chế lỗi:** Khi xóa dòng trên Master thành công, `TransmuterModule` lại tăng `res.Updated++` (không có trường `Deleted`), khối **4. GHI VÀO DB MASTER** trên CMS/UI không có cột `Delete(d)`, và câu SQL `GetDailyFlow` chỉ lọc `al.operation IN ('transmute', 'transmute_sub')` mà bỏ sót `'transmute_poison_pill'` khiến `dlq_count = 0` và báo `BALANCED` sai sự thật!

---

## PHẦN II: VÌ SAO `BRAIN · ARCHITECT` TRONG `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`) KHÔNG PHẢN HỒI ĐÚNG CÂU HỎI CỦA USER? (`VERIFIED file:line`)

Tại [`internal/webui/task_chat.go:1325-1399`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L1325-L1399), hàm `buildContextualDiscussReply` chỉ kiểm tra đúng 2 nhánh `if` rất hẹp:
- Nhánh 1 (`L1350`): `if strings.Contains(lowerMsg, "mày là ai") || strings.Contains(lowerMsg, "model nào") || strings.Contains(lowerMsg, "role nào")`
- Nhánh 2 (`L1365`): `if strings.Contains(lowerMsg, "cảm nhận") || strings.Contains(lowerMsg, "antigravity") || strings.Contains(lowerMsg, "so với")`

Với mọi câu hỏi kỹ thuật khác của User (như *"rồi vì sao chạy ko đúng, giải pháp là gì"*), hàm này hoàn toàn **không đọc** `09_tasks_solution.md`, `13_analysis.md`, `audit_report.md` hay workspace `/agent`, mà rơi thẳng xuống `L1388-1398` để in ra đoạn văn bản khuôn mẫu vô nghĩa (nhại lại câu hỏi của User + in đường dẫn file).

---

## PHẦN III: VÌ SAO DUYỆT `SUB-TASK 1/10 (TASK-M01)` CỦA `TÍCH HỢP MASTER MONGODB DESTINATION` MÀ `GIT STATUS` CHƯA CÓ CODE THAY ĐỔI & KHI NÀO CÓ? (`VERIFIED file:line`)

1. **Nguyên nhân 1 (`synthesizeIsolatedRemediationDiffIfApproved` tại [`internal/webui/telegram_bridge.go:2999-3108`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L2999-L3108) & `subtask_implement` tại [`internal/webui/task_inspector.go:3064-3085`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L3064-L3085)):**
   - Khi anh bấm duyệt `Sub-Task 1/10 (TASK-M01)` ở chế độ **Duyệt Từng Sub-Task**, hệ thống chạy `subtask_approve -> subtask_implement -> subtask_qa` (`telegram_bridge.go:1791-1793`).
   - Tại `subtask_implement` (`task_inspector.go:3076`), hàm `synthesizeIsolatedRemediationDiffIfApproved` hiện đang giữ cơ chế **"Isolated Artifact Diff Only (`External repo untouched per L003`)"** (`telegram_bridge.go:3068-3084`): nó chỉ tách bản vá của `TASK-M01` từ `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/diff.patch` (`99,896 bytes`) rồi ghi vào file nội bộ `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/subtasks/TASK-M01/diff.patch` (`12,174 bytes`), **hoàn toàn KHÔNG chạy `git apply` để ghi code xuống working tree của `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`**!
   - Tiếp đó, tại `subtask_qa` (`task_inspector.go:3096`), hàm `VerifySubTaskCodeOnDisk` (`task_inspector.go:1453-1466`) chỉ kiểm tra thấy tên file `internal/api/master/master_registry_handler_connections.go` có trong `diff.patch` là đánh dấu `candidateOK = true` và báo `DONE` luôn mà không kiểm tra `git status --porcelain` trên repo đích.

2. **Nguyên nhân 2 (Khi nào có code thay đổi theo code hiện tại vs Sau khi sửa):**
   - **Theo code hiện tại:** Kể cả khi anh duyệt `1/10` hay duyệt hết `10/10` Sub-Tasks, code vẫn **chỉ nằm trong `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/diff.patch` và `subtasks/TASK-M01..M10/diff.patch`**, không bao giờ tự động apply vào `/Users/trainguyen/Documents/work/data-hub/...` nên `git status` luôn trống.
   - **Theo cơ chế đúng chuẩn (sau khi bổ sung `applyApprovedSubTaskPatchToTargetRepo`):** **Ngay tại thời điểm anh bấm `Approve Sub-Task k/10`** (bước `subtask_implement`), hệ thống chạy `git -C <TargetRepo> apply --3way` cho đúng các file thuộc Sub-Task `k/10` đó xuống `/Users/trainguyen/Documents/work/data-hub/<repo>`, nên vừa duyệt xong Sub-Task nào là `git status` trên repo đích hiện ngay file code đã sửa của Sub-Task đó!

3. **Nguyên nhân 3 (`scout.MaskSecrets` tại [`internal/scout/scout.go:21-43`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/scout/scout.go#L21-L43) & [`internal/webui/server.go:404`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L404) mask nhầm chuỗi format code trong `diff.patch`):**
   - `uriCredPattern` đã biến chuỗi format Go `postgres://%s:%s@%s` và `mongodb://%s:%s@%s` thành `postgres://***:***@%s` (dòng `355, 376, 417, 543, 618, 638` của `diff.patch`), còn `kvCredPattern` biến biểu thức TS `password: connForm.password,` thành `password:***` (dòng `927, 980, 998`). Khi khôi phục 2 mẫu này và chạy `git apply --3way`, toàn bộ **16/16 file thật** trên 3 repo (`cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`) đều `git apply --check` đạt **`OK 100%`**.

---

## PHẦN IV: VÌ SAO CÁC WAVE CỦA SUB-TASK ĐANG CHẠY BỊ CẮT XÉN (CHỈ 3/5 BƯỚC) & CẦN CHẠY ĐỦ 12 BƯỚC / 6 WAVE NHƯ TASK CHA? (`VERIFIED file:line`)

Anh bắt cực kỳ chuẩn xác! Khi đối chiếu log thực tế trong `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/terminal.log` và code tại [`internal/webui/task_inspector.go:2989-3215`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L2989-L3215) cùng [`internal/webui/telegram_bridge.go:1680-1894`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L1680-L1894), em xin nhận lỗi thẳng thắn: **hiện tại vòng đời Wave của từng Sub-Task đang bị làm tắt lười biếng chỉ còn 3 bước (`Step 2/5 -> Step 3/5 -> Step 4/5`) thay vì chạy đầy đủ 12 Bước / 6 Wave chuẩn như một Task**:

1. **Lỗ hổng 1 — Nhảy cóc qua toàn bộ Wave Pre-Approval (`S0 -> S1 -> S2 -> S2.5 DG0 -> S3 + DG1/DG2`) trước khi trình duyệt Sub-Task (`internal/webui/telegram_bridge.go:1710-1745, 1831-1868`):**
   - Khi chuyển vào `STEP_SUBTASKS` cho `Sub-Task 1/10 (TASK-M01)` hoặc khi chuyển tiếp sang `Sub-Task 2/10 (TASK-M02)`, hàm `handleTelegramApproveSubTaskCommand` chỉ ghi chữ `NEEDS_YOU` vào `.subtask-stage` mà **không hề kích hoạt `subtask_plan` (`Step 1`: `S0 Boot H1 -> S1 Intake -> S2 Deep Repo Scan -> S2.5 Pre-Plan Critique DG0 (:8200) -> S3 Solution & Code Demo -> Laya DG1/DG2 (:8200)`)**.
   - Vì vậy trong `terminal.log`, `TASK-M01` hoàn toàn **không có `Step 1/5`**, mà nhảy thẳng vào **`Step 2/5 · 45%`**!

2. **Lỗ hổng 2 — Cắt xén Wave Thực thi `S4` (`subtask_implement` tại `internal/webui/task_inspector.go:3064-3085`):**
   - Một Task chuẩn ở `S4` phải chạy: **(a) `Worktree & Git Restore-Point (Rule #19)`**, **(b) `Real Code Implementation & Git Apply on TargetRepo Working Tree`**, và **(c) `Gate H4b Zero-Mockup (*Fn), H3 Secret Scan & Laya DG2 (:8200)`**.
   - Nhưng `subtask_implement` tại `task_inspector.go:3064-3085` chỉ in đúng 1 dòng `Step 3/5 · 70%`, không tạo Restore-Point riêng cho Sub-Task trên `st.TargetRepo`, không apply patch thật xuống `st.TargetRepo`, và không quét `H4b/H3` trên các file Go của Sub-Task đó!

3. **Lỗ hổng 3 — Cắt xén Wave Kiểm định `S5` & Bỏ qua luôn Wave Đóng phiên `S6` (`subtask_qa` tại `internal/webui/task_inspector.go:3086-3174`):**
   - Tại `subtask_qa` (`Step 4/5 · 90%`), code chỉ gọi `VerifySubTaskCodeOnDisk` (kiểm tra chuỗi tên file trong `diff.patch`) chứ **không chạy lệnh biên dịch/kiểm thử thật (`buildTargetedRepoCheckCommand` / `go vet` qua `Gatekeeper.RunEvidenceCommand` sinh biên lai `EV-...`) trên `st.TargetRepo`** như `5A Hard Automated Gate` của Task cha (`server.go:1710-1739`).
   - Nghiêm trọng hơn, sau `Step 4/5 · 90%`, code đánh dấu `DONE` luôn mà **bỏ qua hoàn toàn `Step 5/5 (100% — S6 Finalize, Governance Audit H7 & OutputContract Handoff)`**!

---

## PHẦN V: KIỂM CHỨNG TRỰC TIẾP SUPERGROUP `Ai-Orchestrator` (`-1004390214071`) & KIẾN TRÚC TẠO SUB-CHAT (FORUM TOPIC) CHO MỖI TASK (`VERIFIED file:line`)

1. **Bằng chứng kiểm chứng trực tiếp Telegram Bot API (`getChat` trên `-1004390214071`):**
   - Nhóm `Ai-Orchestrator` (`chat_id: -1004390214071`, `type: "supergroup"`) đã trả về **`"is_forum": true`** và anh đã mở quyền cho phép tạo Topic.
2. **Hiện trạng trong [`internal/webui/telegram_bridge.go:367-386, 2653-2702`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L2653-L2702):**
   - Hiện tại toàn bộ lệnh `sendMessage` (`L2669-2672`) chỉ gửi `{"chat_id": chatID, "text": text}` mà **không có `message_thread_id`**, khiến tất cả các Task (`tích hợp Master MongoDB Destination`, `CDC-RECON-01`, `TASK-10`, ...) dồn chung vào 1 luồng chat duy nhất, rất rối khi vừa duyệt 10 Sub-Tasks vừa thảo luận kiến trúc.
   - Struct `telegramIncomingMessage` (`L367-386`) cũng chưa đọc trường `message_thread_id`, nên khi anh nhắn tin trong một Topic con thì bot chưa tự biết anh đang nói về Task nào.
3. **Cơ chế hoạt động của Giải pháp Sub-Chat (`createForumTopic` + `message_thread_id`):**
   - **Mỗi Task = 1 Sub-Chat Topic độc lập (`ensureTelegramTaskTopic`):** Khi bất kỳ Task nào gửi thông báo S3 Plan, mở chế độ duyệt từng Sub-Task (`1/10 .. 10/10`), chạy Wave, hoặc báo Done, hệ thống tự động kiểm tra `cfg.TaskThreadIDs[qt.ExternalID]` (và `.orchestrator/tasks/<SafeTaskKey>/telegram_topic.json`). Nếu chưa có, gọi `POST /bot<TOKEN>/createForumTopic` với tên `📦 [<Repo>] #<ID> <Title>` để tạo ngay 1 Sub-Chat riêng cho Task đó và lưu `message_thread_id`.
   - **Định tuyến 2 chiều theo `message_thread_id`:**
     - **Chiều đi (Outbound):** Mọi bản tin của Task `qt` (S3 Plan Dossier, Card duyệt từng Sub-Task `1/10 -> 10/10`, kết quả Wave 1..5, báo cáo `S4-S5 COMPLETE`, trả lời Co-Architect) đều tự động truyền `"message_thread_id": threadID` để vào đúng Sub-Chat của Task đó (nếu Topic cũ bị xóa trên Telegram thì tự tạo lại Topic mới và gửi tiếp). Riêng bảng tổng hợp `/task` ở sảnh chung sẽ hiển thị kèm mã `💬 Sub-Chat Topic` của từng Task.
     - **Chiều về (Inbound):** Khi anh gõ bất kỳ câu hỏi (`"rồi vì sao chạy ko đúng, giải pháp là gì"`), góp ý sửa giải pháp, hoặc gõ `approve` / bấm nút ngay bên trong Sub-Chat của Task nào, bot đọc `msg.MessageThreadID`, tra ngược ra đúng `TaskKey` của Sub-Chat đó (không cần gõ `#1` hay tên Task!) và phản hồi ngay trong Sub-Chat đó!

---

## PHẦN VI: GIẢI PHẪU 4 VẤN ĐỀ ANH VỪA PHÁT HIỆN TRÊN `tích hợp Master MongoDB Destination` (`Step 15/17` Treo, `39 Runs` Lặp, Retry Mù 3 Lần Không Tạo `TASK-BUG-*` & Cơ Chế Core Nạp Ground-Truth Tranh Biện Với `/laya_router`) (`VERIFIED file:line`)

### 1. Vì sao chạy tới `Step 15/17 · 95%` bị treo và báo lỗi?
- **Bằng chứng vật lý (`internal/webui/server.go:1824-1848, 1929-2081, 3297-3359` & `.orchestrator/tasks/t-ch-h-p-Master-MongoDB-Destination/terminal.log`):**
  1. **Nguyên nhân 1A — Lỗi truyền nhầm thư mục `./test` (không có file `.go` trực tiếp) vào `go build` (`internal/webui/server.go:3345-3351`):**
     - Trong `subtasks.json`, `TASK-M10` có trường `TargetFiles: "centralized-data-service/test/ & giao diện CMS."`.
     - Khi chạy `Step 13/17` (`S5 QA`), hàm `buildTargetedRepoCheckCommand` (`server.go:3345-3351`) kiểm tra `os.Stat("/Users/trainguyen/Documents/work/data-hub/centralized-data-service/test")`. Vì thấy `fi.IsDir() == true`, nó lập tức nhét `./test` vào câu lệnh `go build`:
       ```bash
       go build ./internal/handler/recon ./internal/service/master ./internal/service/recon ./internal/service/source ./test
       ```
     - Tuy nhiên, thư mục `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/test` trên đĩa chỉ chứa 3 thư mục con (`e2e/`, `integration/`, `manual/`) chứ **không có bất kỳ file `.go` nào nằm trực tiếp trong `test/`**!
     - Do đó `go build` trả về lỗi giả:
       `FAILED (EV-20261026T163929-840e): no Go files in /Users/trainguyen/Documents/work/data-hub/centralized-data-service/test`
       khiến `blocking` nhảy từ `0` lên `1` (`server.go:1845-1848`), mặc dù toàn bộ `10/10 Sub-Tasks = PASS` và các cổng `S0-Constitution=PASS | DG1+DG2=PASS | H4b-ZeroMock=PASS | H5-NoGitPush=PASS`!
  2. **Nguyên nhân 1B — Code dừng ở `Step 15/17 · 95%` mà quên cài đặt `Step 16/17` và `Step 17/17` (`internal/webui/server.go:1929-2081`):**
     - Tại `server.go:1929`, `case "qa"` gọi `s.advancePacedWorkflowStep(taskKey, "qa", 15, 17, 95, ...)` rồi kết thúc hàm luôn mà **chưa hề có lệnh chạy `Step 16/17 (98% — S6 Finalize & Governance Linter H7)` và `Step 17/17 (100% — Task Complete)`**, đồng thời không tắt `s.liveStepProgress.Active = false` khi thoát hàm! Vì vậy thanh tiến trình bị kẹt cứng ở `15/17 · 95%`.

### 2. Vì sao `Phase 2` sinh ra `39 runs` trùng lặp và tự lặp `Step 10 -> 11 -> 13 -> 15` 3 lần liên tiếp trong 35 giây (`16:39:18 -> 16:40:10`)?
- **Bằng chứng vật lý (`internal/webui/server.go:1284-1332, 1759, 2055` & `internal/webui/telegram_bridge.go:3111-3199`):**
  1. Tại `runTaskLifecycleActionWithDelay` (`server.go:1284-1332`), vòng lặp `for loop := 1; loop <= maxSelfHealLoops; loop++` (`maxSelfHealLoops = 3`) đang bị viết mâu thuẫn nghiêm trọng:
     - Ở `loop = 1` (`16:39:18`), khi `qa` bị `blocking = 1` (do lỗi `./test` ở trên), `server.go:2072` gọi `s.triggerS5ToS3RePlan(ctx, qt, 1, 3, qaLogs, missingOnDisk)`.
     - Hàm `triggerS5ToS3RePlan` (`telegram_bridge.go:3162-3197`) gửi tin nhắn lên Telegram: *"Đã gửi Hồ Sơ Duyệt Giải Pháp S3... Bấm Approve"*.
     - **NHƯNG** ngay khi `triggerS5ToS3RePlan` chạy xong, luồng điều khiển quay về `server.go:1311-1319`: thay vì **dừng lại chờ anh bấm Approve** (hoặc tự sửa lỗi `./test` rồi mới chạy lại), dòng `if loop < maxSelfHealLoops { continue }` lại **tự động chạy luôn `loop = 2` (`16:39:35`) và `loop = 3` (`16:39:52`)** với câu lệnh `go build ... ./test` y hệt!
     - Sau 3 lần chạy mù trong 35 giây (mà anh chưa hề bấm Approve lần nào!), nó rơi xuống `server.go:1321-1331` và tự kích hoạt `AtomicRollbackWaveSubTasks(..., 1, "Circuit Breaker tripped after 3/3 retries")`, tự ý rollback oan `TASK-M04` và `TASK-M05`!
  2. Mỗi vòng retry như vậy lại gọi `AppendAutopilotRound` ở cả `S4` (`server.go:1759`) lẫn `S5` (`server.go:2055`) mà không chống trùng lặp (`deduplicateAutopilotRounds`), làm phình lịch sử lên **`39 runs`** (`Human Gate` lặp 12 dòng, `Build · S4: block` lặp 18 dòng).

### 3. Vì sao khi có bug phải retry mà không tạo Sub-Task `TASK-BUG-*` ghi rõ lỗi gì rồi tự fix?
- **Bằng chứng vật lý (`internal/webui/telegram_bridge.go:3126-3131`):**
  - Hiện tại trong `triggerS5ToS3RePlan` (`telegram_bridge.go:3126-3131`), code chỉ duyệt:
    ```go
    for _, st := range subTasks {
        if !strings.EqualFold(st.Status, "done") { ... }
    }
    ```
  - Vì cả `10/10` Sub-Task (`TASK-M01 .. TASK-M10`) đều đã có `Status == "done"`, biến `missingIDs` và `missingBullets` bị **rỗng hoàn toàn (`""`)**, còn biến `qaLogs` (chứa lỗi thật `EV-20261026T163929-840e: no Go files in .../test`) thì **bị bỏ rơi không được đọc**!
  - **Giải pháp chuẩn (`diagnoseAndCreateS5BugSubTask` theo Rule #9.8d):**
    - Khi `S5` phát hiện bất kỳ lỗi nào (`qaLogs` / compiler `EV-...` / diff thiếu), hệ thống **BẮT BUỘC**:
      1. Bóc tách chính xác dòng lỗi từ `qaLogs` + mã biên lai `EV-...` + package/file gây lỗi.
      2. Tự động khởi tạo một Sub-Task sửa lỗi tường minh **`TASK-BUG-01` (hoặc `TASK-BUG-02`)** trong `subtasks.json`, `08_tasks.md` và `09_tasks_solution.md` với đầy đủ: **`[Mã lỗi EV-... + File/Package + Nguyên nhân gốc rễ + Hành động Fix]`**, đồng thời in rõ lên `terminal.log` và Sub-Chat Telegram.
      3. Tự động thực thi hành động sửa lỗi cho `TASK-BUG-XX` ngay ở `S4` rồi chạy lại `S5` để đóng `TASK-BUG-XX -> DONE` — **tuyệt đối không bắn thông báo đòi User `Approve` giả trong lúc đang tự động sửa lỗi, và tuyệt đối không tự nổ Circuit Breaker rollback oan khi User chưa duyệt!**
      4. **Kỷ luật "Task phải làm cho xong" (Persistent Until-Done Retry — Không dừng ở 3 lần nếu vẫn tìm ra `TASK-BUG-*`):**
         - Ngay cả khi đã chạy qua 3 vòng retry (`loop > 3`), **hễ hệ thống vẫn bóc tách và tìm ra được một `TASK-BUG-XX` mới có thể xử lý (`TASK-BUG-04`, `TASK-BUG-05`, ... hoặc chữ ký lỗi `bugSignature` đã tiến triển/giảm số lượng lỗi so với vòng trước)** thì hệ thống **BẮT BUỘC reset `sameBugStallCount = 0` và TIẾP TỤC RETRY (`continue`)**!
         - Hệ thống sẽ tự động lặp chu trình `tạo TASK-BUG-XX -> sửa ở S4 -> verify ở S5` cho tới khi sạch bóng bug (`qaBlocking == 0`) và hoàn thành 100% Task (`Step 17/17 · DONE`).

### 4. Vì sao phải làm giống `/agent` Core — nơi Core Agent nạp Full Context để tranh biện với `/laya_router` theo Ground Truth (`Rule #10`)?
- Anh chỉ đạo hoàn toàn chính xác theo **Hiến pháp `/agent` (`GEMINI.md` Rule #9.8e & Rule #10)**:
  - Trong `/agent` Core, **Ground Truth = Code vật lý trên đĩa + Runtime/Compiler thật (`EV-...`)**, còn `/laya_router` (`:8200`) là **Hội đồng Phản biện Neural (Advisory Critic)** để phát hiện điểm mù.
  - Khi `/laya_router` trả về `FLAG`, `UNVALIDATED` hoặc điểm thấp, Core Agent **không bao giờ để Laya chặn mù một bản build đã pass vật lý**, mà Core Agent phải **soạn đầy đủ Context (`REF: <file>:<line>` thực tế trên đĩa + biên lai compiler `EV-... exit=0` + `git diff`) để tranh biện/phản biện lại Laya (`debateAndResolveLayaWithGroundTruth` tương đương `laya_gate.py resolve`)**, và ghi biên bản phản biện kèm bằng chứng vật lý vào `04_decisions.md` + `audit_report.md`!

---

## PHẦN VII: GIẢI PHẪU VÌ SAO TẠO TASK MỚI (`Core-01`) CHƯA TỰ TẠO SUB-TOPIC TELEGRAM THEO `ExternalID` & VÌ SAO BẢN PLAN `S3` CỦA `Core-01` BỊ NÔNG CẠN, LỖI FONT `TASK-CẬ01` VÀ KHÓ ĐỌC (`VERIFIED file:line`)

### 1. Vì sao khi tạo Task mới (`Core-01`) lại chưa có Sub-Topic mới trên Telegram mang Mã Task (`External ID`)?
- **Bằng chứng vật lý (`internal/webui/server.go:827-960` — `handleIntake`):**
  1. Khi anh bấm tạo Task mới trên WebUI (`POST /api/intake`), hàm `handleIntake` nhận `req.ExternalID` (`"Core-01"`) và `req.Title` (`"Cập nhật và Nâng cấp chính mày trong ai-orchestrator từ api cho tới ux,ui"`).
  2. Tại `internal/webui/server.go:921-935`, `handleIntake` chỉ thực hiện 3 việc:
     - Ghi task vào SQLite Queue (`s.Queue.EnqueueRaw(...)` tại `L921`),
     - Tạo nhánh worktree cô lập (`s.ensureTaskIsolation(...)` tại `L929`),
     - Gọi `s.generateDeepTaskArtifacts(r.Context(), qt)` tại `L932` để ghi file vào `.orchestrator/tasks/Core-01/`.
  3. **Điểm khuyết (`server.go:933-945`):** `handleIntake` **chưa hề có lệnh gọi Telegram Bot API `createForumTopic`** với tên lấy từ `req.ExternalID` (`🧩 [Core-01] Cập nhật và Nâng cấp chính mày trong ai-orchestrator từ api cho tới ux,ui`), cũng như chưa bắn thông báo khởi tạo Task + Hồ sơ `S3 Plan` vào `message_thread_id` của Sub-Topic đó! (Hơn nữa ở lượt trước anh mới chỉ `APPROVE` 2 mục của `TASK-CHAT-03`, còn `TASK-CHAT-05` vẫn đang ở trạng thái `Propose-Only` chờ duyệt nên code tạo Forum Topic chưa được phép ghi xuống đĩa theo Rule #13).

### 2. Vì sao Task `Core-01` ("Cập nhật và Nâng cấp chính mày trong `ai-orchestrator` từ api cho tới ux,ui") lại nhận được bản Plan `S3` vừa rối rắm khó đọc, vừa nông cạn vô thưởng vô phạt (`TASK-CẬ01`, `TASK-CẬ02` trỏ bừa vào `gate.go:33`)?
Em đã truy vết trực tiếp file `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/Core-01/09_tasks_solution.md` và mã nguồn bộ sinh Plan trong `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go`. Bản Plan đó **không phải do model `gemini-4-argon` suy luận kiến trúc thật**, mà bị một **template tĩnh cào bằng trong `task_inspector.go`** sinh ra với **4 lỗi thiết kế nghiêm trọng**:

1. **Lỗi 1 — Bộ quét file bị "mù" toàn bộ tầng UX/UI (`.html`, `.css`, `.js`) và luôn rơi về 2 file đầu bảng chữ cái (`internal/agentgate/gate.go:33` & `kernel_bridge.go:22`):**
   - Tại `internal/webui/task_inspector.go:3274`:
     ```go
     if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".py" && ext != ".sql" {
         return nil
     }
     ```
     Bộ quét **bỏ qua hoàn toàn `.html`, `.css`, `.js`**, dẫn tới toàn bộ tầng giao diện WebUI của `ai-orchestrator` (`internal/webui/static/index.html`, `internal/webui/static/app.css`, `internal/webui/static/app.js`) bị tàng hình 100% dù đề bài ghi rõ *"từ api cho tới ux,ui"*!
   - Tệ hơn, tại `internal/webui/task_inspector.go:3308-3315`:
     ```go
     if strings.Contains(relLower, "service") || strings.Contains(relLower, "handler") || strings.Contains(relLower, "worker") || strings.Contains(relLower, "queue") || strings.Contains(relLower, "server") || strings.Contains(relLower, "agentgate") {
         score += 4
     }
     ```
     Vì từ khóa tiếng Việt trong đề bài (`"cập"`, `"nhật"`, `"nâng"`, `"cấp"`, `"chính"`, `"mày"`) không khớp tên hàm tiếng Anh nào trong code, mọi file trong `internal/agentgate/` và `internal/webui/` đều đồng hạng `score = 4`. Khi gọi `sort.SliceStable` (`L3322`), Go giữ nguyên thứ tự duyệt thư mục theo bảng chữ cái (`a` đứng trước `w`: `internal/agentgate/` đứng trước `internal/webui/`)!
   - Hệ quả: `hits[0]` **luôn luôn** là `internal/agentgate/gate.go:33 (type LiveGateBadge struct {)` và `hits[1]` **luôn luôn** là `internal/agentgate/kernel_bridge.go:22 (type TripwireEntry struct {)` — hoàn toàn trật lất so với yêu cầu nâng cấp từ API tới UX/UI!

2. **Lỗi 2 — Cắt chuỗi theo Byte UTF-8 làm hỏng mã Sub-Task thành `TASK-CẬ01` và đóng khung cứng 2 Sub-Task khẩu hiệu sáo rỗng (`internal/webui/task_inspector.go:1076-1157`):**
   - Tại `task_inspector.go:1079-1082`:
     ```go
     w := strings.ToUpper(words[0]) // words[0] = "Cập" -> "CẬP"
     if len(w) > 4 {
         w = w[:4] // Cắt 4 byte đầu: 'C' (1 byte) + 'Ậ' (3 bytes) = "CẬ"
     }
     prefix = "TASK-" + w // -> "TASK-CẬ" -> TASK-CẬ01, TASK-CẬ02!
     ```
     Thay vì dùng chính mã chuẩn `qt.ExternalID` (`Core-01` -> `CORE-01-S01`), hàm lại lấy chữ đầu của tiêu đề tiếng Việt rồi cắt theo byte `w[:4]`, đẻ ra cái mã kỳ dị **`TASK-CẬ01`, `TASK-CẬ02`**.
   - Tại `task_inspector.go:1118-1156`, với mọi Task không có chữ `"mongo"`, hàm **chỉ đóng khung đúng 2 Sub-Task rỗng tuếch** bằng cách ghép nguyên văn câu tiêu đề của anh vào mẫu câu khẩu hiệu:
     - `TASK-CẬ01`: *"Cập nhật logic cốt lõi cho Cập nhật và Nâng cấp chính mày trong `ai-orchestrator` từ api cho tới ux,ui trong `ai-orchestrator`"* -> nhét bừa vào `gate.go:33`.
     - `TASK-CẬ02`: *"Tích hợp dây dẫn & Kiểm thử nghiệm thu DoD trên `ai-orchestrator`"* -> nhét bừa vào `kernel_bridge.go:22`.
     Không hề có bất kỳ sự phân rã nào vào các tầng thật của `ai-orchestrator` (REST/SSE API `server.go`, Deep Planner `task_inspector.go`, Co-Architect Chat `task_chat.go`, Telegram Forum Bridge `telegram_bridge.go`, WebUI `index.html`/`app.css`/`app.js`).

3. **Lỗi 3 — Phần "Code Demo" (`buildConcreteCodeDemoForTask`) chỉ có nhánh `if "mongo"`, còn lại thì dump 2 đoạn struct cũ giống hệt nhau bị cắt cụt (`internal/webui/task_inspector.go:3714-3785`):**
   - Tại `task_inspector.go:3716`: `if strings.Contains(lower, "mongo") { ... }` được viết riêng cho bài toán MongoDB.
   - Với mọi bài toán khác (như `Core-01`), code rơi xuống `L3774-3785`: lấy `hits[0].Snippet` (`gate.go:33`) và `hits[1].Snippet` (`gate.go:40`) in ra 2 lần liên tiếp cùng một `type Gatekeeper struct` cũ kỹ trong `gate.go`, thậm chí dòng cuối còn bị cắt ngang chữ (`ApprovalCh chan`)!

4. **Lỗi 4 — Trình bày (`09_tasks_solution.md`) nặng tính "rác template", lặp từ và rất khó đọc (`internal/webui/task_inspector.go:3433-3439, 3629-3655`):**
   - Mở đầu đập vào mắt 8 dòng bullet metadata khô khan (`Task Key`, `Suffix`, `Worktree`, `CLI Runner`, `Active Role`, `Decision Gate`, `Trạng Thái Cổng H4b`).
   - Mục 1 chỉ nói chung chung *"Đã hoàn tất Wave Phản Biện... quét 121 file... phân rã thành 2 Sub-Tasks"* mà **không chỉ ra được hiện tại `ai-orchestrator` đang yếu/lỗi ở đâu từ API tới UX/UI**.
   - Mục 2 và mục `Clean-Room Approved Spec` lặp đi lặp lại câu tiêu đề dài 15 chữ tới 6 lần mà không có bảng kiến trúc (Layer Matrix) hay giải pháp cụ thể.

---

## PHẦN VIII — ĐIỀU TRA GỐC RỄ (`S2 VERIFIED`): VÌ SAO `@tk001sea_bot` ĐÃ LÊN QUYỀN ADMIN NHƯNG KHÔNG TRẢ LỜI KHI NHẬP PHẢN HỒI & VÌ SAO NỘI DUNG TELEGRAM BỊ ĐỔ NHƯ RÁC (`TASK-CHAT-10`)

### 8.1. Vì sao `@tk001sea_bot` đã được nâng quyền Admin (`status: administrator`, `can_manage_topics: true`) nhưng khi anh nhập cập nhật phản hồi thì Bot không trả lời? (`VERIFIED` 100%)
1. **Nguyên nhân 1 (`VERIFIED` từ Telegram API `getMe` + `internal/webui/telegram_bridge.go:2240-2266, 2052-2085`) — BotFather Group Privacy Mode (`can_read_all_group_messages: false`) + Thiếu `ForceReply` Markup:**
   - Kết quả gọi trực tiếp Telegram API `getMe` và `getChatMember` trên nhóm `Ai-Orchestrator` (`-1004390214071`):
     - `getChatMember`: `"status": "administrator"`, `"can_manage_topics": true`.
     - `getMe`: `"username": "tk001sea_bot"`, **`"can_read_all_group_messages": false`**.
     - `getUpdates?offset=127823183`: **`0` updates mới** được Telegram Server gửi về sau lúc `10:51:25`!
   - Trong kiến trúc Telegram Bot API, **việc nâng quyền Bot lên `administrator` trong một Supergroup KHÔNG tự động tắt `can_read_all_group_messages: false` (Group Privacy Mode của BotFather)**. Khi `can_read_all_group_messages: false`, máy chủ Telegram **CHỈ chuyển tiếp tin nhắn trong nhóm tới Bot (`getUpdates`) khi thỏa 1 trong 3 điều kiện**:
     - (a) Tin nhắn đó là **Reply (`reply_to_message`) trực tiếp vào một tin nhắn của `@tk001sea_bot`** (điều này diễn ra **tự động 100%** nếu tin nhắn nhắc nhập liệu của Bot có gắn **`reply_markup: {"force_reply": true, "selective": true, "input_field_placeholder": "..."}`**!), HOẶC
     - (b) Tin nhắn bắt đầu bằng dấu `/` (như `/revise ...`, `/task`), HOẶC
     - (c) Tin nhắn có nhắc tên `@tk001sea_bot`.
   - Thế nhưng tại `handleTelegramRevisePromptCommand` (`internal/webui/telegram_bridge.go:2252-2264`) và `handleTelegramReviseSubPromptCommand` (`telegram_bridge.go:2071-2083`), khi anh bấm nút **`✏️ Phản hồi / Sửa Giải Pháp`**, code lại gửi tin nhắn kèm `inline_keyboard` (`[ 👁 Xem lại Plan ] [ ❌ Hủy sửa ]`) thay vì `ForceReply` (`{"force_reply": true}`)!
   - Vì gửi `inline_keyboard` chứ không gửi `force_reply: true`, ứng dụng Telegram của anh **không tự động bật ô Reply vào `@tk001sea_bot`**. Khi anh gõ câu phản hồi bình thường vào ô chat rồi bấm Gửi (không bấm Reply vào tin nhắn Bot), **Telegram Server chặn luôn bản tin đó ở tầng server (`0` updates về `getUpdates`)**!

2. **Nguyên nhân 2 (`VERIFIED` tại `internal/webui/telegram_bridge.go:645-662, 844-898, 2201-2206, 2885-2890`) — Kẹt `PendingRevisionTaskKey` ở task cũ (`cdc-01`), không cắt `@tk001sea_bot` ở câu thường, và đánh rơi tin nhắn thường thành `IGNORED`:**
   - Trong `.orchestrator/telegram_settings.json`: lúc `10:43:47` anh bấm `REVISE_PROMPT` cho `cdc-01` (`#44`) nên `pending_revision_task_key = "cdc-01"`. Lúc `10:51:25` anh mở `VIEW_PLAN` cho `Core-01` (`#43`), nhưng `handleTelegramViewPlanCommand` (`telegram_bridge.go:2201-2206`) chỉ cập nhật `LastNotifiedTask = "Core-01"` mà giữ nguyên `PendingRevisionTaskKey = "cdc-01"`.
   - Tại `processIncomingTelegramCommandWithReply` (`telegram_bridge.go:880-898`):
     - Kiểm tra `PendingRevisionTaskKey` (`L880`) **trước** `replyRef` (`L893`), nên nếu anh Reply vào thẻ của `Core-01` (`#43`), code lại đẩy nhầm phản hồi sang `cdc-01` (`#44`)!
     - Hàm `stripBotUsernameFromCommand` (`L645-662`) có điều kiện `if !strings.HasPrefix(trimmed, "/") { return trimmed }`, nên nếu anh tag `@tk001sea_bot <nội dung sửa>`, chuỗi `@tk001sea_bot` không được lọc bỏ!
     - Nếu `PendingRevisionTaskKey == ""` và câu nhắn là câu chỉ đạo sửa giải pháp (không có dấu `?`, không phải câu hỏi trong `isTelegramTechnicalQuestion`), code rơi xuống `L896` và **bỏ qua luôn (`IGNORED`)** thay vì cập nhật giải pháp cho `LastNotifiedTask`!
   - Tại `ensureTelegramApprovePoller` (`telegram_bridge.go:2885-2890`), biến `cfg` được nạp 1 lần ở đầu vòng lặp (`L2850`), nên khi có nhiều update liên tiếp, `s.saveTelegramPersistedConfig(cfg)` ở `L2888` ghi đè mất `PendingRevisionTaskKey` / `LastNotifiedTask` vừa lưu ở update trước!

### 8.2. Vì sao nội dung gửi qua Telegram hiện tại bị đổ tràn lan "như rác, sao mà đọc"? (`VERIFIED` 100%)
1. **Nguyên nhân 3 (`VERIFIED` tại `internal/webui/telegram_bridge.go:1442-1545`) — Lệnh `/task` xả toàn bộ hồ sơ S3 của mọi task vào chung 1 tin nhắn khổng lồ:**
   - Tại `handleTelegramTaskListCommand` (`L1495-1540`), với mỗi task đang ở `⏸ CHỜ APPROVE` (`cdc-01`, `Core-01`...), `/task` in tràn lan đường dẫn file tuyệt đối `/Users/trainguyen/...`, thanh tiến độ, 3 dòng Root Cause, 4 dòng Giải pháp, toàn bộ 10 dòng Sub-Tasks và 10 file diff ngay bên trong danh sách `/task`! Kết quả là `/task` dài hơn 4,500 ký tự, bị băm thành nhiều mảnh `(Phần 1/2)`, `(Phần 2/2)`.
2. **Nguyên nhân 4 (`VERIFIED` tại `internal/webui/telegram_bridge.go:1002-1281` & `internal/webui/telegram_forum_topic.go:205`) — Đổ nguyên bảng Markdown `| ... |` của `UDF-v1`, đường dẫn tuyệt đối `/Users/...` và hướng dẫn dài dòng vào Telegram:**
   - Sau khi nâng cấp `UDF-v1` cho WebUI (`TASK-CHAT-09`), `09_tasks_solution.md` và `buildContextualDiscussReply` dùng bảng Markdown 5 cột (`| Mã | Wave | ... |`) và tiêu đề `## 1..4`.
   - Telegram `parse_mode: "Markdown"` **KHÔNG hỗ trợ bảng `| ... |` hay tiêu đề `##`**: toàn bộ bảng bị vỡ nát thành hàng chục dấu gạch đứng `| :--- |` rối mù trên màn hình điện thoại!
   - Trong `buildTelegramApprovalRequiredMessage` (`L1200-1251`), bản tin in 3 đường dẫn dài `/Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/...`, 10 dòng Sub-Task lặp lại cả mô tả lẫn đường dẫn, cộng thêm 3 đoạn văn hướng dẫn dài 12 dòng ở cuối tin nhắn, khiến mỗi lần xem Plan hay cập nhật phản hồi lại bắn ra 2 tin nhắn `(Phần 1/2)`, `(Phần 2/2)` dày đặc chữ.

---

## PHẦN IX — ĐIỀU TRA GỐC RỄ (`S2 VERIFIED`): 6 LỖ HỔNG KỸ THUẬT & QUẢN TRỊ KHIẾN BÁO CÁO CỔNG `H4b` BỊ NHIỄU, SAI SỐ LIỆU FILE VÀ LỆCH TECH STACK (`TASK-CHAT-11`)

1. **Lỗ hổng 1 (`VERIFIED` tại `internal/webui/server.go:2771-2808` — `inspectTaskRepoDiffFilesAndSummary`) — Đếm trùng file thay đổi (`8 files` thay vì `6 files`):**
   - Tại dòng `2785-2795` (`diff.patch`) và dòng `2818-2822` (`git status --short`), hàm `addFile` nạp đường dẫn tương đối không có tên repo (`models/trans-his/trans-his.model.ts`, `dbHandler/postgres.ts`).
   - Nhưng tại dòng `2796-2807` (quét `subTasks`), dòng `2801` lại ghép thêm `st.TargetRepoName + "/"` (`centralized-export-service/models/trans-his/trans-his.model.ts`, `centralized-export-service/dbHandler/postgres.ts`).
   - Do `seenFiles` không strip tiền tố `<repoName>/` trước khi kiểm tra trùng lặp, 2 file trên bị đếm 2 lần (`6 + 2 = 8 files`).

2. **Lỗ hổng 2 (`VERIFIED` tại `internal/webui/task_inspector.go:1149, 1171`, `internal/webui/task_llm_bridge.go:56-83` & `internal/webui/telegram_message_format.go:542`) — Hardcode ````go` và `go test -race ./...` cho repo TypeScript (`centralized-export-service`):**
   - `task_inspector.go:1171` gắn cứng chuỗi `TestDoD: "Pass go test -race ./... (hoặc npm run build)..."` mà không kiểm tra `package.json` vs `go.mod` trong `repoPath`.
   - `task_llm_bridge.go:56-83` (`detectCodeFenceLang`) trả về `"ts"` thay vì `"typescript"` và khi `relPath == ""` (như tại `unified_doc_format.go:243`), nếu snippet bắt đầu bằng `class TransHisModel {` thì rơi về mặc định `"go"` tại dòng `82`.

3. **Lỗ hổng 3 (`VERIFIED` tại `internal/webui/task_inspector.go:3284-3362` & `.orchestrator/tasks/TASK-13/09_tasks_solution.md:18-30`) — Trích xuất God-Class Snippet thay vì Key Diff / Strategy Interface:**
   - Khi sinh `09_tasks_solution.md` cho `TASK-13`, `buildConcreteCodeDemoForTask` (`task_inspector.go:3351-3362`) lấy nguyên 15 dòng đầu của `trans-his.model.ts:10-30` (chứa cả `dbConnection`, `pgPool`, `knexPg`, `isPostgres`, `schema` nhồi chung trong `class TransHisModel`) thay vì sinh **Key Diff / Code Preview** chuẩn Strategy/Router giữ nguyên signature hàm cho tầng Service.

4. **Lỗ hổng 4 (`VERIFIED` tại `internal/webui/task_inspector.go:1140, 1162` & `.orchestrator/tasks/TASK-13/08_tasks.md:8, 12`) — Vi phạm thứ tự phụ thuộc Wave (`M01` và `M02` cùng nằm ở `Wave 0`):**
   - Trong `DecomposeMultiRepoCodebaseSubTasks`, cả `id1` (`M01`: Xây dựng Model/Repo V2) tại dòng `1140` lẫn `id2` (`M02`: Triển khai Strategy/Router & Integration Test) tại dòng `1162` đều được gán `WaveIndex: rIdx` (`Wave 0`), dẫn đến `M02` không có `DependsOn: ["TASK-13-M01"]` và bị xếp chạy song song ở `Wave 0` thay vì `Wave 1`.

5. **Lỗ hổng 5 (`VERIFIED` tại `internal/webui/unified_doc_format.go:206-254` & `internal/webui/telegram_message_format.go:350-454`) — Thiếu hoàn toàn mục Kiểm soát Rủi ro Vận hành (`Operational & Migration Risks`) và Cơ chế Rollback/Toggle:**
   - Hồ sơ `09_tasks_solution.md` và thẻ Telegram hiện tại chỉ liệt kê Sub-Task mà không có mục **`⚠️ 2. KIỂM SOÁT RỦI RO (RISK & GUARDRAILS)`** (Memory/Streaming/Cursor chống OOM khi export bảng lớn `trans_his`, tương thích Schema/DTO) và không chỉ rõ **Cơ chế Rollback / Toggle** (biến môi trường/cờ điều hướng và mặc định Fallback về V1).

6. **Lỗ hổng 6 (`VERIFIED` tại `internal/webui/unified_doc_format.go:231-235` & `internal/webui/telegram_message_format.go:245, 362-364, 384, 398`) — Văn mẫu Meta-System rườm rà và cắt cụt `...` ở các định danh quan trọng:**
   - `BuildUnifiedS3SolutionDoc` in các câu meta-system (*"Đã quét trực tiếp 347 file mã nguồn..."*, *"Mọi thay đổi ở bước S4 được cô lập trong Git Worktree theo Rule #12, #13, #19..."*) và lặp đường dẫn tuyệt đối `/Users/trainguyen/...`.
   - `formatTelegramCompactApprovalCard` gọi `truncateRunes(..., 68..110)` cắt cụt tên nhánh, tên file và mô tả Sub-Task bằng dấu `...`.

---

## PHẦN X — ĐIỀU TRA GỐC RỄ (`S2 VERIFIED`): VÌ SAO GỬI ĐỊNH HƯỚNG CHO AGENT Ở CMS WEB (`#co-architect-studio`) KHÔNG PHẢN HỒI LẬP TỨC MÀ PHẢI BẤM `F5`? (`TASK-CHAT-13`)

### 10.1. Bằng chứng vật lý từ Runtime Log & `.orchestrator/tasks/Core-01/chat_history.json:107-167`
- Trong file [`.orchestrator/tasks/Core-01/chat_history.json`](file:///Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/Core-01/chat_history.json#L135-L167), lượt gửi định hướng gần nhất của anh trên CMS Web được ghi nhận chính xác từng nano-giây:
  - **`userEntry` (`msg-u-1793088399249300000`):**
    - `content`: `"kiểm tra lại xem nãy giờ đã làm đc gì rồi, sao tao hỏi = tele thì mày lại không sumnary"`
    - `created_at`: **`2026-10-27T15:06:39+07:00`** (`15:06:39.249`)
  - **`agentEntry` (`msg-a-1793088487992726000`):**
    - `created_at`: **`2026-10-27T15:08:07+07:00`** (`15:08:07.992`)
  - **Tổng thời gian treo cứng HTTP `POST /api/tasks/chat`:** **`88.74 giây` (`1 phút 29 giây`)**!

### 10.2. 4 Nguyên Nhân Gốc Rễ (`file:line` VERIFIED 100%)

1. **Nguyên nhân 1 (`VERIFIED` tại `internal/webui/static/app.js:1856-1892` — `sendCoArchitectChatMessage` xóa trắng ô nhập liệu nhưng KHÔNG có Optimistic UI, KHÔNG có trạng thái "Đang suy nghĩ...", và KHÔNG khóa nút gửi):**
   - Tại `app.js:1866`: Ngay khi bấm nút **`💬 Gửi Định Hướng`**, dòng `ta.value = "";` xóa sạch nội dung anh vừa gõ trong `#inp-co-architect-msg`.
   - Nhưng từ dòng `1868` đến `1888`, hàm đứng chờ `const res = await fetch("/api/tasks/chat", ...)` suốt **`88.74 giây`** mới gọi `renderAll()` (`app.js:1888`).
   - Trong suốt `88.74 giây` đó:
     - Tin nhắn anh vừa gửi **không được đẩy (optimistic append)** vào `#co-architect-chat-messages` (`dashboardState.active_task_chat`).
     - **Không có bong bóng trạng thái loading/thinking** (`"⏳ BRAIN đang phân tích..."`) và nút `.btn-cas-send` không hề đổi trạng thái.
     - Kết quả: Ô nhập chữ biến mất, khung chat đứng im phăng phắc suốt gần 1 phút rưỡi khiến người dùng thấy hệ thống hoàn toàn không phản hồi!
     - Ngoài ra, `index.html:432` ghi placeholder `(Ctrl+Enter để gửi nhanh)` nhưng trong `app.js` hoàn toàn chưa gắn sự kiện `keydown` cho `#inp-co-architect-msg`.

2. **Nguyên nhân 2 (`VERIFIED` tại `internal/webui/task_chat.go:807-923`, `internal/webui/task_revise_workflow.go:125, 196`, `internal/webui/task_llm_bridge.go:138` & `internal/webui/static/app.js:86-96` — Không lưu `userEntry` xuống đĩa trước khi gọi LLM + Chặn kép 2 lần gọi LLM `40s + 40s + 8s = 88.74s` + Bị SSE `fetchDashboard()` ghi đè):**
   - **Lỗi không lưu `userEntry` ngay:** Tại `task_chat.go:807`, `history = append(history, userEntry)` chỉ nằm trên RAM của request HTTP đó, còn lệnh `s.saveTaskChatHistory(selected.TaskKey, history)` nằm tận dòng `913` (sau khi LLM chạy xong!).
   - **Lỗi chặn kép 2 vòng LLM (`88.74s`):**
     - Lần 1: `renderContextualDiscussReply` (`task_chat_knowledge.go:32`) hoặc `executeArchitectReviseSolution` (`task_revise_workflow.go:196`) gọi `s.invokeArchitectLLM` (timeout `40s` tại `task_llm_bridge.go:138`).
     - Lần 2: Ngay sau đó, `syncArchitectLLMAnalysisToMasterPlan` (`task_revise_workflow.go:125`) lại gọi đồng bộ `s.generateDeepTaskArtifacts(...)` (`task_inspector.go:2987`), hàm này tiếp tục gọi `s.invokeArchitectLLM` lần thứ 2 (`40s`) + chạy Laya Gate (`~8s`) ngay trên cùng luồng HTTP của khung chat!
   - **Lỗi SSE nền ghi đè và không broadcast khi xong:** Trong suốt `88.74s` chờ đợi, bất kỳ sự kiện SSE nào (`step_progress`, `update`, Telegram poller) cũng kích hoạt `fetchDashboard()` (`app.js:86-96`). Vì `chat_history.json` trên đĩa lúc đó chưa có `userEntry`, `fetchDashboard()` nạp lại lịch sử cũ; và đến giây thứ `88.74` khi `chat_history.json` được lưu ở `task_chat.go:913`, hàm `handleTaskPlanningChat` lại **không phát `s.broadcast("update", ...)`**. Nếu kết nối `fetch("/api/tasks/chat")` của trình duyệt bị nghẽn/race với `fetchDashboard()`, giao diện sẽ giữ nguyên dữ liệu cũ cho tới khi anh bấm **`F5`** (gọi lại `GET /api/dashboard` đọc file `chat_history.json` đã lưu lúc `15:08:07`).

3. **Nguyên nhân 3 (`VERIFIED` tại `internal/webui/static/unified_doc_format.js:348-355, 376-385` vs `internal/webui/static/app.js:1853` — Lệch chiều cao cuộn `scrollTop` sau khi `UDF-v1` biến đổi DOM):**
   - Tại `app.js:1853`, `renderCoArchitectStudio()` gán `listEl.scrollTop = listEl.scrollHeight` khi nội dung tin nhắn còn là thẻ `<pre class="cas-msg-body">` thô.
   - Ngay sau đó, wrapper `wrappedCas` trong `unified_doc_format.js:379` chạy `applyUnifiedFormatToAllViews()` biến đổi `<pre>` thành các thẻ `.udf-section-card` cao hơn nhiều lần nhưng **không cập nhật lại `listEl.scrollTop = listEl.scrollHeight`**, khiến phần tin nhắn mới nhất ở đáy bị trôi khuất xuống dưới khung nhìn.

4. **Nguyên nhân 4 (`VERIFIED` tại `.orchestrator/tasks/Core-01/chat_history.json:107-133`, `internal/webui/telegram_bridge.go:2052-2085` & `ps -p 10934` — Tiến trình `:8090` chạy bản binary cũ từ `14:32:17` & Nhánh Revise Telegram thiếu khối `Summary`):**
   - Tiến trình `PID 10934` (`./bin/orchestrator ui --addr 127.0.0.1:8090`) được bật từ lúc **`14:32:17`** và chưa được restart sau các bản cập nhật code lúc `14:50–15:03`.
   - Đồng thời, lúc `15:05:34` (`Core-01/chat_history.json:107-133`), khi anh hỏi `"kiểm tra lại xem nãy giờ đã làm đc gì rồi"` trên Telegram, do `PendingRevisionTaskKey` đang giữ `"Core-01"`, tin nhắn đi vào `handleTelegramReviseSolutionCommand` của bản binary cũ (`14:32`) vốn chỉ ghi câu xác nhận tĩnh mà không có khối **`📌 Tóm tắt & Trả lời trực tiếp (Summary)`**.

---

## PHẦN XI — ĐIỀU TRA GỐC RỄ (`S2 VERIFIED`): VÌ SAO BOX DETAIL CỦA `💡 S3 Solution & Code Demo (Gate Stop) (Core-01)` KHÔNG FULL HEIGHT? (`TASK-CHAT-14`)

1. **Nguyên nhân 1 (`VERIFIED` tại `internal/webui/static/app.css:1831-1874` & `internal/webui/static/index.html:335-369` — `#sri-markdown-content` bị ép cứng `max-height: 465px` bên trong cột Grid `.sri-markdown-pane` cao `~780px–880px`):**
   - Tại [`internal/webui/static/index.html:335-369`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/index.html#L335-L369) và [`internal/webui/static/app.css:1831-1846`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1831-L1846), khối `.sri-body` dùng CSS Grid 2 cột (`grid-template-columns: 1.15fr 1fr; gap: 10px;`).
   - Mặc định Grid có `align-items: stretch`, nên cột trái (`.sri-markdown-pane`, chứa tiêu đề `💡 S3 Solution & Code Demo (Gate Stop) (Core-01)` và box detail `#sri-markdown-content`) được kéo giãn cao bằng cột phải (`.sri-side-pane`, chứa 3 khối: `#sri-code-demo-content` + `#subtask-fractal-inspector` + `#sri-subtasks-list`, tổng chiều cao thực tế **`~780px–880px`**).
   - Thế nhưng tại [`internal/webui/static/app.css:1857-1874`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1857-L1874):
     ```css
     .sri-pre-block, #sri-markdown-content {
       ...
       flex: 1 1 auto;
       min-height: 260px;
       max-height: 465px;
       overflow-y: auto;
     }
     ```
   - Dòng **`1871` (`max-height: 465px;`)** khóa chặt chiều cao tối đa của `#sri-markdown-content` ở mức `465px` (vô hiệu hóa `flex: 1 1 auto;` khi khung cha cao hơn `465px`). Kết quả là cái box detail dừng cụt lủn ở giữa chừng (`465px`), để lộ một khoảng trống đen sì ~300px–400px bên dưới đáy của `.sri-markdown-pane`!

2. **Nguyên nhân 2 (`VERIFIED` tại `internal/webui/static/unified_doc_format.css:6-16` & `internal/webui/static/app.css:1886, 1900-1922` — Stylesheet `UDF-v1` chưa gỡ `max-height: 465px` và cột phải `.sri-subtasks-box` thiếu `flex: 1 1 auto`):**
   - Trong [`internal/webui/static/unified_doc_format.css:6-16`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/unified_doc_format.css#L6-L16), rule `#sri-markdown-content.udf-doc-root` chỉ ghi đè font/padding/background mà không ghi đè `max-height: none !important; height: 100% !important;`.
   - Ngược lại, ở cột bên phải (`.sri-side-pane`), `.sri-code-block` ([`app.css:1886`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1886)) bị khóa `max-height: 145px;` và `.sri-subtasks-list` ([`app.css:1919`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1919)) bị khóa `max-height: 220px;` mà `.sri-subtasks-box` ([`app.css:1900`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/static/app.css#L1900)) không có `flex: 1 1 auto;`, nên khi nội dung báo cáo bên trái dài hơn cột phải thì cột phải lại bị hụt chiều cao ở dưới đáy.
