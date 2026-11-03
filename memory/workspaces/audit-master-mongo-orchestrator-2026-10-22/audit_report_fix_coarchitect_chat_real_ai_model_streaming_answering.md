# Mandatory Final Adversarial Audit Report (Tier T2)
> Workspace: `audit-master-mongo-orchestrator-2026-10-22`  
> Suffix: `fix_coarchitect_chat_real_ai_model_streaming_answering`  
> Date: 2026-10-27  

---

## Mục 1: Phản biện từng dòng update / git diff & từng task (line-by-line diff critique)
- **`internal/runner/antigravity.go`**:
  - `buildGeminiCLIArgs`: Khi không truyền model flag, tự động fallback về model nhanh `gemini-2.5-flash` (`-m gemini-2.5-flash`) thay vì dùng model nặng.
  - `streamAntigravityOutput`: Thay vì drop toàn bộ dòng JSON khi `extractStreamJSONText(ev) == ""`, code bổ sung kiểm tra `if _, hasType := ev["type"]; hasType` (chỉ bỏ qua Claude streaming event envelopes). Mọi output payload JSON thật từ Gemini CLI đều được giữ lại nguyên vẹn và emit cho caller.
  - Bổ sung PATH environment chứa node binary (`~/.nvm/versions/node/v22.18.0/bin`) để đảm bảo CLI script khởi chạy thành công trong process con.
- **`internal/webui/task_llm_bridge.go`**:
  - Gỡ bỏ hoàn toàn điều kiện chặn `if actionMode != "discuss"` (dòng 185), cho phép 100% tin nhắn chat discuss kết nối trực tiếp vào AI Model qua `agRunner.Complete`.
  - Nâng timeout context từ 10s lên 25s để chống drop do cold-start.
  - Sửa `parseArchitectLLMResponse`: Hỗ trợ parse JSON độc lập chỉ có `direct_answer` hoặc plain text/markdown mà không bắt buộc phải có đầy đủ mảng `solutions/root_causes`.
  - Xóa bỏ silent fallback: Khi runner trả về lỗi (`streamErr != nil` hoặc `err != nil`), hệ thống trả về thông báo lỗi trung thực tới User thay vì âm thầm rơi xuống `synthesizeSymbolGroundedArchitectAnalysis` để sinh văn bản giả mạo.
- **`internal/webui/task_chat_knowledge.go`**:
  - Bảo vệ `directAnswer`: Tuyệt đối không ghi đè `directAnswer` bằng static template `synthesizeProgressAwareDirectAnswer` khi model đã trả về câu trả lời.
  - Thêm `isDirectConversationOrGeneralQuestion`: Nhận diện các câu hỏi đàm thoại chung (như *"mày trả lời nhanh không"*), trả lời trực tiếp ngắn gọn mà không nhồi nhét bảng Sub-Task table và code demo không liên quan.
  - Thêm `isCodeRequestMessage`: Chỉ đính kèm code demo khi User thực sự yêu cầu xem code.

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật
- Không dùng smoke test hay mockup để suy diễn. Đã kiểm thử runtime thật trên HTTP API qua `evidence_run.py`:
  - `EV-20261027T164732-348c`: `go test -v ./internal/runner/...` (PASS)
  - `EV-20261027T164739-7406`: `go test -v ./internal/webui/ -run 'TestArchitectLLM|TestTaskChat'` (PASS)
  - `EV-20261027T164757-5ae3`: `go test -v ./internal/webui/ -run 'TestRealAIStreamingAndDirectConversation'` (PASS)
  - `EV-20261027T165245-c3bd`: `go vet ./...` (PASS)
  - `EV-20261027T164802-92ff`, `EV-20261027T164951-6b69`, `EV-20261027T165208-088a`: Binary build sạch lỗi.
  - `EV-20261027T165010-a4dc`, `EV-20261027T165226-52e0`: HTTP POST runtime call tới `:8090/api/tasks/chat`.

## Mục 3: Kiểm tra Concurrency / Timeout-Heartbeat / Data Continuity
- Quản lý concurrency đa luồng: Timeout được thiết lập an toàn `25*time.Second` chống treo socket HTTP.
- Kênh token streaming có `withHeartbeat(ch, defaultHeartbeatInterval)` giữ kết nối liên tục không bị drop ngắt quãng giữa server và client.
- Kiểm tra multi-worker race detector qua `go test -race` đảm bảo an toàn bộ nhớ.

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty)
- Nghiêm cấm tuyệt đối báo cáo láo và suy diễn giả định:
  - Khi runner gặp lỗi thật (như Google API RateLimit `429 Quota Exceeded` trên Free Tier của `gemini` CLI), hệ thống lập tức thông báo lỗi nguyên văn trung thực tới User:
    `⚠️ Lỗi Model Runner (gemini-2.5-flash): TerminalQuotaError: You have exhausted your daily quota on this model.`
  - Không âm thầm nuốt lỗi rồi gán chuỗi tĩnh giả vờ như đã phân tích source code.

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)
- Phản tỉnh sâu sắc qua các bài học `lessons.md`:
  - Tránh triệt để bài học ghép chuỗi tĩnh thay vì nối AI model thật.
  - Loại bỏ hoàn toàn các điều kiện chặn ẩn (`if actionMode != "discuss"`) gây lệch hành vi runtime so với kỳ vọng của User.
  - Duy trì tính minh bạch: Lỗi runner phải được hiển thị chính xác để User nhận biết và điều chỉnh cấu hình API Key.

## Mục 6: Kết quả QC & nhận xét từ /laya_router (DG3) & laya_gate
- **DG1 Gate:** `06_laya_summary_dg1_20261027T164532.json` -> `PASS`.
- **DG3 Gate:** `06_laya_summary_dg3_20261027T165328.json` -> `PASS`.
  - Quyết định: Cả 3 target claims đều đạt PASS/Resolved với REF vật lý trên mã nguồn (`antigravity.go:256-261`, `task_llm_bridge.go:234-247`, `task_chat_knowledge.go:56-62`).
  - Hệ thống `:8090` đã restart sạch sẽ và sẵn sàng hoạt động.
