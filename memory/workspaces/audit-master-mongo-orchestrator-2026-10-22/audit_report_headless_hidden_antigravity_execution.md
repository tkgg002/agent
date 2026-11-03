---
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: headless_hidden_antigravity_execution
tier: T2
---
# Báo Cáo Audit Phản Biện Toàn Diện (`S5` & `DG3`) — Giải Pháp Chạy Ẩn 100% (`Headless Ephemeral Hidden Execution`) Cho `gemini-4-argon (antigravity)`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `headless_hidden_antigravity_execution`
- **Tier:** `T2`
- **Ngày thực hiện:** 2026-10-28
- **Trạng thái:** `DONE (S0 → S6 Complete — Đã triển khai và kiểm chứng 100% cơ chế chạy ẩn không hiện lên danh sách Conversation của Antigravity IDE và tự động xoá sạch trajectory sau mỗi bước)`

---

## 1. Mục 1: Phản Biện Toàn Bộ "Quá Trình" Vừa Thực Hiện, Từng Task (`TASK-HIDE-01 .. TASK-HIDE-04`), Từng File & Từng Dòng Update (`git diff` Line-by-Line Critique)

### 1.1. Phản Biện Quá Trình & 3 Phát Hiện Tự Kiểm Toán (`SA-HIDE-01 .. SA-HIDE-03` — Vòng Lặp `S5 → S3 → S4 → S5`)

Khi thực hiện Audit phản biện đối kháng toàn bộ quá trình (`TASK-HIDE-01 .. TASK-HIDE-03`) theo đúng yêu cầu của anh, em đã phát hiện **3 điểm vi phạm kiến trúc / kiểm thử (`SA-HIDE-01 .. SA-HIDE-03`)** và đã lập tức kích hoạt vòng lặp tự chữa lành `S5 → S3 → S4 → S5` (`TASK-HIDE-04`) để xử lý dứt điểm:

| Mã lỗi Tự Phản Biện | Hiện tượng Vi phạm / Thiếu sót ở `TASK-HIDE-03` | Bằng chứng `file:line` Trước Khi Sửa | Cách Đã Khắc Phục Triệt Để Trong `TASK-HIDE-04` (`file:line` Sau Khi Sửa) |
|---|---|---|---|
| **`SA-HIDE-01` (File Stuffing / Vi phạm Single Responsibility Principle)** | Ở `TASK-HIDE-03`, em nhồi nhét toàn bộ 160 dòng giao thức `agentapi` và Connect-RPC (`resolveAntigravityLSConnection`, `buildHiddenAntigravityEnv`, `cleanupHiddenAntigravityConversation`, `cleanupLegacySessionFile`, `completeWithAntigravityAgentAPI`) trực tiếp vào `internal/runner/antigravity.go`, làm file phình từ `525` dòng (`HEAD`) lên `713` dòng, và làm `internal/runner/antigravity_test.go` phình từ `117` dòng (`HEAD`) lên `240` dòng. | `internal/runner/antigravity.go:343-657` (`713` dòng `> HEAD 525`) & `internal/runner/antigravity_test.go:124-240` (`240` dòng `> HEAD 117`) | **Tách module đơn trách nhiệm (SRP):** Chuyển toàn bộ giao thức `agentapi` & Connect-RPC ẩn sang file riêng [`internal/runner/antigravity_agentapi_hidden.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden.go#L1-L339) (`339` dòng) và test riêng [`internal/runner/antigravity_agentapi_hidden_test.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden_test.go#L1-L174) (`174` dòng). Đưa [`internal/runner/antigravity.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go#L1-L395) về **`395` dòng (`<= HEAD 525` dòng, giảm `-130` dòng)** và [`internal/runner/antigravity_test.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_test.go#L1-L117) về đúng **`117` dòng (`== HEAD 117` dòng)**. |
| **`SA-HIDE-02` (Dùng Mock Script `/bin/sh` + `httptest.NewServer` Thay Vì E2E Thật Trên `agentapi` + `language_server`)** | Ở `TASK-HIDE-03`, bài test `TestAntigravityRunner_HeadlessEphemeralHiddenExecution` chỉ gọi một script `/bin/sh` giả (`fakeAgyBin`) và `httptest.NewServer` in-memory, còn kiểm chứng runtime chỉ chạy `curl http://127.0.0.1:8090/` lấy HTML trang chủ chứ chưa chạy test E2E tự động gọi thẳng binary `/Users/trainguyen/.gemini/antigravity/bin/agentapi` + `language_server` thật từ Go test. | `internal/runner/antigravity_test.go:124-240` (cũ) | **Xoá bỏ 100% `httptest` & script giả:** Trong [`internal/runner/antigravity_agentapi_hidden_test.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden_test.go#L19-L174), triển khai: (1) `TestAntigravityRunner_RealTCPConnectRPCCleanupAndEnv` dùng TCP socket thật (`net.Listen("tcp", "127.0.0.1:0")`); (2) **`TestAntigravityRunner_RealHostLanguageServerHiddenE2E` (`dòng 85-174`) gọi trực tiếp binary `agentapi` + `language_server` thật trên máy (`EV-20261028T122137-2b6d`)**, xác nhận `hideFromConversationList: true`, stream token thật `"HIDDEN_REAL_OK"`, và xác nhận `get-conversation-metadata` trả về `rpc error: code = NotFound desc = not_found: trajectory not found` sau khi `Complete()` kết thúc. |
| **`SA-HIDE-03` (`cleanupLegacySessionFile` Bỏ Sót `DeleteCascadeTrajectory` Khi Thư Mục `brain/<convID>` Đã Xoá Trên Đĩa)** | Ở `TASK-HIDE-03`, `cleanupLegacySessionFile` gọi `loadValidAgentAPIConversationID(sessionFile)` vốn yêu cầu thư mục `~/.gemini/antigravity/brain/<convID>` phải tồn tại trên đĩa mới trả về `convID`. Nếu thư mục `brain` đã xoá nhưng trajectory vẫn còn trong SQLite của `language_server`, hàm sẽ trả về `""` và bỏ qua không gọi `DeleteCascadeTrajectory`! | `internal/runner/antigravity.go:514-522` (cũ) | Tách hàm [`readRawAgentAPIConversationID`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden.go#L52-L65) và sửa [`cleanupLegacySessionFile`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden.go#L197-L205) gọi trực tiếp `readRawAgentAPIConversationID(sessionFile)` để **luôn gọi `DeleteCascadeTrajectory` lên `language_server` bất kể thư mục `brain/<convID>` còn hay mất trên đĩa**. |

### 1.2. Phản Biện Từng File & Từng Dòng `git diff` (`Line-by-Line Diff Critique`)

1. **[`internal/runner/antigravity.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go#L1-L395) (`NOW = 395` dòng `<= HEAD = 525` dòng, giảm `-130` dòng):**
   - **Dòng `33-39` (`AntigravityRunner`):** Giữ cấu trúc `Model`, `Binary`, `WorkDir`, `SessionFile`, `SessionTitle` ở tầng điều phối Runner.
   - **Dòng `96-123` (`Complete`):** Định tuyến các model Antigravity (`argon`, `antigravity`, `gemini-4`) sang `completeWithAntigravityAgentAPI` (nằm ở module `antigravity_agentapi_hidden.go`) bọc qua `withHeartbeat(ch, defaultHeartbeatInterval)`.
   - **Dòng `321-340` (`findAntigravityBinary`):** Dò tìm binary `agentapi` hoặc `language_server` chính thức trên máy; toàn bộ phần giao thức Connect-RPC & transcript ẩn đã được tách sạch sang `antigravity_agentapi_hidden.go`.

2. **[`internal/runner/antigravity_agentapi_hidden.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden.go#L1-L339) (File mới đơn trách nhiệm `339` dòng):**
   - **Dòng `18` (`hiddenAntigravitySourceMetadata`):** Khai báo hằng số `{"hideFromConversationList":true}` khớp với protobuf `exa.cortex_pb.SourceMetadata.HideFromConversationList` (field 3) của `language_server`.
   - **Dòng `52-65` (`readRawAgentAPIConversationID`):** Đọc và kiểm tra an toàn `convID` từ `sessionFile` (chống ký tự ` \t\n/\\`), không phụ thuộc vào sự tồn tại của thư mục `brain/<convID>` trên đĩa (`SA-HIDE-03`).
   - **Dòng `84-143` (`resolveAntigravityLSConnection`):** Đọc `ANTIGRAVITY_LS_ADDRESS` & `ANTIGRAVITY_CSRF_TOKEN` từ môi trường hoặc tự động dò từ tiến trình `language_server` (`ps -ax -o pid=,command=` + `lsof -nP -a -p <pid> -iTCP -sTCP:LISTEN`).
   - **Dòng `145-168` (`buildHiddenAntigravityEnv`):** Lọc bỏ mọi biến `ANTIGRAVITY_SOURCE_METADATA=` cũ trong `os.Environ()` và ép cứng `ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}`.
   - **Dòng `170-195` (`cleanupHiddenAntigravityConversation`):** Gửi HTTP POST tới `http://<lsAddr>/exa.language_server_pb.LanguageServerService/DeleteCascadeTrajectory` kèm header `x-codeium-csrf-token` (timeout `3s`) và gọi `os.RemoveAll` xoá thư mục `brain/<convID>`.
   - **Dòng `197-205` (`cleanupLegacySessionFile`):** Dùng `readRawAgentAPIConversationID(sessionFile)` để xoá sạch trajectory cũ trên `language_server` ngay cả khi thư mục `brain` trên đĩa đã mất.
   - **Dòng `207-339` (`completeWithAntigravityAgentAPI`):** Khởi tạo phiên ẩn (`hideFromConversationList: true`), stream `PLANNER_RESPONSE` từ `transcript_full.jsonl`, và gọi `doCleanup()` đồng bộ tại dòng `273` (`ctx.Done()`) và dòng `330` (trước `ch <- Token{Done: true}`) kèm `defer doCleanup()` tại dòng `259`.

3. **[`internal/runner/antigravity_test.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_test.go#L1-L117) (`NOW = 117` dòng `== HEAD = 117` dòng):**
   - Giữ nguyên vẹn 100% đúng trách nhiệm kiểm thử factory và discovery cơ bản của `AntigravityRunner`, không nhồi nhét thêm dòng nào vượt `HEAD`.

4. **[`internal/runner/antigravity_agentapi_hidden_test.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden_test.go#L1-L174) (File test mới đơn trách nhiệm `174` dòng — Zero `httptest`, Real Host E2E):**
   - **Dòng `19-83` (`TestAntigravityRunner_RealTCPConnectRPCCleanupAndEnv`):** Mở TCP listener thật (`net.Listen("tcp", "127.0.0.1:0")`), kiểm chứng `buildHiddenAntigravityEnv` và kiểm chứng `cleanupLegacySessionFile` xoá `conv-diskless-legacy-99` qua RPC ngay cả khi không có thư mục `brain` trên đĩa.
   - **Dòng `85-174` (`TestAntigravityRunner_RealHostLanguageServerHiddenE2E`):** Gọi trực tiếp binary `/Users/trainguyen/.gemini/antigravity/bin/agentapi` và `language_server` thật trên máy anh:
     1. Tạo phiên ẩn thật trên `language_server`, gọi `agentapi get-conversation-metadata` xác nhận `"hideFromConversationList": true`.
     2. Ghi `legacyConvID` vào `sessionFile` và xoá thư mục `brain/<legacyConvID>` trên đĩa.
     3. Gọi `AntigravityRunner.Complete()` thật với prompt `"Chỉ trả lời đúng 1 từ duy nhất: HIDDEN_REAL_OK"`, nhận kết quả stream thật từ model.
     4. Gọi lại `agentapi get-conversation-metadata <legacyConvID>` lên `language_server` thật và xác nhận `language_server` trả về `rpc error: code = NotFound desc = not_found: trajectory not found` (chứng minh vật lý 100% trajectory đã bị xoá khỏi `language_server`!).

---

## 2. Mục 2 & Mục 3: Chống Suy Diễn Smoke Test (`Anti-Smoke-Test Speculation`), Concurrency (`WorkerCount > 1`), Timeout-Heartbeat & Watermark Continuity

1. **Multi-Worker Concurrency (`WorkerCount > 1`) & Batch / Isolation:**
   - Khi nhiều Sub-Tasks trong cùng 1 Wave hoặc nhiều Tasks chạy song song (`WorkerCount > 1`), trước đây việc dùng chung 1 file `agentapi_conv_id.txt` và bắn `send-message` đồng thời vào cùng 1 `convID` gây xung đột `prevMaxStep` (worker A đọc nhầm `PLANNER_RESPONSE` của worker B trên cùng `transcript_full.jsonl`).
   - Với kiến trúc **Headless Ephemeral Hidden Execution** trong [`antigravity_agentapi_hidden.go`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity_agentapi_hidden.go#L207-L339): mỗi goroutine gọi `completeWithAntigravityAgentAPI` nhận một `convID` ẩn riêng biệt từ `language_server`, đọc trên thư mục `brain/<convID>` cô lập hoàn toàn của riêng nó, và tự xoá đúng `convID` đó khi kết thúc — **triệt tiêu 100% race condition giữa các worker chạy song song**.
2. **Tương tác `Timeout / Context Cancellation` vs `Heartbeat` (`withHeartbeat`):**
   - Trong `internal/runner/antigravity.go:108-112`, channel `ch` được bọc bởi `withHeartbeat(ch, defaultHeartbeatInterval)`. Khi một bước Wave kéo dài >8s, `withHeartbeat` phát token reasoning keep-alive mà không ảnh hưởng tới vòng lặp đọc `transcript_full.jsonl`.
   - Khi `ctx.Done()` xảy ra do timeout hoặc User bấm Stop (`antigravity_agentapi_hidden.go:272-275`), hàm gọi ngay `doCleanup()` (sử dụng `context.Background()` với timeout `3s` độc lập trong `cleanupHiddenAntigravityConversation` tại dòng `178`) để đảm bảo ngay cả khi `ctx` cha đã bị cancel, request `DeleteCascadeTrajectory` và `os.RemoveAll` vẫn chạy thành công, không bao giờ để rò rỉ phiên mồ côi.
3. **Tính liên tục của Trạng thái / Dọn dẹp Phiên Cũ (`Watermark / Legacy State Continuity`):**
   - Nhờ sửa `cleanupLegacySessionFile` sang dùng `readRawAgentAPIConversationID` (`SA-HIDE-03`), ngay cả khi thư mục `brain/<oldConvID>` trên đĩa đã mất sau sự cố Outage/crash, chỉ cần còn `sessionFile` là `DeleteCascadeTrajectory` vẫn được gọi lên `language_server` để xoá sạch trajectory khỏi SQLite của IDE.

---

## 3. Mục 4: Bằng Chứng Vật Lý & Biên Lai Thực Thi Thực Tế (`EV-...` — Zero Mockup)

Mọi kiểm chứng sau khi tách module `TASK-HIDE-04` đều được thực thi qua `/Users/trainguyen/Documents/work/agent/tooling/evidence_run.py` và lưu biên lai vật lý tại `06_evidence/`:

1. **Biên lai Real Host E2E & Unit Test `EV-20261028T122137-2b6d` (`unit`):**
   - Lệnh: `go test -v -run TestAntigravity ./internal/runner/...`
   - Kết quả: `exit=0`, **5/5 tests `PASS` (`6.210s`)**, trong đó:
     - `TestAntigravityRunner_RealTCPConnectRPCCleanupAndEnv`: `PASS` (TCP socket thật `net.Listen`).
     - `TestAntigravityRunner_RealHostLanguageServerHiddenE2E`: **`PASS` (`5.29s`)** — gọi trực tiếp `/Users/trainguyen/.gemini/antigravity/bin/agentapi` và `language_server` thật trên máy anh, kiểm chứng `hideFromConversationList: true`, nhận token thật `"HIDDEN_REAL_OK"`, và xác nhận `get-conversation-metadata` trả về `NotFound (trajectory not found)` sau khi dọn dẹp.
2. **Biên lai Integration Test `EV-20261028T122155-1783` (`integration`):**
   - Lệnh: `go test -v -run TestApproveAndRunExecutesSubTaskWavesAndUpdatesWorkflowStepDone ./internal/webui/...`
   - Kết quả: `exit=0`, `PASS` (`0.902s`).
3. **Biên lai Runtime Thực Tế `EV-20261028T122213-6aec` (`runtime`):**
   - Lệnh: `curl -fsS http://127.0.0.1:8090/api/dashboard` sau khi build lại `./bin/orchestrator` và khởi động lại daemon `:8090`.
   - Kết quả: `exit=0`, trả về JSON trạng thái thực tế của `TASK-2` (`planner_model: "gemini-4-argon (antigravity)"`, `hard_gate_command: "npm run build && npm test && VerifySubTaskCodeOnDisk"`, `FTS5-BM25: "PASS(355 new)"`).

---

## 4. Mục 5: Vòng Lặp Phản Tỉnh & Khắc Phục (`Self-Improvement Loop`)

- Đã ghi nhận bài học mới **`L001` (`[2026-10-28] Nhồi nhét thêm 160 dòng hàm chạy ẩn agentapi / Connect-RPC vào file internal/runner/antigravity.go đang lớn (621 -> 713 dòng, vượt HEAD = 525 dòng) thay vì tách module đơn trách nhiệm antigravity_agentapi_hidden.go, chỉ dùng script /bin/sh + httptest giả lập trong unit test mà thiếu E2E test gọi agentapi + language_server thật trên máy, và yêu cầu thư mục brain/<convID> phải tồn tại trên đĩa mới gọi DeleteCascadeTrajectory khi dọn sessionFile cũ`)** vào `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` (`wc -l`: `2684 -> 2698` dòng) và chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/build_lessons_index.py` đồng bộ `lessons/INDEX.md` (`355/355` bài học, `index_bytes = 14690 < 15000`).

---

## 5. Mục 6: Kết Quả QC & Nhận Xét Từ Hội Đồng `/laya_router` (`DG0`, `DG1/DG2`, `DG3` Sau Lần Sửa Code Cuối `TASK-HIDE-04`)

1. **Cổng `DG0` Pre-Plan Critique (`06_laya_summary_dg0_20261028T110830.json`):**
   - Trạng thái: **`gate = PASS`** (`C-REQ`, `C-REALITY`, `C-CONTRA` đều đạt `ACCEPT` với confidence `0.9334 .. 0.9639`).
2. **Cổng `DG1` & `DG2` Solution & Code Demo cho `TASK-HIDE-04` (`06_laya_summary_dg1_20261028T121706.json`, `06_laya_summary_dg2_20261028T121706.json`):**
   - Trạng thái: **`gate = PASS`** (`S-FIX`, `S-IMPACT`, `S-PATTERN`, `D-BEHAV`, `D-EDGE`, `D-API` đều đạt `ACCEPT`; `S-RC` đã đối chiếu `REF:` và resolve đạt `PASS`).
3. **Cổng `DG3` Final Audit sau lần sửa code cuối `TASK-HIDE-04` (`06_laya_summary_dg3_20261028T122315.json`):**
   - Trạng thái: **`gate = PASS`** trên cả 3 vòng `r1`, `r2`, `r3`:
     - `A-1` (`buildHiddenAntigravityEnv` trong `antigravity_agentapi_hidden.go:145-168` gắn `ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}`): **`ACCEPT`** (`r1 CORRECT = 0.7795`, `r3 YES = 0.9134`).
     - `A-2` (`cleanupLegacySessionFile` trong `antigravity_agentapi_hidden.go:197-205` dùng `readRawAgentAPIConversationID` để luôn xoá trajectory cũ trên `language_server`): **`ACCEPT`** (`r2 CORRECT = 0.7118`, `r3 YES = 0.8096`).
     - `A-3` (`completeWithAntigravityAgentAPI` trong `antigravity_agentapi_hidden.go:330-338` gọi `doCleanup()` đồng bộ trước `Token{Done: true}`): **`ACCEPT`** (`r2 CORRECT = 0.5668`, `r3 YES = 0.9375`).
4. **Kiểm định Bảo mật (`/security-agent`):**
   - Quét toàn bộ `git diff`: 0 hardcoded secret/token/PII (`H3 PASS`), `convID` được kiểm tra chống ký tự đường dẫn (`strings.ContainsAny(convID, " \t\n/\\")`) tại `readRawAgentAPIConversationID` và `cleanupHiddenAntigravityConversation`, không chạy `git push` (`Rule #15 PASS`).
