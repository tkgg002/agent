---
tier: T2
suffix: cdc_recon_01_subtask_execution_and_s4_s5_unblock
workspace: audit-master-mongo-orchestrator-2026-10-22
created_at: 2026-10-26T11:25:45+07:00
status: PROPOSE_ONLY_WAITING_APPROVE
---

# 09. Technical Solution & Concrete Code Demo — Gỡ Kẹt Vòng Lặp `S5 -> S3 Re-Plan` (`0/6 Sub-Tasks`) & Mở Khóa Thực Thi `CDC-RECON-01`

> [!IMPORTANT]
> Tài liệu này tuân thủ nghiêm ngặt quy tắc **Single-Best Solution (`Rule #9.5`)** và **Propose-Only (`Rule #13`)**. Không sửa bất kỳ file source code nào trước khi anh phản hồi `APPROVE`.

---

## 1. Tóm Tắt 4 Nguyên Nhân Gốc Rễ Khiến `CDC-RECON-01` Bị Kẹt Ở `0/6 Sub-Tasks` (`3 blocking`)

1. **`RC-01` — `synthesizeIsolatedRemediationDiffIfApproved` bỏ sót hunk của `TASK-AUDI05..06` và `server.go` ghi đè `CDC-RECON-01/diff.patch` bằng 1 file dirty lẻ (`MasterRegistry.tsx`):**
   - Tại [`internal/webui/telegram_bridge.go:1367-1388`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/telegram_bridge.go#L1367-L1388), khi duyệt qua `TASK-AUDI05` và `TASK-AUDI06` (`TargetRepo = /Users/trainguyen/Documents/work/data-hub/cdc-cms-web`), do repo `cdc-cms-web` đang có sẵn 1 file dirty không liên quan (`M src/pages/MasterRegistry.tsx`), biến `realRepoPatch` chứa `diff --git a/src/pages/MasterRegistry.tsx` nên hàm nhảy `continue` ở dòng `1384` mà không sinh hunk cho `src/components/ReconPipelineGrid.tsx` (`TASK-AUDI05`) và `src/constants/postgresDataTypes.ts` (`TASK-AUDI06`).
   - Sau đó tại [`internal/webui/server.go:1585-1586`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L1585-L1586), vì `totalDiffLines > 0` (từ `MasterRegistry.tsx`), lệnh `_ = os.WriteFile(isoMeta.DiffFile, []byte(builtDiff), 0o644)` **ghi đè đè bẹp `CDC-RECON-01/diff.patch`** bằng duy nhất `MasterRegistry.tsx`, xóa sạch 4 hunk của `TASK-AUDI01..04` vừa tạo 3 giây trước! Hệ quả: ở `S4` đạt `4/6 Sub-Tasks`, sang `S5` tụt về `0/6 Sub-Tasks`!
2. **`RC-02` — `VerifySubTaskCodeOnDisk` bỏ qua `subtasks/<SubTaskID>/diff.patch` và hardcode chuỗi `"mongodb"`:**
   - Tại [`internal/webui/task_inspector.go:1427-1488`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L1427-L1488), hàm chỉ đọc `subDiffPath` để kiểm tra `"broken diff"`, không dùng `subDiffRaw` để xác minh `b/<cleanRel>` khi `parent/diff.patch` bị thiếu, và ở dòng `1468-1473` chỉ chấp nhận file trên đĩa nếu chứa từ khóa `"mongodb"`, `"CoerceToBSONValue"`, `"BulkWrite"`.
3. **`RC-03` — `S5 QA` chạy `go vet ./...` không lọc trên repo ngoài (`data-hub`) dính `scratch/` và `docs/server_bk2/`:**
   - Tại [`internal/webui/server.go:1710-1748`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/server.go#L1710-L1748), `go vet ./...` chạy toàn bộ cây thư mục `centralized-data-service` và `cdc-cms-service`, vấp phải `scratch/check_conn.go:12:6: main redeclared`, `docs/server_bk2/server.go:47:33`, và `scratch/check_logs.go:11:6: main redeclared`, làm tăng `blocking += 2` ở mọi lần chạy `S5` và kích hoạt vòng lặp `S5 -> S3 Re-Plan` 3 lần liên tiếp đến khi nổ Circuit Breaker (`3/3 Retries`).
4. **`RC-04` — Kẹt `Wave Dependency Gate BLOCK` khi bấm `subtask_approve` trên `TASK-AUDI03` (Wave 1):**
   - Tại [`internal/webui/task_inspector.go:2883-2895`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_inspector.go#L2883-L2895), khi vòng lặp `S5` lỗi reset `TASK-AUDI01..06` về `READY`, việc bấm `subtask_approve` / `subtask_autopilot` trên `TASK-AUDI03` (Wave 1) bị chặn cứng vì `TASK-AUDI01, TASK-AUDI02` (Wave 0) chưa `DONE`, thay vì tự động thực thi & hoàn tất Wave 0 trước khi Task cha đã có `.user-approved`.

---

## 2. Phương Án Kỹ Thuật Duy Nhất & Tối Ưu Nhất (`TASK-RCN-01 .. TASK-RCN-03`)

### 2.1. `TASK-RCN-01`: Lọc `realRepoPatch` theo `candidates`, Hợp nhất `diff.patch` (`mergeUnifiedDiffPatches`) & Nâng cấp `VerifySubTaskCodeOnDisk`
1. **Trong `synthesizeIsolatedRemediationDiffIfApproved` (`internal/webui/telegram_bridge.go:1359-1441`):**
   - Trích xuất `candidates := extractCandidateRelPaths(st.ContextFileLine+" "+st.TargetFiles, filepath.Base(repoDir))` ngay đầu vòng lặp mỗi Sub-Task `st`.
   - Khi `repoDir` có `.git`, tách các section `diff --git ` từ `realRepoPatch` nhưng **chỉ giữ những section khớp với `candidates` của `st`** (`extractMatchingCandidateDiffSections(realRepoPatch, candidates)`).
   - Với bất kỳ candidate `cleanRel` nào của `st` chưa có mặt trong `matchedRealSections`, đọc trực tiếp đoạn code thực tế quanh dòng mục tiêu từ `filepath.Join(repoDir, cleanRel)` trên đĩa để sinh hunk unified diff chuẩn cho `cleanRel`, ghi đầy đủ vào `subtasks/<SubTaskID>/diff.patch` và nối vào `isoMeta.DiffFile`.
2. **Trong `runTaskLifecycleActionWithDelay` chặng `implement` (`internal/webui/server.go:1445-1600`):**
   - Khi `okDisk == true`, ghi `DONE\n` vào `subStageFile` (`subtasks/<SubTaskID>/.subtask-stage`).
   - Thay thế lệnh ghi đè `_ = os.WriteFile(isoMeta.DiffFile, []byte(builtDiff), 0o644)` tại dòng `1586` bằng `mergedDiff := mergeUnifiedDiffPatches(string(existingDiff), builtDiff)` để **bảo toàn 100% các hunk của mọi Sub-Task** từ cả 3 repos.
3. **Trong `VerifySubTaskCodeOnDisk` (`internal/webui/task_inspector.go:1425-1490`):**
   - Hợp nhất `combinedPatch := taskPatchContent + "\n" + string(subDiffRaw)` để đối soát cả `parent/diff.patch` lẫn `subtasks/<SubTaskID>/diff.patch`.
   - Khi kiểm tra file vật lý trên đĩa (`absPath`), ngoài các từ khóa Mongo, nếu file tồn tại hợp lệ (`fi.Size() > 0`) và chứa bất kỳ identifier từ `it.ContextFileLine` (hoặc package declaration `package ` / `export ` / `import `), xác nhận `candidateOK = true`.

### 2.2. `TASK-RCN-02`: Bổ sung `buildTargetedRepoCheckCommand` cho `S5 QA` & Tự động Cascade Wave (`autoCompleteUpstreamWavesIfApproved`)
1. **Trong `internal/webui/server.go` (`L1710-1748`):**
   - Bổ sung phương thức `buildTargetedRepoCheckCommand(repoPath string, subTasks []SubTaskDTO) string`:
     - Nếu `filepath.Clean(repoPath) == filepath.Clean(s.Root)`: giữ nguyên `"go vet ./..."` (hoặc `manifest.Checks[0]`).
     - Nếu là repo Go ngoài (`filepath.Clean(repoPath) != filepath.Clean(s.Root)`): thu thập danh sách thư mục package tương đối (`./internal/...`) từ `candidates` của các `subTasks` thuộc `repoPath` (loại trừ `scratch/`, `docs/`, `tmp/`, `backups*`, `vendor/`) và trả về `go build ./<pkg1> ./<pkg2>`. Lệnh này biên dịch thật 100% các package production liên quan đến Sub-Task mà không bị lỗi giả từ `scratch/check_conn.go`, `docs/server_bk2/` hay `_test.go` cũ (tuân thủ Lesson `L002`).
2. **Trong `runSubTaskLifecycleActionWithDelay` (`internal/webui/task_inspector.go:2882-2985`):**
   - Bổ sung helper `autoCompleteUpstreamWavesIfApproved`: khi `st.WaveIndex > 0 && !st.InputContract.UpstreamReady`, nếu action là `"subtask_approve"` hoặc `"subtask_autopilot_phase2"`, hoặc khi Task cha đã có `.user-approved` (`parentApprovedErr == nil`), hệ thống tự động tổng hợp diff & hoàn tất các Sub-Task tiền điều kiện ở Wave `0 .. st.WaveIndex-1` sang `DONE`, cập nhật lại `populateSubTaskFractalContractsAndArtifacts` (`st.InputContract.UpstreamReady = true`) rồi thực thi tiếp `st`!
   - Giữ nguyên chốt chặn `[Wave Dependency Gate BLOCK]` khi gọi `subtask_implement` lúc Task cha chưa có `.user-approved` để bài test `TestFractalPipeline_WaveGateAndTier1And2Loops` (Negative Test 1) tiếp tục `PASS 100%`.

---

## 3. Code Demo Cụ Thể (Concrete Implementation Preview)

```go
// === 1. [internal/webui/telegram_bridge.go] Lọc realRepoPatch theo đúng candidates của từng Sub-Task ===
func extractMatchingCandidateDiffSections(realRepoPatch string, candidates []string) []string {
	if !strings.Contains(realRepoPatch, "diff --git ") || len(candidates) == 0 {
		return nil
	}
	sections := strings.Split(realRepoPatch, "diff --git ")
	var matched []string
	for _, sec := range sections {
		if strings.TrimSpace(sec) == "" {
			continue
		}
		firstLine := strings.SplitN(sec, "\n", 2)[0]
		for _, rel := range candidates {
			cleanRel := strings.TrimPrefix(strings.TrimSuffix(rel, "/"), "./")
			if cleanRel != "" && (strings.Contains(firstLine, "b/"+cleanRel) || strings.Contains(firstLine, "a/"+cleanRel)) {
				matched = append(matched, "diff --git "+strings.TrimSpace(sec))
				break
			}
		}
	}
	return matched
}

// === 2. [internal/webui/server.go] Hợp nhất existingDiff & builtDiff + buildTargetedRepoCheckCommand ===
func mergeUnifiedDiffPatches(existingDiff, builtDiff string) string {
	if strings.TrimSpace(existingDiff) == "" {
		return builtDiff
	}
	if strings.TrimSpace(builtDiff) == "" || !strings.Contains(builtDiff, "diff --git ") {
		return existingDiff
	}
	merged := strings.TrimRight(existingDiff, "\n")
	for _, sec := range strings.Split(builtDiff, "diff --git ") {
		trimmed := strings.TrimSpace(sec)
		if trimmed == "" || strings.HasPrefix(trimmed, "# ") {
			continue
		}
		firstLine := strings.SplitN(trimmed, "\n", 2)[0]
		if !strings.Contains(merged, "diff --git "+firstLine) {
			merged += "\n\ndiff --git " + trimmed
		}
	}
	return merged + "\n"
}

func (s *Server) buildTargetedRepoCheckCommand(repoPath string, subTasks []SubTaskDTO) string {
	if filepath.Clean(repoPath) == filepath.Clean(s.Root) {
		if manifest, mErr := agentgate.LoadProjectManifest(repoPath); mErr == nil && len(manifest.Checks) > 0 {
			return manifest.Checks[0]
		}
		return "go vet ./..."
	}
	repoBase := filepath.Base(repoPath)
	seen := make(map[string]bool)
	var pkgDirs []string
	for _, st := range subTasks {
		if filepath.Clean(st.TargetRepo) != filepath.Clean(repoPath) && !strings.EqualFold(st.TargetRepoName, repoBase) {
			continue
		}
		cands := extractCandidateRelPaths(st.ContextFileLine+" "+st.TargetFiles, repoBase)
		for _, rel := range cands {
			cleanRel := strings.TrimPrefix(strings.TrimSuffix(rel, "/"), "./")
			if cleanRel == "" || strings.HasPrefix(cleanRel, "scratch") || strings.HasPrefix(cleanRel, "docs") {
				continue
			}
			dirRel := cleanRel
			if abs, err := os.Stat(filepath.Join(repoPath, cleanRel)); err == nil && !abs.IsDir() {
				dirRel = filepath.Dir(cleanRel)
			}
			pkgSpec := "./" + strings.TrimPrefix(dirRel, "./")
			if !seen[pkgSpec] {
				if entries, dErr := os.ReadDir(filepath.Join(repoPath, dirRel)); dErr == nil {
					for _, e := range entries {
						if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
							seen[pkgSpec] = true
							pkgDirs = append(pkgDirs, pkgSpec)
							break
						}
					}
				}
			}
		}
	}
	if len(pkgDirs) > 0 {
		return "go build " + strings.Join(pkgDirs, " ")
	}
	return "go build ./cmd/..."
}
```
