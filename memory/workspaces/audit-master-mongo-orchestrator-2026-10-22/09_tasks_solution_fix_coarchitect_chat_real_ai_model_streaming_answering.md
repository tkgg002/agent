# 09. Tasks Solution — Đấu Nối 100% Model AI Thật Vào Co-Architect Chat (Chấm Dứt Ghép Chuỗi Tĩnh & Nuốt JSON Model) (`fix_coarchitect_chat_real_ai_model_streaming_answering`)

- **Tier:** T2
- **Suffix:** fix_coarchitect_chat_real_ai_model_streaming_answering
- **Phương án tối ưu duy nhất:** Đấu nối trực tiếp Model AI thật sự (`AntigravityRunner` với model nhẹ `gemini-2.5-flash` qua CLI / API) vào 100% câu hỏi đối thoại trong Co-Architect Studio, sửa `streamAntigravityOutput` trong `internal/runner/antigravity.go` để chuyển giao nguyên vẹn chuỗi JSON của Model (không nuốt mất dòng JSON bắt đầu bằng `{`), và xóa bỏ hoàn toàn cơ chế ghi đè chuỗi tĩnh trong `internal/webui/task_chat_knowledge.go`.

---

## 1. 📌 Tóm tắt & Trả lời trực tiếp (Summary)

- **Triệu chứng & Nguyên nhân tận gốc:**
  1. **Bị chặn ở tầng gọi LLM (`task_llm_bridge.go:173`):** Câu lệnh `if actionMode != "discuss"` đã chặn 100% các câu hỏi chat của User (vốn có `actionMode == "discuss"`), khiến hàm `invokeArchitectLLM` không bao giờ gọi Model thật mà rơi thẳng xuống hàm ghép chuỗi tĩnh `synthesizeSymbolGroundedArchitectAnalysis`.
  2. **Bị nuốt token trong `antigravity.go:232-242` (`streamAntigravityOutput`):** Khi Model trả về một dòng JSON hợp lệ `{"direct_answer": "..."}`, hàm kiểm tra `if strings.HasPrefix(trimmed, "{")` rồi gọi `extractStreamJSONText(ev)`. Do `extractStreamJSONText` chỉ tìm các key của Claude (`content`, `text`, `response`), nó trả về `""` và sau đó chạy `continue`, **nuốt mất toàn bộ dòng JSON của Model** khiến ứng dụng nhận được `raw.String() == ""` và lại rơi vào fallback ghép chuỗi!
  3. **Bị ghi đè văn mẫu tĩnh trong `task_chat_knowledge.go:38-46`:** Ngay cả khi Model trả về kết quả, đoạn mã `if !isProgressOrStatusQuestion(cleanMsg)` lại lấy `fbRC`, `fbSol` và câu văn mẫu `synthesizeProgressAwareDirectAnswer` ghi đè lên `directAnswer` của Model!
- **Giải pháp dứt điểm:**
  1. **Trong `internal/runner/antigravity.go:232-245`:**
     - Nếu `extractStreamJSONText(ev)` trả về `""` và `backendKind != "claude"` (hoặc không có key `type`), không được `continue` nuốt mất dòng đó, mà phải chuyển tiếp nguyên văn dòng `line + "\n"` để ứng dụng nhận được trọn vẹn chuỗi JSON của Model!
     - Trong `buildGeminiCLIArgs`: Khi `cleanModel` là `gemini-4-argon` hoặc rỗng, tự động truyền `-m gemini-2.5-flash` để phản hồi cực nhanh (`~3–5s`).
  2. **Trong `internal/webui/task_llm_bridge.go:170-245`:**
     - Xóa bỏ hoàn toàn điều kiện `if actionMode != "discuss"` để 100% câu hỏi chat đều gọi Model AI thật sự.
     - Cải tiến System Prompt và User Message: Cung cấp ngữ cảnh Task ngắn gọn và hướng dẫn Model trả lời trực tiếp, thông minh, đúng bản chất câu hỏi của User (hỏi chào hỏi thì chào hỏi, hỏi tốc độ thì trả lời tốc độ, hỏi tiến độ/kỹ thuật thì trả lời tiến độ/kỹ thuật).
  3. **Trong `internal/webui/task_chat_knowledge.go:30-65`:**
     - Giữ nguyên 100% câu trả lời thông minh của Model AI (`llmRes.DirectAnswer`). Tuyệt đối không ghi đè bằng `synthesizeProgressAwareDirectAnswer`.

---

## 2. 💻 Code Demo Cốt Lõi

```go
// File: internal/runner/antigravity.go:232-245
		if strings.HasPrefix(trimmed, "{") {
			var ev map[string]any
			if err := json.Unmarshal([]byte(trimmed), &ev); err == nil {
				if txt := extractStreamJSONText(ev); txt != "" {
					totalOut += len(txt)
					ch <- Token{Text: txt}
					continue
				}
				// If ev is NOT a Claude envelope event (no "type" field), pass the raw JSON line through!
				if _, hasType := ev["type"]; !hasType && backendKind != "claude" {
					totalOut += len(line) + 1
					ch <- Token{Text: line + "\n"}
					continue
				}
			}
		}

// File: internal/webui/task_llm_bridge.go:170-175
	// Always invoke real AI Model for ALL modes (including "discuss")!
	if !isTestTaskOrTempRoot(s.Root, qt.ExternalID) && !strings.HasSuffix(os.Args[0], ".test") {
		llmCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		defer cancel()
		// agRunner invocation with gemini-2.5-flash in os.TempDir() ...
	}

// File: internal/webui/task_chat_knowledge.go:30-50
	if llmRes, err := s.invokeArchitectLLM(context.Background(), qt, subTasks, cleanMsg, selectedModel, "discuss"); err == nil && llmRes != nil {
		rootCauses = llmRes.RootCauses
		solutions = llmRes.Solutions
		codeSnippet = strings.TrimSpace(llmRes.CodeDemo)
		directAnswer = strings.TrimSpace(llmRes.DirectAnswer) // Keep 100% Model AI answer!
	}
```

---

## 3. 🛡️ Bảng Đối Soát Cổng `H4b` & Bằng Chứng `REF:`

- `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go:232-245`
- `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_llm_bridge.go:170-195`
- `REF: /Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat_knowledge.go:30-55`
