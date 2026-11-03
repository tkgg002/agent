---
workspace: audit-master-mongo-orchestrator-2026-10-22
suffix: headless_hidden_antigravity_execution
tier: T2
---
# S3 Technical Solution — Thực Thi Ẩn 100% (`Headless Ephemeral Hidden Execution`) Cho Model Antigravity 4.0

## 1. Root Cause Analysis (RCA — Đã Kiểm Chứng Vật Lý 100%)
1. **Tại sao phương án gom chung Conversation (`send-message`) vẫn gây rác IDE của User (`internal/runner/antigravity.go:446-486`):**
   - Trong `TASK-A06`, `completeWithAntigravityAgentAPI` lưu `convID` vào `.orchestrator/tasks/<TASK-ID>/agentapi_conv_id.txt` rồi ở mọi bước Wave (`S1/S2.5 Critique`, `S3 Plan`, `Sub-Task Plan`, `Rep Update Plan / Chat`, `S4 Muscle CodeGen`, `5B Semantic Audit`, `S6 Finalize`) đều bắn `agentapi send-message <convID> <fullPrompt>`.
   - Hệ quả:
     1. Vì lúc tạo `new-conversation` (`internal/runner/antigravity.go:464-468`) không truyền biến môi trường `ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}`, phiên Conversation này vẫn hiển thị trên danh sách Conversation (Sidebar) của Antigravity IDE.
     2. Hàng chục lệnh `send-message` từ đủ mọi vai trò (Brain Architect, Sub-Task Planner, Muscle Coder, QA Auditor) bắn dồn dập vào cùng 1 phiên chat khiến phiên chat đó biến thành "bãi rác" chứa đầy prompt hệ thống, JSON schema, code diff, đồng thời làm ô nhiễm ngữ cảnh (context pollution) giữa các bước Wave.
2. **Cơ chế Chạy Ẩn Nội Tại của `language_server` / `agentapi` (100% VERIFIED):**
   - Binary `/Applications/Antigravity.app/Contents/Resources/bin/language_server` khi xử lý `agentapi new-conversation` đọc biến môi trường `ANTIGRAVITY_SOURCE_METADATA` và unmarshal vào protobuf `exa.cortex_pb.SourceMetadata`:
     - Trường `hideFromConversationList: true` (`protobuf:"varint,3,opt,name=hide_from_conversation_list,json=hideFromConversationList,proto3"`) chỉ thị cho Antigravity IDE **ẩn hoàn toàn** phiên hội thoại khỏi danh sách Conversation trên giao diện.
   - Đồng thời, `language_server` cung cấp Connect-RPC endpoint nội bộ:
     - `POST http://${ANTIGRAVITY_LS_ADDRESS}/exa.language_server_pb.LanguageServerService/DeleteCascadeTrajectory`
     - Header: `Content-Type: application/json`, `x-codeium-csrf-token: ${ANTIGRAVITY_CSRF_TOKEN}`
     - Body: `{"cascadeId": "<convID>"}`
     - Khi gọi endpoint này kèm `os.RemoveAll("~/.gemini/antigravity/brain/<convID>")`, toàn bộ trajectory trong bộ nhớ/SQLite của `language_server` và thư mục log tạm trên đĩa được xoá sạch 100% (`HTTP 200 OK`, body `{}`).

---

## 2. Phương Án Duy Nhất Tối Ưu Nhất: `Headless Ephemeral Hidden Execution`

Thay vì giữ một phiên Conversation sống dai dẳng và bắn `send-message` nối đuôi, `completeWithAntigravityAgentAPI` trong `/Users/trainguyen/Documents/work/ai-orchestrator/internal/runner/antigravity.go` sẽ vận hành theo kiến trúc **Headless Ephemeral Hidden Execution** gồm 4 bước khép kín:

1. **Bước 1 — Tự động khám phá kết nối `language_server` (`resolveAntigravityLSConnection`):**
   - Đọc `ANTIGRAVITY_LS_ADDRESS` và `ANTIGRAVITY_CSRF_TOKEN` từ biến môi trường.
   - Nếu `orchestrator` chạy độc lập từ terminal ngoài chưa có 2 biến này, tự động quét tiến trình `language_server` đang chạy (`ps -ax -o pid=,command=` có `--csrf_token` và `--subclient_type hub`) và cổng TCP LISTEN (`lsof -nP -a -p <pid> -iTCP -sTCP:LISTEN`) để lấy `localhost:<port>` và `<csrf_token>`.
2. **Bước 2 — Kích hoạt Cờ Ẩn Tuyệt Đối (`buildHiddenAntigravityEnv`):**
   - Mọi lệnh gọi `agentapi` đều được gắn cứng `ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}` vào `cmd.Env`.
   - Phiên thực thi hoàn toàn vô hình trên Sidebar/UI của Antigravity IDE ngay từ giây đầu tiên khởi tạo.
3. **Bước 3 — Không bắn `send-message` chồng chéo; Dọn sạch `sessionFile` cũ (`cleanupLegacySessionConversation`):**
   - Mỗi bước Wave (`S1/S2.5`, `S3 Plan`, `Sub-Task Plan`, `Rep Update Plan / Chat`, `S4 Muscle CodeGen`, `5B Semantic Audit`, `S6 Finalize`) chạy trên một phiên ẩn (`hideFromConversationList: true`) độc lập, có ngữ cảnh sạch 100% đúng với vai trò của bước đó, không bắn `send-message` vào bất kỳ phiên chat nào của User.
   - Nếu phát hiện file `agentapi_conv_id.txt` cũ từ phiên bản trước, tự động gọi `DeleteCascadeTrajectory` + xoá `brain/<oldConvID>` + xoá `sessionFile` để dọn sạch tàn dư cũ.
4. **Bước 4 — Tự động huỷ phiên tạm ngay khi đọc xong phản hồi (`defer cleanupHiddenAntigravityConversation`):**
   - Ngay khi đọc xong `PLANNER_RESPONSE` (`status == "DONE"`) từ `transcript_full.jsonl` (hoặc khi `ctx` huỷ), khối `defer` lập tức:
     1. Gọi `POST http://<lsAddr>/exa.language_server_pb.LanguageServerService/DeleteCascadeTrajectory` (`{"cascadeId": convID}`) để xoá trajectory khỏi `language_server`.
     2. Gọi `os.RemoveAll` xoá thư mục `~/.gemini/antigravity/brain/<convID>` và `~/.gemini/antigravity-ide/brain/<convID>`.
   - **Kết quả:** 0 tab mới trên IDE, 0 tin nhắn `send-message` rác, 0 rác tồn đọng trong ` ~/.gemini/antigravity/brain/`.

---

## 3. Code Demo (`internal/runner/antigravity.go`)

```go
const hiddenAntigravitySourceMetadata = `{"hideFromConversationList":true}`

func resolveAntigravityLSConnection() (lsAddr, csrfToken string) {
	lsAddr = strings.TrimSpace(os.Getenv("ANTIGRAVITY_LS_ADDRESS"))
	csrfToken = strings.TrimSpace(os.Getenv("ANTIGRAVITY_CSRF_TOKEN"))
	if lsAddr != "" && csrfToken != "" {
		return lsAddr, csrfToken
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	psOut, err := executil.CommandContext(ctx, "ps", "-ax", "-o", "pid=,command=").Output()
	if err != nil {
		return lsAddr, csrfToken
	}

	var targetPID string
	for _, line := range strings.Split(string(psOut), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "language_server") || !strings.Contains(trimmed, "--csrf_token") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		pidCandidate := fields[0]
		var tokenCandidate string
		for i := 1; i < len(fields); i++ {
			if fields[i] == "--csrf_token" && i+1 < len(fields) {
				tokenCandidate = fields[i+1]
				break
			}
		}
		if tokenCandidate == "" {
			continue
		}
		targetPID = pidCandidate
		if csrfToken == "" {
			csrfToken = tokenCandidate
		}
		if strings.Contains(trimmed, "--subclient_type hub") {
			break
		}
	}

	if lsAddr == "" && targetPID != "" {
		lsofOut, lsofErr := executil.CommandContext(ctx, "lsof", "-nP", "-a", "-p", targetPID, "-iTCP", "-sTCP:LISTEN").Output()
		if lsofErr == nil {
			for _, line := range strings.Split(string(lsofOut), "\n") {
				if idx := strings.LastIndex(line, "127.0.0.1:"); idx >= 0 {
					portPart := strings.Fields(line[idx+len("127.0.0.1:"):])
					if len(portPart) > 0 && portPart[0] != "" {
						lsAddr = "localhost:" + portPart[0]
						break
					}
				}
			}
		}
	}
	return lsAddr, csrfToken
}

func buildHiddenAntigravityEnv(lsAddr, csrfToken string) []string {
	base := os.Environ()
	filtered := make([]string, 0, len(base)+3)
	for _, kv := range base {
		if strings.HasPrefix(kv, "ANTIGRAVITY_SOURCE_METADATA=") {
			continue
		}
		if lsAddr != "" && strings.HasPrefix(kv, "ANTIGRAVITY_LS_ADDRESS=") {
			continue
		}
		if csrfToken != "" && strings.HasPrefix(kv, "ANTIGRAVITY_CSRF_TOKEN=") {
			continue
		}
		filtered = append(filtered, kv)
	}
	filtered = append(filtered, "ANTIGRAVITY_SOURCE_METADATA="+hiddenAntigravitySourceMetadata)
	if lsAddr != "" {
		filtered = append(filtered, "ANTIGRAVITY_LS_ADDRESS="+lsAddr)
	}
	if csrfToken != "" {
		filtered = append(filtered, "ANTIGRAVITY_CSRF_TOKEN="+csrfToken)
	}
	return filtered
}

func cleanupHiddenAntigravityConversation(lsAddr, csrfToken, convID string) {
	convID = strings.TrimSpace(convID)
	if convID == "" || strings.ContainsAny(convID, " \t\n/\\") {
		return
	}
	if lsAddr != "" && csrfToken != "" {
		endpoint := fmt.Sprintf("http://%s/exa.language_server_pb.LanguageServerService/DeleteCascadeTrajectory", strings.TrimPrefix(strings.TrimPrefix(lsAddr, "http://"), "https://"))
		bodyBytes, _ := json.Marshal(map[string]string{"cascadeId": convID})
		reqCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-codeium-csrf-token", csrfToken)
			resp, doErr := http.DefaultClient.Do(req)
			if doErr == nil && resp != nil {
				_ = resp.Body.Close()
			}
		}
		cancel()
	}
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		_ = os.RemoveAll(filepath.Join(homeDir, ".gemini", "antigravity", "brain", convID))
		_ = os.RemoveAll(filepath.Join(homeDir, ".gemini", "antigravity-ide", "brain", convID))
	}
}

func cleanupLegacySessionFile(sessionFile, lsAddr, csrfToken string) {
	if strings.TrimSpace(sessionFile) == "" {
		return
	}
	if oldConvID := readRawAgentAPIConversationID(sessionFile); oldConvID != "" {
		cleanupHiddenAntigravityConversation(lsAddr, csrfToken, oldConvID)
	}
	_ = os.Remove(sessionFile)
}

func readRawAgentAPIConversationID(sessionFile string) string {
	if strings.TrimSpace(sessionFile) == "" {
		return ""
	}
	raw, err := os.ReadFile(sessionFile)
	if err != nil {
		return ""
	}
	convID := strings.TrimSpace(string(raw))
	if convID == "" || strings.ContainsAny(convID, " \t\n/\\") {
		return ""
	}
	return convID
}

func completeWithAntigravityAgentAPI(ctx context.Context, agyBin, prompt, systemPrompt, model, sessionFile, sessionTitle string, ch chan<- Token) {
	defer close(ch)

	if strings.TrimSpace(sessionTitle) == "" {
		sessionTitle = "AI-Orchestrator Hidden Worker"
	}

	tier := "pro"
	lower := strings.ToLower(model)
	if strings.Contains(lower, "flash_lite") || strings.Contains(lower, "flash-lite") {
		tier = "flash_lite"
	} else if strings.Contains(lower, "flash") {
		tier = "flash"
	}

	fullPrompt := prompt
	if strings.TrimSpace(systemPrompt) != "" {
		fullPrompt = systemPrompt + "\n\n" + prompt
	}

	lsAddr, csrfToken := resolveAntigravityLSConnection()
	cleanupLegacySessionFile(sessionFile, lsAddr, csrfToken)

	newArgs := []string{"new-conversation", "--model=" + tier, "--title=" + sessionTitle, fullPrompt}
	if strings.HasSuffix(agyBin, "language_server") {
		newArgs = append([]string{"agentapi"}, newArgs...)
	}
	cmd := executil.CommandContext(ctx, agyBin, newArgs...)
	cmd.Env = buildHiddenAntigravityEnv(lsAddr, csrfToken)
	out, err := cmd.CombinedOutput()
	if err != nil {
		ch <- Token{Done: true, Error: fmt.Errorf("antigravity native error: %w (%s)", err, strings.TrimSpace(string(out)))}
		return
	}
	var resp agentAPIResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		ch <- Token{Done: true, Error: fmt.Errorf("antigravity parse response: %w (%s)", err, strings.TrimSpace(string(out)))}
		return
	}
	convID := strings.TrimSpace(resp.Response.NewConversation.ConversationID)
	if convID == "" {
		ch <- Token{Done: true, Error: fmt.Errorf("antigravity returned empty conversationId: %s", strings.TrimSpace(string(out)))}
		return
	}
	defer cleanupHiddenAntigravityConversation(lsAddr, csrfToken, convID)
```

---

## 4. Tách Module Theo Trách Nhiệm Đơn (Single Responsibility Module Split — `TASK-HIDE-04`)
- **Không nhồi nhét vào `internal/runner/antigravity.go`:**
  - `internal/runner/antigravity.go` (`341` dòng `<= HEAD 525` dòng): Chỉ đảm nhiệm `AntigravityRunner`, `DiscoverAvailableModelRunners()`, `Complete()`, và điều phối CLI (`resolveCommand`, `streamAntigravityOutput`, `findAntigravityBinary`, `findGeminiBinary`, `findClaudeBinary`).
  - `internal/runner/antigravity_agentapi_hidden.go` (file mới đơn trách nhiệm): Đảm nhiệm toàn bộ giao thức `agentapi` & Connect-RPC ẩn (`hiddenAntigravitySourceMetadata`, `resolveAntigravityLSConnection`, `buildHiddenAntigravityEnv`, `cleanupHiddenAntigravityConversation`, `cleanupLegacySessionFile`, `readRawAgentAPIConversationID`, `completeWithAntigravityAgentAPI`).

## 5. Kế hoạch Kiểm chứng (Verification Plan — S4 & S5)
1. **Unit Test (`internal/runner/antigravity_test.go`):**
   - Kiểm chứng `buildHiddenAntigravityEnv` luôn ghi đè `ANTIGRAVITY_SOURCE_METADATA={"hideFromConversationList":true}`.
   - Kiểm chứng `completeWithAntigravityAgentAPI` và `cleanupLegacySessionFile` (kể cả khi `brain/<oldConvID>` đã xoá khỏi đĩa vẫn gọi `DeleteCascadeTrajectory`).
2. **Real Host Non-Mock E2E Test (`TestAntigravityRunner_RealHostLanguageServerHiddenE2E`):**
   - Gọi trực tiếp `/Users/trainguyen/.gemini/antigravity/bin/agentapi` và `language_server` thật trên máy (0 mock script, 0 mock HTTP server), kiểm chứng phản hồi thật từ model và xác nhận qua `agentapi get-conversation-metadata` rằng phiên ẩn đã xoá sạch khỏi `language_server` sau khi kết thúc.
