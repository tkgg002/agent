---
tier: T2
suffix: pipeline_resilience_and_cross_model_gates
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-23T10:07:30+07:00
mode: propose-only
---

# 01. Requirements — Nâng cấp Kiến trúc Multi-Agent Pipeline Resilience & Cross-Model Gates (`A..E`) + Thống kê Thực nghiệm `Self-Healing`

## 1. Bối cảnh & Yêu cầu của User
User đánh giá trực diện kiến trúc pipeline multi-agent hiện tại (`Start -> Plan (S1–S3) -> You -> Implement (S4) -> Local QA & DG3 (S5) -> Isolated Branch (S6)`) và đề xuất **5 hạng mục nâng cấp cốt lõi (`A` đến `E`)** kèm **1 câu hỏi thực nghiệm**:

1. **A. Xóa bỏ Rủi ro "Cùng một góc mù" (`Monoculture Bias` -> `Cross-Model Auditing`):**
   - **Hiện trạng:** Tất cả `Slots A..K` (`Architect`, `Coder`, `Laya Gate`, `QA Auditor`, `Security Guard`) đều đang dùng chung một model family duy nhất là `gemini-4-argon (antigravity)`.
   - **Yêu cầu:** Đa dạng hóa model theo Slot (`Cross-Model Auditing`) để bắt chéo lỗi sắc bén hơn và tối ưu latency:
     - `MUSCLE` (`Slot B / D / E` — Coder): giữ `gemini-4-argon (antigravity)` (hoặc model thực thi chính).
     - `QA Auditor` (`Slot K` — Adversarial QA & Diff Critique): chuyển sang họ model độc lập `claude-opus-4-6 (antigravity)`.
     - `Laya Gate` (`Slot H` — Decision Gate DG1/DG2/DG3): chuyển sang `claude-sonnet-4-6 (antigravity)`.
     - `Security Guard` (`Slot I` — Secret/PII/Authz Scanner): chuyển sang `codex-5.3 (antigravity)`.
     - Hiển thị rõ huy hiệu **`Cross-Model Audit Active`** (hoặc cảnh báo `Monoculture Warning` nếu người dùng cố tình chỉnh trùng model giữa Coder và Auditor).

2. **B. Chốt chặn `Circuit Breaker (Max 3 Retries)` & Phân tách 2 tầng Vòng lặp `Self-Healing` (`S5 -> S4 -> S5` vs `S5 -> S3 -> Human Gate`):**
   - **Hiện trạng:** Vòng lặp `Self-Healing` đang cho phép lùi thẳng `S5 -> S3 -> S4 -> S5` mà không qua lại `Human Gate` (nguy cơ **Scope Drift** lệch kiến trúc đã duyệt) và không có bộ đếm `Max Retry Limit` (nguy cơ lặp vô tận / đốt token khi kẹt rule câu chữ ở `DG3`).
   - **Yêu cầu:**
     - Thêm **Max Retry Limit (`Circuit Breaker: Max 3 Retries`)** hiển thị rõ trên UI (`0/3 Self-Heal Retries · Max 3`). Nếu vượt quá `3` lần retry mà chưa pass, tự động ngắt mạch (`CIRCUIT_OPEN`) và đẩy trạng thái về `NEEDS_YOU` để con người can thiệp (`Escalation Rule #15`).
     - Phân loại rõ 2 mức độ lùi:
       - **Tier-1 Implementation Self-Heal (`S5 -> S4 -> S5`, tối đa 3 vòng):** Khi lỗi thuộc về implementation / syntax / compile / unit test -> chỉ cho phép lùi từ `S5` về `S4 (MUSCLE)` để sửa code theo đúng Plan `S3` đã duyệt.
       - **Tier-2 Architectural Re-Plan (`S5 -> S3 -> Human Gate NEEDS_YOU`):** Nếu lỗi nặng đến mức phải lùi về `S3` (thay đổi Plan/Architecture trong `09_tasks_solution.md`) -> **BẮT BUỘC** kích hoạt lại `Human Gate (NEEDS_YOU)` để User `APPROVE` lại trước khi chạy `S4`.

3. **C. Định vị lại `Worktree Provisioning (Slot C)` về trước `Implement (S4)` & Đổi tên Stage 6 thành `Finalize & Diff Review`:**
   - **Hiện trạng:** Node `Isolated Branch (Worktree Ready - Slot C)` đang đặt ở cuối sơ đồ (Stage 6) gây hiểu nhầm bước khởi tạo môi trường vật lý thành bước kết thúc.
   - **Yêu cầu:**
     - Kéo trạng thái khởi tạo môi trường **`Worktree Provisioned (Slot C)`** về đứng ngay trước/tại bước **`Implement (S4)`** (và tích hợp badge `Worktree Attached: task/<id>` ngay trên header của Task/Sub-Task).
     - Đổi tên và vai trò Stage 6 cuối luồng thành **`Finalize & Diff Review (Ready for Merge)`** (đóng gói deliverable, đối soát `git diff` cuối cùng tại worktree và sẵn sàng Merge/PR dưới chốt chặn `No-Push H5`).

4. **D. Cơ chế `Contract/Integration Sync Gate` & `Atomic Wave Rollback` cho `Multi-Repo Wave`:**
   - **Hiện trạng:** Khi chạy `Multi-Repo Wave (Slot E / F)` gồm nhiều Sub-Tasks phụ thuộc nhau, nếu Sub-Task 1 thành công nhưng Sub-Task 2 thất bại không thể self-heal ở `DG3`, trạng thái giữa các worktree bị lệch.
   - **Yêu cầu:**
     - Bổ sung **`Contract/Integration Sync Gate`** nhẹ giữa các Sub-Tasks trong cùng một Wave (đối chiếu `OutputContract.ExportedSymbols` của Sub-Task trước với `InputContract.RequiredFrom` của Sub-Task sau).
     - Bổ sung cơ chế **`Atomic Wave Rollback`**: lưu `WaveRestorePointSHA` cho toàn bộ worktree trong Wave; nếu bất kỳ Sub-Task nào trong Wave chạm ngưỡng Circuit Breaker (`3/3 Retries Failed`), tự động rollback đồng loạt các worktree trong Wave đó về `WaveRestorePointSHA` ban đầu để bảo toàn tính nguyên tử (`Atomicity`).

5. **E. Tách bạch Stage 5 thành 2 tầng: `5A Hard Deterministic Gate (Zero-LLM)` đứng chắn trước `5B Semantic Audit Gate (Cross-Model QA + DG3 + G1–G8)`:**
   - **Hiện trạng:** Khối `Local QA & DG3` đang gộp chung kiểm tra kỹ thuật với `QA Auditor (DG3)`, gây lãng phí gọi LLM nếu code chưa compile được hoặc rớt unit test cơ bản.
   - **Yêu cầu:** Tách rõ 2 tầng tuần tự trong Stage 5:
     - **Tầng 5A — `Hard Deterministic Gate (Zero-LLM)`:** Chạy thuần script hệ thống (`go vet ./...`, `golangci-lint`, `go test -race`, `go build`). Nếu fail -> trả thẳng `stdout/stderr` về `MUSCLE (S4)` (`Tier-1 Self-Heal S5A -> S4`), tốn **0 lần gọi LLM `QA Auditor`**.
     - **Tầng 5B — `Semantic Audit Gate (Cross-Model QA + DG3 + G1–G8 + Security Guard)`:** Chỉ khi `5A Hard Gate` xanh `100%` (`exit=0`) mới gọi `QA Auditor (claude-opus-4-6)`, `Laya Gate (claude-sonnet-4-6)` và `Security Guard (codex-5.3)` vào đối chiếu ngữ nghĩa với bản Plan `S1–S3`.

6. **F. Câu hỏi Thực nghiệm Trực diện từ User:**
   - *"Trong thực tế lúc bạn cho chạy các task trước đây, bước `Self-Healing (S5->S3->S4->S5)` hay bị kẹt ở lỗi code thực sự (compile/test fail) hay bị kẹt do rule đối chiếu câu chữ của cổng `DG3` nhiều hơn?"*
   - **Yêu cầu:** Trả lời trung thực bằng số liệu thống kê thực tế từ toàn bộ `79` biên lai `06_evidence/EV-*.json` và `24` kết quả `06_laya_raw_*.json` trong workspace hiện tại (không suy diễn).

## 2. Chế độ Quản trị (Governance Mode)
- **Tier:** `T2`
- **Trạng thái hiện tại:** `Propose-Only (S1 -> S3)` theo `Rule #13`. **TUYỆT ĐỐI CẤM** sửa bất kỳ file source code nào (`.go`, `.js`, `.html`, `.css`) trước khi User phản hồi `APPROVE`.
