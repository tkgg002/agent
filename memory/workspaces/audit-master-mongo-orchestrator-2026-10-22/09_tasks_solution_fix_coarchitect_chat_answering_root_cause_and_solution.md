---
tier: T2
suffix: fix_coarchitect_chat_answering_root_cause_and_solution
mode: standard
status: PROPOSE_ONLY_WAITING_APPROVE
---

# 09. Giải Pháp Kỹ Thuật Tối Ưu (S3 Single-Best Solution) — Trả Lời Trực Tiếp Câu Hỏi Kỹ Thuật Trong `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`) & Đồng Bộ Hồ Sơ `CDC-RECON-01`

## 1. Kiến trúc Giải pháp Duy Nhất (Single-Best Solution)

1. **Nâng cấp `buildContextualDiscussReply` & Bổ sung `extractTaskDeepKnowledgeForDiscussion` trong [`internal/webui/task_chat.go:1325-1399`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L1325-L1399):**
   - Đọc kết hợp hồ sơ của Task từ `.orchestrator/tasks/<TaskKey>/` (`09_tasks_solution.md`, `13_analysis.md`, `audit_report.md`, `08_tasks.md`) **VÀ** workspace `/agent/memory/workspaces/` liên kết (bao gồm cả `audit-shadow-master-update-drop-2026-10-20` cho `CDC-RECON-01`).
   - Trích xuất đầy đủ:
     - **🔍 Nguyên nhân gốc rễ (`Vì sao chạy không đúng`):** Các mục `Sơ hở F1..F8` / `Root Cause` kèm `file:line` và số liệu DB thực tế.
     - **💡 Giải pháp kỹ thuật & Code Demo (`Giải pháp là gì`):** Các mục `Giải pháp cho Sơ hở F1..F8`, `AF-01..AF-05` và khối `Code Demo`.
     - **📋 Danh mục Sub-Tasks & Điểm chạm Đa Repo (`08_tasks.md`):** Trạng thái từng Sub-Task và các file thay đổi trên từng Repo.
   - Khi User hỏi bất kỳ câu hỏi nào về nguyên nhân/giải pháp/kiến trúc (`vì sao`, `tại sao`, `nguyên nhân`, `ko đúng`, `sai`, `lỗi`, `giải pháp`, `sửa`, `fix`, `làm sao`, `code`, `file`, hoặc câu hỏi thảo luận chung), `buildContextualDiscussReply` trả lời trực tiếp, đầy đủ và chi tiết vào đúng câu hỏi của User thay vì nhại lại câu hỏi + liệt kê đường dẫn file!
2. **Đồng bộ hồ sơ thực chứng của `CDC-RECON-01` & Cập nhật `chat_history.json`:**
   - Bảo toàn nội dung phân tích `F1..F8` trong `.orchestrator/tasks/CDC-RECON-01/13_analysis.md` và cập nhật ngay câu trả lời lúc `15:21:12` trong `.orchestrator/tasks/CDC-RECON-01/chat_history.json` để giao diện `:8090` hiển thị câu trả lời chi tiết ngay lập tức.
3. **Hỗ trợ Hỏi-Đáp Kỹ Thuật trực tiếp trên Telegram (`internal/webui/telegram_bridge.go:495-535`):**
   - Khi User nhắn câu hỏi thảo luận (`vì sao...`, `giải pháp là gì...`) trên Telegram, tự động gọi `buildContextualDiscussReply` và gửi câu trả lời chi tiết qua Telegram.

---

## 2. Code Demo Chi Tiết (`internal/webui/task_chat.go`)

```go
// Trong internal/webui/task_chat.go (thay thế nhánh fallback khuôn mẫu tại L1379-1399):
func (s *Server) buildContextualDiscussReply(
	qt queue.QueuedTask,
	isoMeta TaskIsolationDTO,
	subTasks []SubTaskDTO,
	cleanMsg string,
	agentRoleLabel, agentSlot, skillBadge, selectedModel string,
	lessonRefs []string,
) string {
	lowerMsg := strings.ToLower(cleanMsg)
	// ... (giữ Case 1: "mày là ai" & Case 2: "so với antigravity") ...

	// Bóc tách toàn bộ Nguyên nhân gốc rễ (Root Causes F1..Fn), Giải pháp kỹ thuật (Solutions) & Code Demo
	// từ 13_analysis.md, 09_tasks_solution.md (cả trong .orchestrator/tasks/<TaskKey> và /agent/memory/workspaces/)
	rootCauses, solutions, codeSnippet, fileTouches := s.extractTaskDeepKnowledgeForDiscussion(qt, isoMeta, subTasks)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**[%s · `%s` · `%s`]** Dạ anh, em xin trả lời trực tiếp câu hỏi của anh trên **`%s` (%s)**:\n\n",
		agentRoleLabel, agentSlot, selectedModel, qt.ExternalID, qt.Title))

	if len(rootCauses) > 0 {
		sb.WriteString("### 🔍 1. Nguyên nhân gốc rễ (Vì sao chạy không đúng)\n")
		for i, rc := range rootCauses {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rc))
		}
		sb.WriteString("\n")
	}

	if len(solutions) > 0 {
		sb.WriteString("### 💡 2. Giải pháp kỹ thuật khắc phục triệt để\n")
		for i, sol := range solutions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, sol))
		}
		sb.WriteString("\n")
	}

	if strings.TrimSpace(codeSnippet) != "" && (strings.Contains(lowerMsg, "giải pháp") || strings.Contains(lowerMsg, "code") || strings.Contains(lowerMsg, "sửa") || strings.Contains(lowerMsg, "fix") || strings.Contains(lowerMsg, "vì sao")) {
		sb.WriteString("### 🧩 3. Trích đoạn Code Demo cốt lõi\n")
		sb.WriteString("```go\n" + truncateRunes(codeSnippet, 900) + "\n```\n\n")
	}

	if len(fileTouches) > 0 {
		sb.WriteString("### 📂 4. Điểm chạm Đa Repository & Sub-Tasks (`file:line`)\n")
		for _, ft := range fileTouches {
			sb.WriteString("- " + ft + "\n")
		}
	}

	return strings.TrimSpace(sb.String())
}
```
