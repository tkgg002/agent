---
tier: T2
suffix: evaluate_ecc_and_ai_orchestrator
---

# 13 — Analysis: Mổ xẻ Mã nguồn `affaan-m/ecc` (`v2.2.3`) & Đối sánh Kiến trúc với `ai-orchestrator` + `/agent` Core

## 1. Bằng chứng Khảo sát Mã nguồn Thật (`VERIFIED` — `LS-20261020-readme-only` & `LS-20261021-prior-workspace-lookup`)
- **Lịch sử Workspace (`upgrade-agent-infrastructure/00_context.md:4`):** Trước đây hệ thống đã từng khảo sát bản `everything-claude-code` `v1.10.0` (chỉ là bộ sưu tập Markdown rules/skills/commands cho Claude Code).
- **Bản hiện tại (`affaan-m/ecc` `v2.2.3` — `package.json:1-4`, `VERSION:1`):** `ecc` đã tiến hóa từ một bộ plugin Markdown (`v1.10.0`) thành **`ecc-universal` (`v2.2.3`)** gồm 2 phần tách biệt:
  1. **Tầng Harness Plugin đa nền tảng (Node.js + Markdown):**
     - `71` specialized agents (`AGENTS.md:3, 17-88`), `311` skills (`skills/`), `97` commands (`commands/`), `25` rules (`rules/`), hỗ trợ đồng thời `.claude-plugin/`, `.codex/`, `.cursor/`, `.gemini/`, `.opencode/` (`package.json:54-71`).
     - **GateGuard Fact-Forcing Hook (`scripts/hooks/gateguard-fact-force.js:1-140`, `2623` dòng):** Thay vì hỏi *"Are you sure?"*, hook `PreToolUse` bắt buộc Agent phải chứng minh đã chạy lệnh đọc/tìm kiếm (`turnScan().searchesMayNameTarget(scan, filePath)` tại dòng `66-73`) và trả lời các câu hỏi thực tế (importers, public API, schema) trước lần sửa đầu tiên (`first-touch Edit/Write`) hoặc trước lệnh Bash/SQL nguy hiểm (`DESTRUCTIVE_SQL` tại dòng `126`).
     - **Plan Canvas (`scripts/plan-canvas.js:1-120`, `406` dòng):** Mở một HTTP server cục bộ tách rời để hiển thị file `.plan.md` trên trình duyệt và dùng CLI `node scripts/plan-canvas.js await <file>` long-polling chờ Human duyệt.
     - **ECC Memory Vault (`docs/design/ecc-memory-vault.md:1-140` & `scripts/memory-mcp.mjs`):** Lưu trữ tri thức dạng Markdown có frontmatter `schema: "ecc.memory.v1"` (`trust: "unreviewed"`, create-only không ghi đè tại dòng `24-25`) qua MCP stdio server.
     - **Harness Audit (`scripts/harness-audit.js:12-25`):** Chấm điểm tĩnh repo trên 12 hạng mục (`Tool Coverage`, `Context Efficiency`, `Quality Gates`, `Memory Persistence`, `Eval Coverage`, `Security Guardrails`...).
  2. **Tầng Control Plane `ECC 2.0` (`ecc2/` — Rust TUI + SQLite `rusqlite`):**
     - Viết bằng Rust (`ecc2/Cargo.toml:1-61`, `ratatui 0.30` + `tokio` + `rusqlite 0.40` + `git2 0.21`), nhưng mắc bệnh **Monolithic God Files nghiêm trọng**:
       - `ecc2/src/main.rs`: **`12,830` dòng** trong 1 file duy nhất!
       - `ecc2/src/tui/dashboard.rs`: **`15,460` dòng** trong 1 file duy nhất!
       - `ecc2/src/session/manager.rs`: **`8,181` dòng** trong 1 file duy nhất!
       - `ecc2/src/worktree/mod.rs`: **`2,688` dòng**!
     - Quản lý phiên CLI (`ecc2/src/session/runtime.rs:36-135` dùng `DbWriter` channel ghi stdout/stderr của tiến trình con vào SQLite), tạo Git Worktree (`ecc2/src/worktree/mod.rs:96-120` + kiểm tra xung đột merge giữa 2 nhánh `BranchConflictPreview` tại dòng `47-53`), giao tiếp liên phiên (`ecc2/src/comms/mod.rs:30-50` với `MessageType::TaskHandoff`, `Query`, `Response`, `Completed`, `Conflict`), và đánh giá cấu hình harness bằng content-addressable hash (`ecc2/src/harness_eval.rs:8-98`).

---

## 2. Bảng Đối Sánh Trực Diện: `affaan-m/ecc` (`v2.2.3`) vs `ai-orchestrator` + `/agent` Core

| Tiêu chí Kiến trúc | `affaan-m/ecc` (`v2.2.3` + `ecc2` Rust) | `ai-orchestrator` (Go) + `/agent` Core | Đánh giá Thắng / Thua |
| :--- | :--- | :--- | :--- |
| **1. Bản chất Hệ thống** | **Bộ sưu tập Prompt/Skill khổng lồ (`311` skills, `71` agents) + Session Launcher TUI (`ecc2`)**: Chạy từng phiên CLI rời rạc, nghiêng về chiều rộng (breadth) đa ngôn ngữ/framework. | **Autonomous SDLC Workflow Engine (`S0 -> S6`) + Sovereign Governance Kernel (`/agent`)**: Điều phối khép kín từ Intake (`S1`), Multi-Repo Scan (`S2`), Pre-Plan Critique (`DG0`), Plan (`S3`), 6-Wave Sub-Task Execution (`S4`), Dual-Layer 5A/5B QA (`S5`) tới `S6`. | **`ai-orchestrator` + `/agent` vượt trội về chiều sâu SDLC & kiểm soát trạng thái**, trong khi `ecc` mạnh về kho template/skill đa ngôn ngữ. |
| **2. Cổng Kiểm định Quyết định (Decision Gate)** | Dựa trên regex/heuristic (`gateguard-fact-force.js:66-126`) hoặc gọi lại chính LLM (`agent-evaluator`). Không có Neural Verifier độc lập. | **Local Neural Decision Engine `/laya_router` (`convaiinnovations/laya` `:8200`, 0 LLM token, ~33ms/pass)** kèm Calibration (`control_true`/`control_false`) và Corrupted-Diff Debate tại `DG0`, `DG1`, `DG2`, `DG3` (`internal/agentgate/laya_runner.go`). | **`ai-orchestrator` + `/agent` thắng tuyệt đối**: Chặn ảo giác bằng mô hình neural độc lập 0 tốn token kèm câu hỏi bẫy định chuẩn. |
| **3. Cơ chế Fact-Forcing trước khi sửa file (First-Touch Read/Search Proof)** | **`gateguard-fact-force.js:45-73` rất hay**: Quét lịch sử tool call trong lượt (`turnScan`) để kiểm tra xem Agent đã thực sự `read`/`grep` file mục tiêu chưa trước khi cho phép `Edit/Write` lần đầu. | `agy_guard.py` (`H4`/`H4b`) chặn sửa code khi chưa có S1/S3/`APPROVE`, và kiểm tra `NOW <= HEAD` khi sửa file lớn, nhưng trong `ai-orchestrator` (Muscle S4) đã nạp thẳng snippet thật của file mục tiêu vào prompt (`readRealSubTaskTouchpointSnippet`). | **Điểm đáng học hỏi từ `ecc`**: Cơ chế kiểm tra `Overlap / Conflict Detection` giữa các phiên đang sửa cùng 1 file (`ecc2/src/comms/mod.rs:49`, `FileActivityOverlap` tại `dashboard.rs:23`). |
| **4. Kiến trúc Quản lý Bộ nhớ & Bài học (Memory Architecture)** | `.ecc/memory/` (`ecc.memory.v1` Markdown + MCP stdio `scripts/memory-mcp.mjs`): Mọi bài học ghi vào ở trạng thái `trust: "unreviewed"`, không tự động biến thành ràng buộc cưỡng chế ở S0 Boot. | **2-Tier Memory Map (`lessons.md` -> `lessons/INDEX.md` + `topics/01..05.md`) + `tripwires.json`**: Tự động quét keyword tại Hook `H1` (`S0 Boot`) và bơm `MatchedLessons` + `MatchedTripwires` thẳng vào `01_requirements.md` của từng Task (`internal/agentgate/kernel_bridge.go`). | **`ai-orchestrator` + `/agent` vượt trội**: Biến bài học xương máu thành **Tripwire cưỡng chế tự động** ngay từ bước S0/S1 thay vì chỉ lưu trữ thụ động. |
| **5. Điều phối Sub-Tasks, Multi-Repo & Self-Healing** | `ecc2` chỉ có `Delegate` (`ecc2/src/main.rs:136-151`) từ session A sang session B và `Template`; không có DAG Wave 0..4 tự động kết hợp Circuit Breaker 2 tầng. | **12-Step / 6-Wave Fractal Sub-Task Engine (`internal/webui/subtask_wave_engine.go:16-430`)**: Có `InputContract`/`OutputContract`, `5A Hard Gate (Zero-LLM)` chạy trước `5B Cross-Model Semantic QA`, Circuit Breaker 3 lần thử + `AtomicRollbackWaveSubTasks`. | **`ai-orchestrator` thắng tuyệt đối** về kiến trúc thực thi tự hành nhiều bước và tự chữa lành có kiểm soát. |
| **6. Giao diện Điều khiển & Tương tác Con người (Control Plane UX)** | `ecc` bị phân mảnh thành 3 giao diện rời rạc: (1) `ecc_dashboard.py` bằng Tkinter cổ điển (`ecc_dashboard.py:10`), (2) `scripts/plan-canvas.js` chỉ xem 1 file markdown, (3) `ecc2` TUI terminal (`ratatui`). Không có Telegram Forum Topic Bridge. | **Hợp nhất WebUI Co-Architect Studio (`:8090`, SSE real-time, Kanban, Wave Matrix, Edit Task/Sub-Task Modal, Diff Inspector) + Telegram Forum Topic Bridge 2 chiều (`internal/webui/telegram_bridge.go`)**. | **`ai-orchestrator` vượt trội hoàn toàn** về trải nghiệm điều hành tập trung Web + Mobile (Telegram). |
| **7. Chất lượng Tổ chức Code (Code Modularity & Hygiene)** | Vi phạm chính nguyên tắc `<800 lines` mà `AGENTS.md:135,142` của `ecc` tự đề ra: `ecc2/src/tui/dashboard.rs` dài **`15,460` dòng**, `ecc2/src/main.rs` dài **`12,830` dòng**, `ecc2/src/session/manager.rs` dài **`8,181` dòng**; `hooks/hooks.json` lặp lại chuỗi one-liner Node.js dài 1,800 ký tự tới 15 lần! | Đang áp dụng kỷ luật **Single-Responsibility & `NOW <= HEAD`** nghiêm ngặt, kèm kiểm chứng biên lai vật lý `EV-...` (`evidence_run.py`) khớp `src_fingerprint`. | **`ai-orchestrator` + `/agent` kỷ luật hơn hẳn**: Nói đi đôi với làm, cưỡng chế bằng linter `verify_governance.py` và hook `agy_guard.py`. |

---

## 3. Ba Tinh Hoa Đáng Chắt Lọc Nhất Từ `affaan-m/ecc` Để Nâng Cấp Cho `ai-orchestrator` (Nếu User Muốn Tích Hợp)
1. **Cross-Task / Cross-SubTask File Overlap & Conflict Radar (`FileActivityOverlap` & `BranchConflictPreview` — tham chiếu `ecc2/src/worktree/mod.rs:47-53` & `ecc2/src/comms/mod.rs:49`):**
   - Phát hiện sớm khi 2 Sub-Tasks thuộc cùng 1 Wave song song (hoặc 2 Tasks đang chạy đồng thời) chạm vào cùng một file (`TargetFiles` / `ContextFileLine` trùng nhau), tự động cảnh báo xung đột hoặc đẩy Sub-Task thứ 2 sang Wave kế tiếp trước khi chạy `S4`.
2. **Pre-S4 First-Touch Fact-Forcing Verification (`GateGuard` concept — tham chiếu `scripts/hooks/gateguard-fact-force.js:45-73`):**
   - Trước khi `S4 Muscle` sửa một file trong `TargetRepo`, bắt buộc trích xuất danh sách `importers / callers` thật của file đó trong repo để truyền vào context sinh code, tránh sửa gãy signature của caller.
3. **Deterministic Content-Addressable Evaluation Digest (`CandidateSpec` — tham chiếu `ecc2/src/harness_eval.rs:8-95`):**
   - Gắn mã băm SHA-256 chuẩn hóa (`canonical_config + trace_refs + evidence_refs`) vào biên lai kiểm định Sub-Task (`EV-SUB-...`) để phát hiện tức thì nếu artifact hoặc diff bị sửa đổi sau khi qua cổng QA.
