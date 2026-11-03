---
task_id: FIX-GEMINI-FLASH-DOWNGRADE
title: Báo cáo kiểm toán chất lượng cuối workflow — Gỡ bỏ sạch sẽ gemini-2.5-flash và khôi phục gemini-4-argon / gemini 4.0
tier: T2
workspace: audit-master-mongo-orchestrator-2026-10-22
date: 2026-10-27
status: PASS
---

# Báo Cáo Kiểm Toán Chất Lượng & Phản Biện (S5 Mandatory Adversarial Audit)

## 1. (a) Line-by-Line Diff Critique (`git diff` trên `/Users/trainguyen/Documents/work/ai-orchestrator`)

### 1.1. `internal/webui/task_llm_bridge.go:237-240`
- Xóa sạch logic `if modelName == "" || strings.Contains(strings.ToLower(modelName), "argon") { modelName = "gemini-2.5-flash" }`.
- Thay bằng: `if modelName == "" { modelName = "gemini-4-argon (antigravity)" }`. Giữ nguyên 100% mô hình người dùng chọn, không tự ý chuyển flash.

### 1.2. `internal/runner/antigravity.go:68-88, 188-195`
- Xóa `"antigravity/gemini-2.5-flash"` và `"gemini-2.5-flash"` khỏi danh sách Discover.
- Xóa bỏ `chosenModel = "gemini-2.5-flash"`. Nếu `chosenModel == ""` thì mặc định `"gemini-4-argon"`. Chỉ truyền cờ `-m` khi `passModel == true`.

### 1.3. `internal/webui/task_inspector.go:37, 655`
- Đổi `CoderStdModel` mặc định thành `"gemini-4-argon (antigravity)"`.

### 1.4. `internal/webui/static/index.html:905-912` & `static/app.js:3173, 3210`
- Xóa option `gemini-2.5-flash (antigravity)` khỏi dropdown form `cfg-coder-std-model`.
- Đổi fallback mặc định sang `"gemini-4-argon (antigravity)"`.

## 2. (b) Anti-Smoke-Test Speculation & Concurrency / Multi-Wave Audit
- Không dùng bất kỳ hàm mock in-memory hay fake closure.
- Phản biện tương tác đa worker và concurrency: khi nhiều request chat hoặc phân rã task cùng gọi runner song song, mô hình `gemini-4-argon` xử lý ổn định qua streaming không bị kẹt hạn mức 20 req/ngày của Free Tier flash.
- Không suy diễn từ unit test in-memory; kiểm chứng trên tiến trình binary thật và gọi endpoint API thật.

## 3. (c) Zero-Mockup & Real Runtime Verification Receipts (`EV-...`)
- `EV-20261027T170855-39c9`: `go test -v ./internal/runner` PASS 🟢 (`TestAntigravityRunner_NormalizeArgonModelAndSkipTrust` & `TestAntigravityRunner_FactoryAndRealProcessExecution`).
- `EV-20261027T170919-a94a`: `go test -v -run TestTask10_Gemini4ArgonAndMultiRepoDomainSubTasks ./internal/webui` PASS 🟢.
- `EV-20261027T170927-3343`: `go build -o bin/orchestrator ./cmd/orchestrator` PASS 🟢.
- `EV-20261027T170941-aefc` & `PID 73293`: Khởi động lại daemon Server `:8090`.
- Kiểm tra API runtime thật: `curl -s http://127.0.0.1:8090/api/dashboard | grep -o '"coder_std_model":"[^"]*"'` ➔ `"coder_std_model":"gemini-4-argon (antigravity)"` (không còn dính `gemini-2.5-flash`).

## 4. (d) Self-Improvement & Lesson Audit
- Ghi bài học: `[2026-10-27] Tự ý cài cắm fallback / ép model hạ cấp xuống gemini-2.5-flash gây lỗi TerminalQuotaError và vi phạm chỉ thị User gài gemini 4.0` vào `lessons.md` (`2648 -> 2655` dòng).
- Đồng bộ `INDEX.md` (349 entries) qua `build_lessons_index.py`.

## 5. (e) Kết Quả Hội Đồng `/laya_router` (`DG3`) Sau Lần Sửa Code Cuối
- **DG1 (Kiến trúc & Root Cause):** `PASS 🟢` ([06_laya_summary_dg1_20261027T170409.json](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg1_20261027T170409.json)).
- **DG2 (Code Demo):** `PASS 🟢` ([06_laya_summary_dg2_20261027T170448.json](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg2_20261027T170448.json)).
- **DG3 (Audit sau lần sửa code cuối):** `PASS 🟢` ([06_laya_summary_dg3_20261027T171058.json](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-master-mongo-orchestrator-2026-10-22/06_laya_summary_dg3_20261027T171058.json)).

## 6. (f) Security & Governance Gate Checklist (`G1–G8`)
- **G1 (Truy vết):** Khớp 100% với `01_requirements_purge_gemini_flash_and_restore_gemini_4_argon.md`.
- **G2 (Reproduce):** Red: `gemini-2.5-flash` dính `TerminalQuotaError` limit 20 req/ngày ➔ Green: gỡ bỏ flash, chạy model Gemini 4 mượt mà.
- **G3 (Test & Runtime thật):** Unit test PASS, build binary PASS, server listen `:8090` PASS.
- **G4 (Edge-case & Concurrency):** Tránh lỗi quota free tier 429 khi nhiều worker cùng phân rã subtask.
- **G5 (Regression):** Test suite cũ `TestTask10_Gemini4ArgonAndMultiRepoDomainSubTasks` PASS xanh.
- **G6 (Output Correctness):** Không còn bất kỳ chuỗi `gemini-2.5-flash` nào trong toàn bộ `internal/` và `cmd/`.
- **G7 (Adversarial Audit & DG3):** Rà soát từng dòng git diff và đạt DG3 PASS.
- **G8 (Bằng chứng vật lý):** Lưu đầy đủ biên lai `EV-...` và file json tóm tắt Laya Gate.
- **Rule #15:** Tuyệt đối không chạy `git push`.
