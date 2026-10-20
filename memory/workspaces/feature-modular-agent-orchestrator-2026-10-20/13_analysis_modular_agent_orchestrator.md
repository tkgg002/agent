# 13 — Architectural Audit & Comparative Evaluation: Bộ `/agent` (`S0 → S6`) vs. `Event/Queue-Driven Modular Multi-Agent System`

> **Chế độ:** `Audit Propose-Only` (Rule #1, Rule #13 — Chỉ đọc, đối sánh thực chứng kèm `REF: file:line`, không tự ý sửa source code).
> **Workspace:** `feature-modular-agent-orchestrator-2026-10-20` | **Suffix:** `modular_agent_orchestrator`

---

## 1. Tóm tắt Nhận định Cốt lõi (Executive Verdict)

Khi đặt **bộ `/agent` kèm Master Workflow `S0 → S6` hiện tại của anh** lên bàn cân với **Event/Queue-Driven Modular Multi-Agent System** (`Scout → Task Queue → Sorting → Priority Queue → Planner [Opus + Wave DAG] → Coder [Dynamic Opus/Sonnet] → Reviewer/QA [Autopilot Loop]`), kết luận kỹ thuật thẳng thắn và chính xác nhất là:

1. **Bộ `/agent` hiện tại của anh là một "Hệ thống Quản trị Chất lượng & Kỷ luật Thực thi Cấp Hạt nhân" (Inner Execution & Quality Kernel) cực kỳ xuất sắc — vượt xa các Orchestrator mã nguồn mở thông thường về độ tin cậy (Trust, Anti-Hallucination, Anti-Mockup, Clean-Room & Mathematical Gate).**
2. **Tuy nhiên, bộ `/agent` hiện đang vận hành theo mô hình "Single-Session Interactive Pull" (Đơn luồng - Tương tác đồng bộ theo từng phiên chat), nên thiếu "Tầng Điều phối Dây chuyền Bất đồng bộ" (Outer Event/Queue-Driven Control Plane).**
3. **Hai kiến trúc này KHÔNG thay thế nhau mà là 2 nửa ghép lại thành một hệ thống hoàn chỉnh:**
   - **Event/Queue-Driven System** đóng vai trò **"Nhà máy & Băng chuyền Tự động" (Throughput, Triage, Parallelism, Cost Routing)**.
   - **Bộ `/agent` (`GEMINI.md` + `agy_guard.py` + `laya_gate.py` + `mutation_gate.py`)** đóng vai trò **"Luật Pháp, Khuôn Mẫu & Trạm Kiểm Định Chất Lượng" (Quality & Governance Kernel)** chạy bên trong từng Worker trên băng chuyền đó.

---

## 2. Bảng Đối Sánh Trực Diện 10 Trục Kiến Trúc (`VERIFIED`)

| Trục đánh giá | Bộ `/agent` + Workflow `S0 → S6` hiện tại | `Event/Queue-Driven Modular Multi-Agent System` | Đánh giá Thắng/Thua & Hướng Kết Hợp |
| :--- | :--- | :--- | :--- |
| **1. Cơ chế kích hoạt công việc (Task Ingestion)** | **Thủ công (Human-Triggered):** Anh phải tự đọc Jira/GitHub/Log rồi gõ lệnh mở phiên chat (`GEMINI.md:68-69`, `agy_guard.py:130-134`). | **Tự động 100% (`Scout Agent`):** Script Non-LLM định kỳ quét Jira API, GitHub Issues đẩy vào `Task Queue` (`0 LLM token`). | **Queue-Driven thắng về tự động hóa đầu vào.** Cần bổ sung `Scout Worker` phía trước bước `S1 Intake`. |
| **2. Xếp hạng ưu tiên (Triage & Prioritization)** | **Không có hàng đợi:** Anh tự chọn làm việc nào trước bằng cách mở tab chat tương ứng. | **Tự động (`Sorting Agent`):** Bốc từ `Task Queue`, chấm điểm mức độ khẩn cấp và xếp vào `Priority Task Queue`. | **Queue-Driven thắng về quản lý Backlog.** Giúp hệ thống luôn xử lý ticket `Blocker/Critical` trước. |
| **3. Phân rã công việc & Chạy song song (Task Decomposition)** | **Danh sách phẳng (`08_tasks_*.md`):** Liệt kê checklist Markdown (`brain-delegate/SKILL.md:69-74`), thực thi tuần tự trong 1 workspace. | **Đồ thị phụ thuộc theo đợt (`Wave 0..N` DAG — Ảnh 1):** Phân biệt rõ `READY`, `BLOCKED`, `NEEDS YOU`, `DONE` và nhánh `critical`; mở khóa tự động khi xong Wave trước. | **Queue-Driven thắng về khả năng chạy song song.** Cho phép nhiều Coder (`A..K`) bốc các node `READY` cùng lúc trên các `git worktree` riêng. |
| **4. Định tuyến Model & Tối ưu Chi phí (Model & Cost Routing)** | **Chuyển đổi thủ công trên UI IDE:** Anh phải tự chọn model trên thanh cài đặt; chưa tự động ép model theo độ khó của từng sub-task. | **Định tuyến động (`Dynamic Model Routing`):** Cố định `Opus 5.5` cho `Planner`; tự động chọn `Opus` (task khó) hoặc `Sonnet` (task vừa/dễ) cho `Coder`; `Scout` dùng Script `$0`. | **Queue-Driven thắng về tối ưu chi phí (`$`) & hiệu năng.** |
| **5. Cô lập Vai trò & Phòng sạch (Separation of Concerns)** | **Đã có luật phân vai (`GEMINI.md:18-21`) & Clean-Room (`GEMINI.md:95-97`, `agy_guard.py:456-502`):** Chặn cứng Brain sửa code trong `mode: clean-room` và buộc gọi `invoke_subagent`. Tuy nhiên ở task `T1/T2` thường, 1 phiên chat vẫn kiêm nhiệm cả Brain → Muscle → QA sau khi `APPROVE`. | **Cô lập tuyệt đối ở cấp Tiến trình/Hàng đợi:** `Planner`, `Coder`, `Reviewer` là các Worker độc lập chỉ giao tiếp qua Queue & Artifact, không chia sẻ chung lịch sử chat. | **Kết hợp cả hai là mạnh nhất:** Dùng Queue Worker độc lập cho mọi task + giữ luật `Clean-Room Spec` của `/agent` khi refactor module cũ. |
| **6. Cưỡng chế Kỷ luật Thực thi (Runtime Governance Enforcement)** | **Cực mạnh (`hooks/agy/agy_guard.py:7-21`):** 7 Lifecycle Hooks (`H1–H7`) chặn trực tiếp ở tầng Tool Call: cấm xóa/ghi đè memory (`H2`), cấm lộ secret (`H3`), cấm sửa code khi thiếu `S1`/`DG1`/`APPROVE` hoặc gắn mock `*Fn` vào code thật (`H4`), cấm `git push` (`H5`), ép chạy checks (`H6`) & chặn dừng khi thiếu `audit_report` (`H7`). | **Mềm / Phụ thuộc Orchestrator Loop:** Các hệ thống như `Open-Makers` chủ yếu nhắc trong prompt (`AGENTS.md`) và chạy lại `build/test`, chưa có hook chặn tool-call chống gắn mock láo hay bảo vệ file append-only. | **Bộ `/agent` của anh THẮNG TUYỆT ĐỐI.** Phải giữ nguyên `agy_guard.py` + `verify_governance.py` làm hàng rào bảo vệ cho mọi Worker. |
| **7. Chất lượng Kiểm thử & Chống "Test Phế" (Test Verification)** | **Cực mạnh (`GEMINI.md:76, 99-100`, `tooling/mutation_gate.py:1-12`):** Cấm suy diễn từ Smoke Test / Mockup; bắt buộc phản biện Concurrency/Timeout/Watermark và chạy **3-Mutant Gate (`mutation_gate.py`)** đảm bảo $\ge 3$ bản làm hỏng code đều bị test bắt sống (`KILLED`). | **TDD cơ bản (`Phase A -> Phase C`):** QA viết test trước, Coder viết code cho pass test. Nhưng nếu cả QA và Coder đều là LLM thì dễ sinh ra "test hình thức" (test pass nhưng không bắt được bug thật). | **Bộ `/agent` của anh THẮNG TUYỆT ĐỐI.** Đưa `mutation_gate.py` và Anti-Smoke-Speculation vào `Local QA` của Autopilot sẽ bịt kín lỗ hổng TDD giả tạo. |
| **8. Hội đồng Phán quyết & Trọng tài (Decision & Review Gate)** | **Thẩm định Toán học Độc lập (`/laya_router` — `tooling/laya_gate.py:16-24`):** Có 3 cổng `DG1/DG2/DG3`, ép trích dẫn `REF: path:start-end` khớp 100% file thật trên đĩa + cài bẫy đối chứng (`control_false`, `control_true`, `corrupted-diff`). | **PM LLM Arbitration + Vòng lặp `AUTOPILOT` (Ảnh 2):** Chạy `Plan review`, `Code review`, `Local QA` tối đa 3 rounds (`blocking` vs `advisory`), đo đếm rõ `turns`, `minutes`, `$ cost`. | **Bù khuyết hoàn hảo:** Dùng giao diện vòng lặp `Round 1..3` + sổ cái `$ cost / turns` của Autopilot, nhưng phán quyết mỗi Round phải đi qua `laya_gate.py` (`DG1/DG2/DG3`). |
| **9. Bộ nhớ Dài hạn & Tự học (Memory & Self-Improvement)** | **2-Tier Lessons Map (`lessons/INDEX.md` + `build_lessons_index.py` — 249 bài học chuẩn 5 phần A/B/X/Y):** Rất mạnh về ghi nhớ "kinh nghiệm xương máu / lỗi từng vấp", nhưng việc tra cứu hồ sơ giữa >70 workspaces cũ chưa có full-text search tự động. | **OpenClaw-style Memory (`MEMORY.md` + `SQLite FTS5 BM25` + Hybrid Embedding):** Tự động băm nhỏ (chunk 400 tokens) và tìm kiếm Top-K đoạn kiến trúc/task cũ liên quan bơm vào prompt. | **Kết hợp cả hai:** Giữ `lessons/INDEX.md` cho Luật & Bài học chống lặp lỗi + thêm `SQLite FTS5 BM25` để tự động recall kiến trúc từ `memory/workspaces/`. |
| **10. Khả năng Quan sát & Điều khiển Đa luồng (Observability UI)** | **Giao diện Chat đơn lẻ + File Markdown trong Workspace:** Khó nhìn tổng thể khi có 100+ sub-tasks hoặc nhiều dự án chạy cùng lúc. | **3 Màn hình Chuyên dụng (Ảnh 1, 2, 3):** Bản đồ `Wave 0..N DAG`, Bảng `AUTOPILOT` (`turns / min / $`), và Lưới `Multi-Terminal Agent Grid` (`Workers A..K`). | **Queue-Driven System thắng tuyệt đối về UX/Observability.** |

---

## 3. Phân tích Chi tiết 6 Khoảng trống Kiến trúc (`F1–F6`) của Bộ `/agent` Hiện tại (`VERIFIED`)

### `F1 [P1 - High]` — Mô hình Thực thi Đang Bị Khóa Vào "Single-Session Interactive Pull"
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py:130-134` & `GEMINI.md:68-69`):**
  - Toàn bộ trạng thái điều phối trong `agy_guard.py` dựa trên `last_user_idx(st)` (tìm lượt `USER_INPUT` cuối cùng trong `transcript.jsonl` của phiên chat hiện tại).
- **Hệ quả:**
  - Anh phải trực tiếp ngồi điều phối từng phiên chat: giao việc → chờ Brain lập `09_tasks_solution` → đọc và gõ `APPROVE` → chờ Muscle code và chạy audit. Không thể nạp sẵn 20 tickets để hệ thống tự động xếp hàng, tự lập plan gối đầu nhau sẵn sàng chờ anh duyệt.

### `F2 [P1 - High]` — Thiếu Tầng `Non-LLM Scout` & `Priority Sorting` Ở Đầu Nguồn
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/GEMINI.md:68-71`):**
  - Quy trình bắt đầu ngay tại `S0 BOOT` và `S1 INTAKE` khi đã có sẵn yêu cầu từ người dùng. Chưa có module worker chạy nền (0 tốn LLM token) để tự động hút ticket từ Jira/GitHub về hàng đợi và phân loại mức độ ưu tiên (`Priority Task Queue`).

### `F3 [P1 - High]` — Hồ sơ Task (`08_tasks_*.md`) Là Văn Bản Tĩnh, Chưa Có Engine Thực Thi Đồ Thị `Wave 0..N` DAG
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/skills/brain-delegate/SKILL.md:69-74` & `tooling/verify_governance.py:25-39`):**
  - `08_tasks_<suffix>.md` hiện đóng vai trò tài liệu đặc tả cho con người và Agent đọc, còn `verify_governance.py` chỉ kiểm tra sự tồn tại của file theo glob/suffix.
- **Hệ quả:**
  - Khi gặp một Epic lớn (ví dụ 50–176 items như Ảnh 1), hệ thống chưa tự động tính toán được task nào đang `READY` (không bị phụ thuộc), task nào đang `BLOCKED` (chờ bảng DB hoặc API ở Wave trước), và chưa thể tự động mở khóa (`unblock`) Wave tiếp theo ngay khi Wave trước vừa `DONE`.

### `F4 [P1 - High]` — Chưa Tự Động Hóa `Dynamic Model Routing` & Cô Lập Context Mặc Định Cho Mọi Task
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/GEMINI.md:18-21` & `hooks/agy/agy_guard.py:456-502`):**
  - Chỉ khi khai báo `mode: clean-room` kèm `clean_room_forbidden_sources` thì Hook `H4b` mới cưỡng chế tách sub-agent độc lập (`invoke_subagent`). Với các task `T1/T2` thông thường, một model duy nhất trong cùng phiên chat đang thực hiện cả `S1–S3` (Brain) lẫn `S4–S6` (Muscle/QA).
- **Hệ quả:**
  - Chưa tận dụng được nguyên lý **"Planner luôn dùng Opus 5.5, còn Coder tùy độ khó của từng sub-task trong Plan mà tự động bật worker Opus hay Sonnet"** để vừa tiết kiệm chi phí vừa giữ context của từng worker luôn sạch và ngắn gọn.

### `F5 [P2 - Medium]` — Thiếu Sổ Cái Định Lượng `Autopilot Rounds & Cost/Turn Ledger`
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/tooling/verify_governance.py:213-241`):**
  - Nhật ký `05_progress_*.md` ghi nhận chuỗi sự kiện theo thời gian (`- [Timestamp] [Agent:Model] Action`), rất tốt cho kiểm toán quy trình nhưng chưa lưu cấu trúc định lượng của từng vòng lặp (`Round 1..3`: số `turns`, thời gian chạy `min`, số lỗi `blocking/advisory`, và chi phí `$ USD` như Ảnh 2).

### `F6 [P2 - Medium]` — Kho Hồ Sơ `memory/workspaces/` (>70 Workspaces) Chưa Có Chỉ Mục Tìm Kiếm Tự Động (`SQLite FTS5 BM25`)
- **Bằng chứng (`REF: /Users/trainguyen/Documents/work/agent/OPERATOR_MAP.md:26-27`):**
  - Hiện tại mới có `tooling/build_lessons_index.py` lập chỉ mục cho `lessons.md` (`249` bài học). Còn tri thức kiến trúc nằm rải rác trong hơn 70 thư mục `memory/workspaces/<ws>/` chưa được đánh chỉ mục vào `SQLite FTS5` để tự động recall vào đầu bước `S1/S2`.

---

## 4. Kiến Trúc Hợp Nhất Tối Ưu Nhất: Đưa `/agent` Làm "Hạt Nhân Kỷ Luật" Của `Queue-Driven Multi-Agent System`

Thay vì chọn 1 trong 2, phương án tối ưu duy nhất là **bọc tầng `Event/Queue-Driven Orchestrator` ra bên ngoài bộ `/agent` hiện có**:

```mermaid
flowchart TB
    subgraph Outer["TẦNG NGOÀI: Event/Queue-Driven Orchestrator (Throughput & Routing)"]
        direction LR
        S["1. Scout Worker (Non-LLM Script - $0)"] --> TQ[("Task Queue")]
        TQ --> SO["2. Sorting Worker (Priority Scorer)"]
        SO --> PTQ[("Priority Task Queue")]
        PTQ --> P["3. Planner Worker (Khóa cứng Opus 5.5)"]
        P --> DAG[("Wave 0..N DAG Queue (READY / BLOCKED)")]
        DAG --> C["4. Coder Workers A..K (Dynamic Opus/Sonnet + Git Worktree)"]
        C --> RQ[("Review & QA Queue")]
        RQ --> R["5. Reviewer & Local QA Worker (Autopilot Rounds + Cost $)"]
    end

    subgraph Inner["TẦNG LÕI: Bộ /agent Hiện Tại (Governance & Quality Kernel)"]
        direction TB
        K1["S0 Boot: GEMINI.md + 2-Tier Lessons (INDEX.md)"]
        K2["S1-S3 + DG1/DG2: Workspace Docs + laya_gate.py (Calibrated Claims)"]
        K3["H1-H7 Hooks (agy_guard.py): Chặn Mock *Fn, Chặn Overwrite, Chặn Git Push"]
        K4["S4 Clean-Room + 3-Mutant Gate (mutation_gate.py)"]
        K5["S5-S6: Adversarial Line-by-Line Audit + DG3 + verify_governance.py"]
    end

    P -. "Thực thi S1-S3 & DG1/DG2" .-> K1 & K2
    C -. "Chịu sự giám sát của H2-H6 & Clean-Room" .-> K3 & K4
    R -. "Thực thi S5-S6, Mutation Gate & DG3" .-> K4 & K5
```

### Tóm lại:
- **Nếu chỉ dùng `Event/Queue-Driven Multi-Agent` mà bỏ `/agent`:** Anh có một dây chuyền tự động hóa rất nhanh và đẹp mắt, nhưng các Agent sẽ tự biên tự diễn unit test giả (`mockup`), tự chấm pass cho nhau và đẩy hàng loạt PR lỗi lên repo.
- **Nếu giữ nguyên `/agent` làm lõi và lắp thêm `Event/Queue-Driven Orchestrator` ở tầng ngoài:** Anh vừa có **dây chuyền tự động từ Jira/GitHub → Priority Queue → Wave DAG → Multi-Coder → Draft PR**, vừa giữ được **100% kỷ luật thép (`H1–H7`, `/laya_router` `DG1–DG3`, `Clean-Room`, `3-Mutant Gate`, `Anti-Smoke-Speculation`)** đã được tôi luyện qua 249 bài học thực chiến.

---

## 5. Bảng Kết quả Kiểm định Decision Gate `DG3`

Toàn bộ các nhận định và bằng chứng `F1–F6` trong báo cáo này được kiểm định qua `tooling/laya_gate.py run --stage dg3` (file claims: `06_laya_claims_dg3_eval_modular_agent_orchestrator.json`, file summary: `06_laya_summary_dg3_20261020T133341.json`, `gate: PASS`, `trusted_rounds: ["r2", "r3"]`):

| ID | Category | Verdict | R1 | R2 (Trusted) | R3 (Trusted) | Ground Truth REF |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `A-F1` | `real` (Interactive Single-Session vs Queue-Driven) | `ACCEPT` | `0.8588` | `0.9037` | `0.7852` | `hooks/agy/agy_guard.py:130-134` |
| `A-F2` | `real` (Static Markdown Checklist vs Wave 0..N DAG) | `ACCEPT` | `0.9462` | `0.7429` | `0.8229` | `tooling/verify_governance.py:25-39` |
| `A-F3` | `real` (Single Session Model vs Dynamic Model Routing) | `ACCEPT` | `0.8691` | `0.8407` | `0.8313` | `hooks/agy/agy_guard.py:456-502` |
| `C-FALSE-1` | `control_false` (Bẫy sai: laya_gate không kiểm tra REF) | `REJECT` | `0.1328` | `0.0956` | `0.0680` | `tooling/laya_gate.py:92-128` |
| `C-FALSE-2` | `control_false` (Bẫy sai: H5 cho phép git push) | `REJECT` | `0.3693` | `0.0885` | `0.0642` | `hooks/agy/agy_guard.py:553-563` |
| `C-TRUE-1` | `control_true` (Mốc đúng: H1–H7 hooks trong agy_guard.py) | `ACCEPT` | `0.8705` | `0.8505` | `0.8398` | `hooks/agy/agy_guard.py:7-21` |

