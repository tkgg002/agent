---
tier: T2
suffix: cdc_recon_01_subtask_execution_and_s4_s5_unblock
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-26T11:22:00+07:00
status: S1_S3_PROPOSE_ONLY
---

# 01. Requirements — Xử Lý Lỗi Kẹt Vòng Lặp `S5 -> S3 Re-Plan` (`0/6 Sub-Tasks`) & Mở Khóa Thực Thi Toàn Trình `CDC-RECON-01`

## 1. Bối Cảnh & Yêu Cầu Của User
User chạy task **`CDC-RECON-01` ("Audit vì sao khi chạy recon shadow -> master thì kết quả không đúng")** trên giao diện Orchestrator (`:8090`, phạm vi 3 repos: `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`; 6 Sub-Tasks: `TASK-AUDI01 .. TASK-AUDI06`) và phản ánh:
1. Tiến trình bị lặp 3 lần liên tiếp ở `🛡️ S5 Audit Hook -> 🧠 S3 Re-Plan` (`11:09:42`, `11:09:58`, `11:10:13`) với kết quả:
   - `Kết quả Đối soát Vật lý (VerifySubTaskCodeOnDisk): Đạt 0/6 Sub-Tasks (3 blocking)`
2. Sau 3 lần retry, Circuit Breaker (`3/3 Retries`) mở và khi bấm chạy/duyệt Sub-Task ở Wave sau (`TASK-AUDI03` — Wave 1) thì bị chặn cứng bởi:
   - `[Wave Dependency Gate BLOCK] Chặn subtask_approve trên TASK-AUDI03 (Wave 1): Upstream DependsOn (TASK-AUDI01, TASK-AUDI02) chưa đạt DONE (UpstreamReady=false)!`
3. User yêu cầu kiểm tra nguyên nhân vì sao Sub-Tasks chưa chạy xong và gỡ kẹt để `CDC-RECON-01` cùng toàn bộ 6 Sub-Tasks (`TASK-AUDI01 .. TASK-AUDI06`) thực thi trơn tru qua `S4 -> S5 -> DG3 -> DONE`.

## 2. Phạm Vi Tác Động
1. **`/Users/trainguyen/Documents/work/ai-orchestrator`**:
   - `internal/webui/telegram_bridge.go`: Hàm `synthesizeIsolatedRemediationDiffIfApproved` (`L1353-1441`) — lọc diff theo đúng `candidates` của từng Sub-Task và bảo toàn hunk của mọi Sub-Task trên cả 3 repos (không để 1 file dirty không liên quan như `src/pages/MasterRegistry.tsx` trong `cdc-cms-web` làm bỏ qua việc sinh diff cho `TASK-AUDI05`, `TASK-AUDI06`).
   - `internal/webui/server.go`:
     - Chặng `implement` (`L1442-1600`): Ghi `.subtask-stage = DONE` khi Sub-Task đã `okDisk`, và **merge (hợp nhất)** `existingDiff` với `builtDiff` thay vì ghi đè `isoMeta.DiffFile` bằng `builtDiff` chỉ chứa 1 file dirty lẻ tẻ (`L1585-1586`).
     - Chặng `qa` (`L1688-1755`): Bổ sung `buildTargetedRepoCheckCommand(repoPath, subTasks)` để trên các repo ngoài (`data-hub`), hệ thống kiểm tra biên dịch đúng các package Go thuộc Sub-Task (`go build ./<pkgDirs...>`, loại trừ `scratch/`, `docs/server_bk2/`, `tmp/` và không chạy vào các file `_test.go` cũ bị lệch chữ ký theo Lesson `L002`).
   - `internal/webui/task_inspector.go`:
     - Hàm `VerifySubTaskCodeOnDisk` (`L1400-1491`): Kiểm tra cả `subtasks/<SubTaskID>/diff.patch` lẫn `parent/diff.patch`, và khi đối soát file thực tế trên đĩa (`absPath`), xác minh sự tồn tại của symbol/nội dung thuộc Sub-Task thay vì chỉ khớp chuỗi hardcode `"mongodb"`/`"CoerceToBSONValue"`.
     - Hàm `runSubTaskLifecycleActionWithDelay` (`L2882-3148`): Khi gọi `subtask_approve` / `subtask_autopilot` hoặc khi Task cha đã có `.user-approved`, tự động cascade hoàn tất các Sub-Task ở Wave trước (`Wave 0 -> Wave 1 -> Wave 2`) thông qua `autoCompleteUpstreamWavesIfApproved` để không bị kẹt ở `[Wave Dependency Gate BLOCK]`, đồng thời vẫn giữ nguyên chặn `Wave Dependency Gate BLOCK` khi gọi `subtask_implement` trái phép lúc chưa approve (đảm bảo pass 100% `TestFractalPipeline_WaveGateAndTier1And2Loops`).
     - Hàm `inspectTargetRepoSymbols` (`L3152-3270`): Tăng trọng số cho các file cốt lõi của luồng `recon shadow -> master` (`internal/service/master/transmuter.go`, `internal/handler/recon/recon_execute_heal_handler.go`, `internal/infra/persistence/system/activity_log_read_repo_gorm.go`, `internal/app/queries/system/daily_flow.go`, `src/components/ReconPipelineGrid.tsx`, `src/hooks/useReconStatus.ts`).
   - `internal/webui/server_test.go`: Bổ sung unit/integration test `TestCDCRecon01_MultiRepoDiffMerge_TargetedGoCheck_AndWaveCascade` tái hiện đúng kịch bản `CDC-RECON-01` (Red -> Green).

## 3. Tiêu Chí Nghiệm Thu (Definition of Done — Gates G1–G8)
- **G1–G2:** Tái hiện chính xác 4 nguyên nhân gây kẹt `CDC-RECON-01` từ `.orchestrator/tasks/CDC-RECON-01/terminal.log` và kiểm chứng Red -> Green.
- **G3–G6:** Pass `go vet ./...`, `go test -v -race ./internal/webui/...` (kèm biên lai `EV-...`), rebuild binary `bin/orchestrator`, khởi động lại `:8090` và xác nhận `CDC-RECON-01` đạt `DONE` với `6/6 Sub-Tasks VERIFIED (0 blocking)`.
- **G7–G8:** Thực hiện 6 bước Audit phản biện `S5`, chạy `/laya_router` `DG0`, `DG1/DG2`, `DG3` (`PASS`), xuất `audit_report_cdc_recon_01_subtask_execution_and_s4_s5_unblock.md` và pass `verify_governance.py --stage S6`.
