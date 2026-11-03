---
tier: T2
suffix: evaluate_ecc_and_ai_orchestrator
---

# 09 — Tasks Solution: Đánh giá Chuyên sâu `affaan-m/ecc` (`v2.2.3`) & Giải pháp Chắt lọc Tinh hoa cho `ai-orchestrator`

## 1. Giải pháp Kiến trúc Duy nhất Tối ưu (Single Best Recommendation — Không bê nguyên xi `ecc`)

Dựa trên kết quả mổ xẻ toàn bộ mã nguồn thật của `affaan-m/ecc` (`v2.2.3`) và đối chiếu với `ai-orchestrator` + `/agent` Core:
- **KHÔNG tích hợp bộ 311 skills / 71 agents hay `ecc2` Rust TUI của `ecc`** vào `ai-orchestrator` vì:
  1. `ai-orchestrator` + `/agent` đã có **Autonomous SDLC Workflow (`S0 -> S6`)**, **12-Step / 6-Wave Sub-Task Engine (`subtask_wave_engine.go:16-430`)**, **Local Neural Decision Gate `/laya_router` (`:8200`, 0 LLM token)**, **2-Tier Tripwire Memory Map (`INDEX.md` + `tripwires.json`)**, và **WebUI Co-Architect Studio + Telegram Forum Bridge** vượt trội hơn hẳn mô hình chạy session rời rạc của `ecc2`.
  2. Codebase `ecc2/` của `ecc` vi phạm nghiêm trọng nguyên tắc `<800 lines` mà chính `AGENTS.md:135,142` đề ra (`dashboard.rs` dài `15,460` dòng, `main.rs` dài `12,830` dòng, `manager.rs` dài `8,181` dòng).
- **CHỈ CHẮT LỌC ĐÚNG 3 CƠ CHẾ KỸ THUẬT ĐÁNG GIÁ NHẤT TỪ `ecc` (`v2.2.3`) VÀO TẦNG GO ENGINE CỦA `ai-orchestrator` (nếu anh `APPROVE`):**
  1. **Cross-SubTask File Overlap & Wave Auto-Promote (`internal/webui/subtask_overlap_guard.go` — cảm hứng từ `FileActivityOverlap` tại `ecc2/src/tui/dashboard.rs:23` và `MessageType::Conflict` tại `ecc2/src/comms/mod.rs:49`):**
     - Khi phân rã hoặc chỉnh sửa Sub-Tasks, nếu 2 Sub-Tasks nằm cùng `WaveIndex` (chạy song song) nhưng có giao tập `TargetFiles` / `ContextFileLine` trên cùng `TargetRepo`, tự động nâng `WaveIndex` của Sub-Task đứng sau lên `WaveIndex + 1` và tự động gắn `DependsOn` vào Sub-Task đứng trước. Điều này triệt tiêu 100% lỗi xung đột merge hoặc lỗi xếp chung `M01` + `M02` vào `Wave 0`.
  2. **First-Touch Caller & Importer Fact-Forcing Context (`internal/webui/subtask_caller_scan.go` — cảm hứng từ `gateguard-fact-force.js:1-73`):**
     - Trước khi `S4 Muscle` (`invokeMuscleSubTaskCodeGen`) sửa một file trong `TargetRepo`, tự động quét nhanh (`ripgrep` / AST scan) các file đang `import` hoặc gọi hàm của file mục tiêu đó để đính kèm danh sách **Verified Callers & Public Signatures** vào prompt của Muscle, ngăn chặn việc sửa gãy signature của tầng Service/Controller gọi vào.
  3. **Canonical Content-Addressable Evidence Digest (`internal/agentgate/evidence_digest.go` — cảm hứng từ `CandidateSpec` tại `ecc2/src/harness_eval.rs:8-95`):**
     - Bổ sung mã băm SHA-256 chuẩn hóa (`CanonicalSHA256(subTaskSpec + diffPatch + testOutput)`) vào biên lai `EV-SUB-...` của từng Sub-Task để chống giả mạo hoặc lệch trạng thái giữa `diff.patch` và kết quả `5A Hard Gate`.

---

## 2. Code Demo (Thiết kế Module đơn trách nhiệm `internal/webui/subtask_overlap_guard.go`)

```go
package webui

import (
	"path/filepath"
	"strings"
)

// SubTaskFileOverlap describes a detected file touchpoint collision between two Sub-Tasks in the same Wave.
type SubTaskFileOverlap struct {
	UpstreamSubTaskID   string `json:"upstream_sub_task_id"`
	DownstreamSubTaskID string `json:"downstream_sub_task_id"`
	SharedFile          string `json:"shared_file"`
	PromotedFromWave    int    `json:"promoted_from_wave"`
	PromotedToWave      int    `json:"promoted_to_wave"`
}

// ResolveParallelWaveFileOverlaps inspects Sub-Tasks in the same WaveIndex and automatically promotes
// colliding downstream Sub-Tasks to WaveIndex+1 while wiring DependsOn (inspired by ECC2 FileActivityOverlap).
func ResolveParallelWaveFileOverlaps(items []SubTaskDTO) ([]SubTaskDTO, []SubTaskFileOverlap) {
	if len(items) <= 1 {
		return items, nil
	}
	out := make([]SubTaskDTO, len(items))
	copy(out, items)

	var overlaps []SubTaskFileOverlap
	for i := 0; i < len(out); i++ {
		filesI := extractNormalizedTouchpointFiles(out[i])
		if len(filesI) == 0 {
			continue
		}
		for j := i + 1; j < len(out); j++ {
			if out[i].WaveIndex != out[j].WaveIndex {
				continue
			}
			if out[i].TargetRepoName != "" && out[j].TargetRepoName != "" && !strings.EqualFold(out[i].TargetRepoName, out[j].TargetRepoName) {
				continue
			}
			filesJ := extractNormalizedTouchpointFiles(out[j])
			if shared := firstSharedFile(filesI, filesJ); shared != "" {
				oldWave := out[j].WaveIndex
				out[j].WaveIndex = oldWave + 1
				out[j].DependsOn = appendUniqueString(out[j].DependsOn, out[i].SubTaskID)
				overlaps = append(overlaps, SubTaskFileOverlap{
					UpstreamSubTaskID:   out[i].SubTaskID,
					DownstreamSubTaskID: out[j].SubTaskID,
					SharedFile:          shared,
					PromotedFromWave:    oldWave,
					PromotedToWave:      out[j].WaveIndex,
				})
			}
		}
	}
	return out, overlaps
}

func extractNormalizedTouchpointFiles(st SubTaskDTO) []string {
	cands := extractCandidateRelPaths(st.TargetFiles+" "+st.ContextFileLine, st.TargetRepoName)
	var norm []string
	for _, c := range cands {
		clean := filepath.ToSlash(strings.TrimSpace(c))
		if clean != "" {
			norm = appendUniqueString(norm, clean)
		}
	}
	return norm
}

func firstSharedFile(a, b []string) string {
	set := make(map[string]bool, len(a))
	for _, item := range a {
		set[strings.ToLower(item)] = true
	}
	for _, item := range b {
		if set[strings.ToLower(item)] {
			return item
		}
	}
	return ""
}

func appendUniqueString(slice []string, val string) []string {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return slice
		}
	}
	return append(slice, val)
}
```

---

## 3. Cập nhật Thiết kế Tự Chữa Lành `S5 -> S3 -> S4 -> S5` (`TASK-ECC-04`)

1. **Đặt đúng Layer theo Design Pattern gốc (`internal/scout/caller_scan.go`):**
   - Chuyển `ScanTouchpointCallersForMuscle` và `ReadSiblingPeerSampleInDir` từ `internal/webui/` sang `internal/scout/caller_scan.go` (`package scout`).
   - Bổ sung bộ trích xuất symbol khai báo thực tế (`extractDeclaredSymbolsFromContent`: `func`, `type`, `class`, `interface`, `const`) từ nội dung file mục tiêu để bắt chính xác caller trong cả repo Go lẫn TypeScript/JS/Python.
2. **Vòng lặp Hội tụ Điểm bất động (`Fixed-Point Convergence`) trong `ResolveParallelWaveFileOverlaps` (`internal/webui/subtask_overlap_guard.go`):**
   - Chạy tối đa `len(out)` vòng hội tụ để khi phần tử `out[j]` được nâng lên `Wave W+1` mà đụng phải `out[k]` (`k < j`) vốn đã ở sẵn `Wave W+1`, hệ thống tiếp tục nâng `out[j]` lên `Wave W+2` và nối `DependsOn`.
   - Chuẩn hóa `effectiveSubTaskRepoName(st)` (ưu tiên `st.TargetRepoName`, fallback sang `filepath.Base(st.TargetRepo)`).
   - Đồng bộ lại tiền tố `Wave %d` trong `WorkflowStep` tại `enrichSubTasksWithRoleAndModel` (`internal/webui/task_chat.go`) khi `WaveIndex` thay đổi.
3. **Bổ sung trường `CanonicalDigest` vào `agentgate.EvidenceReceipt` & Đọc `diff.patch` thật (`internal/agentgate/kernel_bridge.go` & `internal/webui/task_inspector.go`):**
   - Thêm trường `CanonicalDigest string` (`json:"canonical_digest,omitempty"`) vào `EvidenceReceipt`.
   - Trong `writeSubTaskPhysicalEvidenceReceipt`, đọc nội dung `filepath.Join(absSubDir, "diff.patch")` trên đĩa kết hợp `srcFingerprint` để tính `CanonicalDigest`.
