---
tier: T2
suffix: cdc_recon_01_subtask_execution_and_s4_s5_unblock
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-26T11:23:00+07:00
---

# 13. Deep Investigation (`S2`) & Pre-Plan Critique (`S2.5 / DG0`) — Vì Sao `CDC-RECON-01` Bị Kẹt Ở `0/6 Sub-Tasks` & `Wave Dependency Gate BLOCK`

## 1. Bảng Truy Vết Root Cause Vật Lý (`VERIFIED` vs `HYPOTHESIS`)

| ID | Phân loại | Vị trí (`file:line`) | Cơ chế Gây Lỗi Thực Tế |
| :--- | :---: | :--- | :--- |
| `RC-01` | `VERIFIED` | `internal/webui/telegram_bridge.go:1367-1388` & `internal/webui/server.go:1583-1599` | **Ghi đè `diff.patch` đa Repo bằng 1 file dirty không liên quan (`MasterRegistry.tsx`):** Trong `synthesizeIsolatedRemediationDiffIfApproved` (`telegram_bridge.go:1369-1385`), khi duyệt tới `TASK-AUDI05` và `TASK-AUDI06` (`TargetRepo = cdc-cms-web`), do `cdc-cms-web` đang có sẵn 1 file dirty không liên quan (`M src/pages/MasterRegistry.tsx`), `realRepoPatch` khác rỗng nên hàm nhảy `continue` ở dòng `1384` mà **không sinh diff cho `ReconPipelineGrid.tsx` và `postgresDataTypes.ts`**. Tiếp đó, tại `server.go:1585-1586`, vì `totalDiffLines > 0` (từ `MasterRegistry.tsx`), lệnh `os.WriteFile(isoMeta.DiffFile, []byte(builtDiff), 0o644)` **ghi đè đè bẹp toàn bộ `CDC-RECON-01/diff.patch`** bằng duy nhất `MasterRegistry.tsx`, xóa sạch 4 hunk của `TASK-AUDI01..04` vừa tạo 3 giây trước! Bằng chứng trong `terminal.log:12-16`: lúc `11:09:29` (`S4`) đạt `4/6 Sub-Tasks`, nhưng lúc `11:09:39` (`S5`) tụt về `0/6 Sub-Tasks`! |
| `RC-02` | `VERIFIED` | `internal/webui/task_inspector.go:1427-1488` | **`VerifySubTaskCodeOnDisk` bỏ qua `subtasks/<SubTaskID>/diff.patch` & hardcode từ khóa `"mongodb"`:** Tại dòng `1428`, hàm đọc `subDiffRaw` (`subtasks/<SubTaskID>/diff.patch`) nhưng chỉ kiểm tra chuỗi `"broken diff"`, hoàn toàn không dùng `subDiffRaw` để xác minh `b/<cleanRel>` khi `parent/diff.patch` bị ghi đè. Đồng thời tại dòng `1468-1473`, khi kiểm tra file vật lý trên đĩa (`absPath`), hàm chỉ cho `candidateOK = true` nếu nội dung file chứa chuỗi `"mongodb"`, `"CoerceToBSONValue"`, `"BulkWrite"`, khiến mọi Sub-Task không phải Mongo (như `CDC-RECON-01`) bị đánh trượt `"unverified on disk"`. |
| `RC-03` | `VERIFIED` | `internal/webui/server.go:1710-1748` & `CDC-RECON-01/terminal.log:18-45` | **`S5 QA` chạy `go vet ./...` không lọc trên repo ngoài (`data-hub`) dính `scratch/` và `docs/server_bk2/`:** Tại `server.go:1711`, `checkCmd := "go vet ./..."` chạy toàn bộ cây thư mục của `centralized-data-service` và `cdc-cms-service`, vấp phải các file nháp `scratch/check_conn.go:12:6: main redeclared`, `docs/server_bk2/server.go:47:33: undefined: handlerrecon.DLQHandler`, và `scratch/check_logs.go:11:6: main redeclared`. Kết quả là mỗi lần chạy `S5` đều bị cộng `blocking += 2`, đẩy Task quay về `S5 -> S3 Re-Plan` lặp 3 lần liên tiếp đến khi nổ Circuit Breaker (`3/3 Retries`). |
| `RC-04` | `VERIFIED` | `internal/webui/task_inspector.go:2883-2895, 2961-2982` | **Kẹt `Wave Dependency Gate BLOCK` khi bấm `subtask_approve` trên `TASK-AUDI03` (Wave 1):** Vì vòng lặp `S5` lỗi đã reset toàn bộ `TASK-AUDI01..06` về `READY` (`server.go:1788-1790`), khi User bấm chạy/duyệt `TASK-AUDI03` (Wave 1), `checkUpstreamWaveGate("subtask_approve")` chặn cứng do `TASK-AUDI01, TASK-AUDI02` (Wave 0) chưa `DONE`, thay vì tự động thực thi tuần tự các Sub-Task tiền điều kiện ở Wave 0 khi Task cha đã được User `APPROVE`. |

## 2. Kịch Bản Tái Hiện Lỗi (`Gate G2: Red -> Green`)
- **Trạng thái `RED` (Hiện tại trong `.orchestrator/tasks/CDC-RECON-01/`):**
  - `terminal.log:12-124`:
    - `11:09:29 [S4 Implement · B] CDC-RECON-01: HARD GATE BLOCK: only 4/6 Sub-Tasks verified on disk`
    - `11:09:39 [S5 QA & DG3 · K] CDC-RECON-01 Verdict=BLOCK | [centralized-data-service] go vet ./... FAILED (EV-20261026T040933-aa34): vet: scratch/check_conn.go:12:6 ... | [cdc-cms-service] go vet ./... FAILED (EV-20261026T040938-1b6d): vet: scratch/check_logs.go:11:6 ... | Sub-Task Gate BLOCK: 0/6 verified on disk`
    - `11:12:27 [Wave Dependency Gate BLOCK] Chặn subtask_approve trên TASK-AUDI03 (Wave 1): Upstream DependsOn (TASK-AUDI01, TASK-AUDI02) chưa đạt DONE (UpstreamReady=false)!`
- **Trạng thái `GREEN` (Sau khi áp dụng giải pháp ở `S4`):**
  - `synthesizeIsolatedRemediationDiffIfApproved` + `mergeUnifiedDiffPatches` bảo toàn đủ 6 hunk của `TASK-AUDI01..06` trong `CDC-RECON-01/diff.patch` và từng `subtasks/TASK-AUDI0X/diff.patch`.
  - `buildTargetedRepoCheckCommand` kiểm chứng biên dịch đúng các package Go của Sub-Task (`go build ./internal/...`) với `exit_code == 0` (`EV-...`).
  - `CDC-RECON-01` chạy `S4 -> S5 -> DG3` đạt `Stage = DONE`, `6/6 Sub-Tasks = DONE`, `0 blocking`.

## 3. Wave Phản Biện Trước Khi Lên Plan (`S2.5 / DG0` — `/laya_router` Local `:8200`, 0 LLM Token)
| Claim ID | Hạng mục Phản Biện | Kết quả | Bằng chứng Vật lý (`REF:`) |
| :--- | :--- | :---: | :--- |
| `C-REQ-01` | Kiểm chứng `synthesizeIsolatedRemediationDiffIfApproved` nhảy `continue` sớm khi `realRepoPatch` chứa file dirty không thuộc Sub-Task | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go:1367-1388` |
| `C-REALITY-01` | Kiểm chứng `server.go` ghi đè `isoMeta.DiffFile` bằng `builtDiff` và chạy `go vet ./...` không lọc trên repo ngoài | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go:1582-1599` |
| `C-CONTRA-01` | Kiểm chứng `VerifySubTaskCodeOnDisk` bỏ qua `subDiffRaw` và `checkUpstreamWaveGate` chặn `subtask_approve` | `VERIFIED` | `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go:1425-1452` |
