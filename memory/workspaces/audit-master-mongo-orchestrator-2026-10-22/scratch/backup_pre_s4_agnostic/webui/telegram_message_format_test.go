package webui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/notify"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

func TestTelegramCompactFormat_ForceReplyRouting_AndViewPlanContextSwitch(t *testing.T) {
	// 1. Verify notify.StripBotMentionAnywhere / stripBotUsernameFromCommand strips @tk001sea_bot anywhere
	if got := stripBotUsernameFromCommand("@tk001sea_bot đổi batch size từ 100 lên 500"); got != "đổi batch size từ 100 lên 500" {
		t.Fatalf("expected stripBotUsernameFromCommand to strip leading @tk001sea_bot, got %q", got)
	}
	if got := notify.StripBotMentionAnywhere("/task@tk001sea_bot"); got != "/task" {
		t.Fatalf("expected StripBotMentionAnywhere to strip /task@tk001sea_bot -> /task, got %q", got)
	}
	if !isTelegramApproveCommand("approve @tk001sea_bot") {
		t.Fatalf("expected 'approve @tk001sea_bot' to be recognized as approve command")
	}

	// 2. Verify extractTaskRefFromReplyContext and extractSubTaskRefFromReplyContext on compact cards & ForceReply prompts
	forceReplyPrompt := "✏️ *[CHẾ ĐỘ PHẢN HỒI / SỬA GIẢI PHÁP S3 (#43)]*\n• *Task đang chọn sửa:* `Core-01` (#43) — audit toàn diện"
	if ref := extractTaskRefFromReplyContext(forceReplyPrompt); ref != "Core-01" && ref != "#43" {
		t.Fatalf("expected extractTaskRefFromReplyContext on ForceReply prompt to return Core-01 or #43, got %q", ref)
	}
	subPrompt := "✏️ *[CHẾ ĐỘ SỬA PLAN RIÊNG CHO SUB-TASK 1/6 · `CORE-01-S01` (#43)]*\n• *Task cha:* `Core-01` (#43)"
	if subID := extractSubTaskRefFromReplyContext(subPrompt); subID != "CORE-01-S01" {
		t.Fatalf("expected extractSubTaskRefFromReplyContext to return CORE-01-S01, got %q", subID)
	}

	// 3. Direct SQLite & Server Engine test (zero httptest.NewServer mockup)
	ctx := context.Background()
	srv := newTestWebServer(t)

	for _, tk := range []struct {
		key   string
		title string
	}{
		{key: "Task-Old-01", title: "Task cũ dùng để kiểm tra chuyển ngữ cảnh"},
		{key: "Core-01", title: "audit toàn diện kiến trúc của hệ thống để áp dụng hết các quy tắc của agent core"},
	} {
		id, _, err := srv.QueueStore.EnqueueRaw(ctx, queue.RawTask{
			ExternalID:  tk.key,
			Title:       tk.title,
			Description: tk.title,
			Complexity:  "high",
		})
		if err != nil {
			t.Fatalf("EnqueueRaw %s: %v", tk.key, err)
		}
		if err := srv.QueueStore.UpdateStage(ctx, id, queue.StageNeedsYou, "antigravity", ""); err != nil {
			t.Fatalf("UpdateStage %s: %v", tk.key, err)
		}
	}

	qtCore, err := srv.findQueuedTask(ctx, "Core-01")
	if err != nil {
		t.Fatalf("findQueuedTask Core-01: %v", err)
	}
	isoCore, _ := srv.ensureTaskIsolation(ctx, *qtCore)
	subsCore := srv.loadOrGenerateSubTasks(*qtCore)

	// Verify compact approval card is a single chunk (<= 3,000 runes) without UDF-v1 boilerplate or raw pipe tables
	cardText, _ := srv.buildTelegramApprovalRequiredMessage(ctx, *qtCore, isoCore, subsCore, "S3 Plan Ready")
	if len([]rune(cardText)) > 3000 {
		t.Fatalf("expected compact approval card to be <= 3000 runes (1 single Telegram card), got %d runes:\n%s", len([]rune(cardText)), cardText)
	}
	if chunks := splitTelegramTextIntoSafeChunks(cardText, 3500); len(chunks) != 1 {
		t.Fatalf("expected compact approval card to never split into multiple chunks, got %d chunks", len(chunks))
	}
	if strings.Contains(cardText, "| :--- |") || strings.Contains(cardText, "/Users/") {
		t.Fatalf("expected compact approval card to strip raw pipe tables and /Users/ paths, got:\n%s", cardText)
	}

	// Simulate user clicking REVISE_PROMPT on Task-Old-01 first, then clicking VIEW_PLAN on Core-01 (#ID) directly on Server
	qtOld, _ := srv.findQueuedTask(ctx, "Task-Old-01")
	for _, cmdData := range []string{
		fmt.Sprintf("REVISE_PROMPT #%d", qtOld.ID),
		fmt.Sprintf("VIEW_PLAN #%d", qtCore.ID),
	} {
		if _, err := srv.processIncomingTelegramCommandWithReply(ctx, "", cmdData, "Queue card", "-1004390214071", "trainguyen", 0); err != nil {
			t.Fatalf("processIncomingTelegramCommandWithReply(%q): %v", cmdData, err)
		}
	}

	// Now user sends feedback containing the words "giải pháp" and "nguyên nhân"
	fbText := "@tk001sea_bot Sửa lại giải pháp: bổ sung kiểm tra nguyên nhân lỗi H4c trong gate.go và gọn lại các bước Wave"
	triggeredTask, err := srv.processIncomingTelegramCommandWithReply(ctx, "", fbText, "", "-1004390214071", "trainguyen", 0)
	if err != nil {
		t.Fatalf("processIncomingTelegramCommandWithReply feedback: %v", err)
	}
	if triggeredTask != "Core-01" {
		t.Fatalf("expected feedback after VIEW_PLAN Core-01 to revise Core-01 (not stale Task-Old-01), got %q", triggeredTask)
	}
	rawCorePlan, err := os.ReadFile(isoCore.PlanFile)
	if err != nil || !strings.Contains(string(rawCorePlan), "bổ sung kiểm tra nguyên nhân lỗi H4c") {
		t.Fatalf("expected Core-01 09_tasks_solution.md to contain updated feedback, err=%v", err)
	}

	// 4. Verify formatTelegramCoArchitectReply strips raw pipe tables and stays compact
	sampleUDF := "## 1. Bối cảnh & Điểm chạm (`file:line`)\n| Hạng mục | Chi tiết |\n| :--- | :--- |\n| **Điểm chạm** | `internal/agentgate/gate.go:118` |\n## 2. Nguyên nhân gốc rễ (Root Cause)\n- Thiếu đồng bộ trạng thái poller.\n"
	formattedReply := formatTelegramCoArchitectReply(*qtCore, sampleUDF)
	if strings.Contains(formattedReply, "| :--- |") || !strings.Contains(formattedReply, "gate.go:118") {
		t.Fatalf("expected formatTelegramCoArchitectReply to convert pipe table into clean bullets, got:\n%s", formattedReply)
	}

	// 5. Verify normalizeAndDeduplicateRepoRelativePaths deduplicates 8 mixed paths into 6 unique relative paths
	rawMixedPaths := []string{
		"centralized-export-service/models/trans-his/trans-his.model.ts:10",
		"centralized-export-service/dbhandler/postgres.ts:6",
		"start.ts",
		"svc-env.ts",
		"defined/app-setting.ts",
		"dbHandler/postgres.ts",
		"models/trans-his/trans-his.model.ts",
		"test/unit/models/trans-his.model.test.ts",
	}
	dedup6 := normalizeAndDeduplicateRepoRelativePaths(rawMixedPaths, "centralized-export-service")
	if len(dedup6) != 6 {
		t.Fatalf("expected 6 deduplicated relative files, got %d: %v", len(dedup6), dedup6)
	}
	for _, f := range dedup6 {
		if strings.HasPrefix(f, "centralized-export-service/") {
			t.Fatalf("expected repo prefix to be stripped, got %q", f)
		}
	}

	// 6. Verify real disk inspection on /Users/trainguyen/Documents/work/data-hub/centralized-export-service (if present)
	realExportRepo := "/Users/trainguyen/Documents/work/data-hub/centralized-export-service"
	if st, statErr := os.Stat(realExportRepo); statErr == nil && st.IsDir() {
		profile := detectRepoTechStackProfile(realExportRepo, dedup6)
		if profile.CodeFence != "typescript" || profile.Label != "Node.js / TypeScript" {
			t.Fatalf("expected real centralized-export-service profile to be Node.js / TypeScript, got %+v", profile)
		}
	}

	// 7. Verify notify.ConvertTelegramMarkdownToHTML renders bold, italic, inline code, and typescript code block cleanly
	htmlOut := notify.ConvertTelegramMarkdownToHTML("⏸ **[H4b GATE]** — `TASK-13` *(Node.js / TypeScript)*\n```typescript\nexport class TransHisModel {}\n```")
	if !strings.Contains(htmlOut, "<b>[H4b GATE]</b>") || !strings.Contains(htmlOut, "<i>(Node.js / TypeScript)</i>") || !strings.Contains(htmlOut, "<pre><code class=\"language-typescript\">") {
		t.Fatalf("unexpected HTML conversion output:\n%s", htmlOut)
	}
}
