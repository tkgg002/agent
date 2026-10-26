# 08. Tasks Checklist — Xử Lý Lỗi Kẹt `CDC-RECON-01` (`0/6 Sub-Tasks`, `S5 -> S3 Re-Plan` Loop & `Wave Dependency Gate BLOCK`)

> **Workspace**: `audit-master-mongo-orchestrator-2026-10-22`
> **Tier**: `T2`
> **Suffix**: `cdc_recon_01_subtask_execution_and_s4_s5_unblock`

## Nhóm 1: Khắc Phục Ghi Đè `diff.patch` Đa Repo & Đối Soát `VerifySubTaskCodeOnDisk` (`TASK-RCN-01`)
- [x] **TASK-RCN-01 — Bảo toàn `diff.patch` của mọi Sub-Task qua các Repo & nâng cấp `VerifySubTaskCodeOnDisk`** (`EV-20261026T120809-23aa`, `EV-20261026T121100-4601`)
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:1353-1474`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1442-1602, 3085-3152`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1250-1305, 1400-1495`
  - **Mô tả**:
    1. Trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:1359-1474`), trích xuất `candidates := extractCandidateRelPaths(...)` trước; chỉ sử dụng hunk từ `realRepoPatch` nếu hunk đó thực sự chạm vào file trong `candidates` của Sub-Task `st`. Với bất kỳ candidate nào chưa có trong `realRepoPatch`, đọc đoạn code thực tế từ file trên đĩa (`filepath.Join(repoDir, cleanRel)`) để tổng hợp hunk unified diff cho `st`, ghi vào `subtasks/<SubTaskID>/diff.patch` và nối vào `isoMeta.DiffFile`.
    2. Trong `server.go:1583-1601` & `server.go:3085-3152`, hợp nhất (`mergeUnifiedDiffPatches`) giữa `existingDiff` và `builtDiff` để không mất các hunk đã tổng hợp của các repo khác, đồng thời ghi `.subtask-stage = DONE` và reset `WaveRollbackTriggered = false` khi `VerifySubTaskCodeOnDisk` đạt `true`.
    3. Trong `VerifySubTaskCodeOnDisk` (`task_inspector.go:1425-1495`), kiểm tra kết hợp `parent/diff.patch` và `subtasks/<SubTaskID>/diff.patch` (chỉ khi có header `"diff --git "`), khớp chính xác `" b/"+cleanRel` / `"\n+++ b/"+cleanRel` để tránh false-positive từ tên thư mục `/cdc-cms-web/`.
  - **Test / DoD**: `VerifySubTaskCodeOnDisk` đạt `6/6 Sub-Tasks` cho `CDC-RECON-01` ở cả `S4` và `S5`, không bị mất diff khi `cdc-cms-web` có file dirty không liên quan (`MasterRegistry.tsx`).

## Nhóm 2: Lọc Package Mục Tiêu Khi Chạy Check Ở `S5 QA` & Tự Động Cascade Wave (`TASK-RCN-02`)
- [x] **TASK-RCN-02 — Bổ sung `buildTargetedRepoCheckCommand` (tránh `scratch/` & `docs/server_bk2/`) và mở khóa `Wave Dependency Gate`** (`EV-20261026T120817-63bd`, `EV-20261026T121100-4601`)
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1715-1752, 3154-3222`, `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:2888-3005, 3180-3280`
  - **Mô tả**:
    1. Trong `server.go:1715-1752` & `server.go:3154-3222`, bổ sung `buildTargetedRepoCheckCommand(repoPath string, subTasks []SubTaskDTO) string`: với repo ngoài (`filepath.Clean(repoPath) != filepath.Clean(s.Root)`), thu thập danh sách thư mục package Go thực tế của các Sub-Task thuộc `repoPath` (loại trừ `scratch/`, `docs/`) và chạy `go build ./<pkgDirs...>` để kiểm chứng biên dịch code production thật mà không bị fail giả bởi `scratch/check_conn.go`, `docs/server_bk2/` hay `_test.go` cũ.
    2. Trong `runSubTaskLifecycleActionWithDelay` (`task_inspector.go:2888-2946`), bổ sung `autoCompleteUpstreamWavesIfApproved`: khi người dùng bấm `subtask_approve` / `subtask_autopilot` trên Sub-Task ở Wave > 0, tự động thực thi & hoàn tất các Sub-Task tiền điều kiện ở Wave trước (`Wave 0 -> Wave 1 -> Wave 2`), đồng thời giữ nguyên chốt chặn `Wave Dependency Gate BLOCK` khi gọi `subtask_implement` lúc chưa approve để pass `TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch`.
    3. Trong `inspectTargetRepoSymbols` (`task_inspector.go:3261-3277`), tăng trọng số cho các file cốt lõi của `recon shadow -> master` (`service/recon/`, `handler/recon/`, `service/master/transmuter.go`, `persistence/system/activity_log_read_repo_gorm.go`, `queries/system/daily_flow.go`, `components/ReconPipelineGrid.tsx`, `hooks/useReconStatus.ts`).
  - **Test / DoD**: `S5 QA` của `CDC-RECON-01` pass `0 blocking` với biên lai `EV-...` thật, không còn lặp `S5 -> S3 Re-Plan` hay kẹt `Wave Dependency Gate BLOCK`.

## Nhóm 3: Kiểm Thử Tích Hợp & Xác Minh Runtime `:8090` (`TASK-RCN-03`)
- [x] **TASK-RCN-03 — Viết test `TestCDCRecon01_MultiRepoDiffMerge_TargetedGoCheck_AndWaveCascade` & verify live `:8090`** (`EV-20261026T121100-4601`, `EV-20261026T121113-3e7f`, `EV-20261026T121143-eb4a`)
  - **Context (`file:line`)**: `/Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server_test.go:2303-2401`
  - **Mô tả**: Viết bài test kiểm chứng toàn trình `CDC-RECON-01` (gồm cả trường hợp 1 repo có file dirty không liên quan, targeted Go check bỏ qua `scratch/`, và cascade Wave 0 -> Wave 1 -> Wave 2), chạy `evidence_run.py` lấy biên lai `EV-...`, khởi động lại server `:8090` và kích hoạt `CDC-RECON-01` hoàn tất `DONE` (`6/6 Sub-Tasks VERIFIED`, `HandoffReady = true`).
  - **Test / DoD**: Pass 100% test suites với `-race`, live `:8090` báo `CDC-RECON-01` `Stage = DONE`, `6/6 Sub-Tasks = DONE`, `DG3 = PASS`.
