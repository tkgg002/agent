package webui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/agentgate"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/config"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/memory"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/scout"
)

func newTestWebServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	qStore, err := queue.Open(filepath.Join(tmpDir, "queue.db"))
	if err != nil {
		t.Fatalf("open queue store: %v", err)
	}
	t.Cleanup(func() { _ = qStore.Close() })

	memSys, err := memory.OpenMemory(context.Background(), filepath.Join(tmpDir, "memory"), filepath.Join(tmpDir, "memory.db"), memory.Options{})
	if err != nil {
		t.Fatalf("open memory store: %v", err)
	}
	t.Cleanup(func() { _ = memSys.Close() })

	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gk := agentgate.New("/Users/trainguyen/Documents/work/agent")
	return NewServer(tmpDir, "feature-modular-agent-orchestrator-2026-10-20", "feat/modular-queue-agent-kernel", cfg, qStore, memSys.Store, gk, 258)
}

func TestWebUIServer_StaticIndexAndThreeScreensDashboard(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify GET / returns embedded index.html with all 3 screen containers + Cross-Screen Context Bar, and 0 mockup strings.
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /, got %d", res.StatusCode)
	}
	var htmlBuf bytes.Buffer
	_, _ = htmlBuf.ReadFrom(res.Body)
	html := htmlBuf.String()
	for _, marker := range []string{"screen-wave", "screen-autopilot", "screen-vibe", "cross-screen-context-bar", "wave-svg-overlay", "Vibe System Agent"} {
		if !strings.Contains(html, marker) {
			t.Fatalf("expected index.html to contain %q", marker)
		}
	}
	for _, forbiddenMock := range []string{"176 of 176", "16 claude runs · 64 min · $17.62", "tiem-my", "Cửa hàng thú", "selftest"} {
		if strings.Contains(html, forbiddenMock) {
			t.Fatalf("expected index.html to be free of mockup string %q", forbiddenMock)
		}
	}

	// 1b. Verify GET /app.css defines full-height flex .sri-pre-block / #sri-markdown-content and scrollable .cas-messages-list.
	cssRes, err := http.Get(ts.URL + "/app.css")
	if err != nil {
		t.Fatalf("GET /app.css: %v", err)
	}
	defer cssRes.Body.Close()
	var cssBuf bytes.Buffer
	_, _ = cssBuf.ReadFrom(cssRes.Body)
	cssStr := cssBuf.String()
	for _, cssMarker := range []string{".sri-pre-block, #sri-markdown-content", "flex: 1 1 auto;", ".cas-messages-list, .cas-messages-box, #co-architect-chat-messages", ".cas-msg-card.user-msg", ".cas-msg-card.agent-msg"} {
		if !strings.Contains(cssStr, cssMarker) {
			t.Fatalf("expected /app.css to contain %q", cssMarker)
		}
	}

	// 2. Verify GET /api/dashboard returns 100% real workspace state and cross-screen entity graph linkage.
	dashRes, err := http.Get(ts.URL + "/api/dashboard")
	if err != nil {
		t.Fatalf("GET /api/dashboard: %v", err)
	}
	defer dashRes.Body.Close()
	if dashRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /api/dashboard, got %d", dashRes.StatusCode)
	}

	var rawDashBuf bytes.Buffer
	_, _ = rawDashBuf.ReadFrom(dashRes.Body)
	rawDashJSON := rawDashBuf.String()

	for _, forbiddenMock := range []string{"PM-02", "SRV-01", "airship#31030", "tiem-my", "Cửa hàng thú", "selftest", `"T11"`, `"T30"`, `"T99"`} {
		if strings.Contains(rawDashJSON, forbiddenMock) {
			t.Fatalf("expected /api/dashboard JSON to be free of mockup string %q", forbiddenMock)
		}
	}

	var payload DashboardResponse
	if err := json.Unmarshal(rawDashBuf.Bytes(), &payload); err != nil {
		t.Fatalf("decode dashboard response: %v", err)
	}

	// Screen 1: Real TASK-1..TASK-7 synced from 08_tasks_modular_agent_orchestrator.md
	if len(payload.Waves) < 6 {
		t.Fatalf("expected at least 6 wave columns (Wave 0..5) for TASK-1..7, got %d", len(payload.Waves))
	}
	sumWaveNodes := 0
	foundTasks := make(map[string]WaveNodeDTO)
	for _, w := range payload.Waves {
		if w.TotalCount != len(w.Nodes) {
			t.Fatalf("expected unclamped Wave %d total_count == len(nodes) (%d), got %d", w.Wave, len(w.Nodes), w.TotalCount)
		}
		sumWaveNodes += len(w.Nodes)
		for _, n := range w.Nodes {
			foundTasks[n.Key] = n
			if n.AgentSlot == "" || n.WorkflowStepNum < 1 || n.WorkflowStepNum > 17 {
				t.Fatalf("expected cross-screen linkage (AgentSlot & WorkflowStepNum 1..17) on node %s, got slot=%q step=%d", n.Key, n.AgentSlot, n.WorkflowStepNum)
			}
		}
	}
	if payload.TotalItems != sumWaveNodes {
		t.Fatalf("expected unclamped TotalItems == sumWaveNodes (%d), got %d", sumWaveNodes, payload.TotalItems)
	}
	for _, wantKey := range []string{"TASK-1", "TASK-2", "TASK-3", "TASK-4", "TASK-5", "TASK-6", "TASK-7"} {
		if _, ok := foundTasks[wantKey]; !ok {
			t.Fatalf("expected real workspace task %s in Wave DAG, got keys: %v", wantKey, foundTasks)
		}
	}

	// Check real critical dependency edge TASK-6 -> TASK-7
	hasTaskEdge := false
	for _, e := range payload.Edges {
		if e.From == "TASK-6" && e.To == "TASK-7" && e.Critical {
			hasTaskEdge = true
			break
		}
	}
	if !hasTaskEdge {
		t.Fatalf("expected real critical edge TASK-6 -> TASK-7 in edges: %+v", payload.Edges)
	}

	// Screen 2: Autopilot Pipeline & Cycle/Round Cost ($) Ledger from real Laya summaries
	if len(payload.AutopilotStages) != 7 {
		t.Fatalf("expected 7 Autopilot pipeline stages (including critique DG0), got %d", len(payload.AutopilotStages))
	}
	if len(payload.AutopilotCycles) < 1 {
		t.Fatalf("expected at least 1 Autopilot cycle section, got %d", len(payload.AutopilotCycles))
	}
	if payload.TotalRuns < 1 {
		t.Fatalf("expected real autopilot rounds synced from workspace 06_laya_summary_*.json, got runs=%d", payload.TotalRuns)
	}

	// Screen 3: Vibe System Agent Studio (17 real workflow steps + 11 multi-agent panes linked to TASK-1..7)
	if len(payload.WorkflowSteps) != 17 {
		t.Fatalf("expected 17 workflow steps, got %d", len(payload.WorkflowSteps))
	}
	if payload.WorkflowSteps[0].Title != "Tổng quan" || strings.HasPrefix(payload.WorkflowSteps[0].Title, "1.") {
		t.Fatalf("expected step 1 title 'Tổng quan' without duplicate numeric prefix, got %+v", payload.WorkflowSteps[0])
	}
	if len(payload.AgentPanes) != 11 {
		t.Fatalf("expected 11 agent panes (A..K), got %d", len(payload.AgentPanes))
	}
	for _, p := range payload.AgentPanes {
		if p.ActiveTaskKey == "" {
			t.Fatalf("expected pane %s to have cross-screen ActiveTaskKey populated", p.Slot)
		}
	}
}

func TestWebUIServer_CreateTaskTriggerCycleAndAgentMessage(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Verify baseline dashboard builds cleanly before creating task
	if _, err := srv.BuildDashboardResponse(context.Background()); err != nil {
		t.Fatalf("BuildDashboardResponse baseline: %v", err)
	}

	// 1. POST /api/tasks with secret token in description (must be masked by Rule #20)
	taskBody := map[string]any{
		"external_id": "TASK-8",
		"title":       "Cross-Screen Telemetry Stream with ghp_1234567890abcdefghijklmnopqrstuv",
		"description": "Connect TASK-7 cross-screen events to SSE",
		"milestone":   "M2",
		"epic":        "UX",
		"assignee":    "Coder",
		"complexity":  "high",
		"depends_on":  []string{"TASK-7"},
		"critical":    true,
	}
	rawJSON, _ := json.Marshal(taskBody)
	resp, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(rawJSON))
	if err != nil {
		t.Fatalf("POST /api/tasks: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/tasks, got %d", resp.StatusCode)
	}
	var afterCreate DashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&afterCreate); err != nil {
		t.Fatalf("decode afterCreate: %v", err)
	}

	foundTask8 := false
	for _, w := range afterCreate.Waves {
		for _, n := range w.Nodes {
			if n.Key == "TASK-8" {
				foundTask8 = true
				if strings.Contains(n.Title, "ghp_1234567890") {
					t.Fatalf("expected secret token to be redacted in TASK-8 title, got %q", n.Title)
				}
				if n.Model != "opus" {
					t.Fatalf("expected high complexity task TASK-8 to route to opus, got %q", n.Model)
				}
				if n.AgentSlot != "B" {
					t.Fatalf("expected Coder task TASK-8 to link to AgentSlot B, got %q", n.AgentSlot)
				}
			}
		}
	}
	if !foundTask8 {
		t.Fatalf("expected newly created task TASK-8 in Wave graph")
	}

	// 2. POST /api/pipeline/trigger runs Phase 1 (S1-S3 Plan) on TASK-8 & increments TASK-8's Autopilot rounds dynamically
	runsAfterCreate := afterCreate.TotalRuns
	trigResp, err := http.Post(ts.URL+"/api/pipeline/trigger", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST /api/pipeline/trigger: %v", err)
	}
	defer trigResp.Body.Close()
	if trigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/pipeline/trigger, got %d", trigResp.StatusCode)
	}
	var afterTrigger DashboardResponse
	if err := json.NewDecoder(trigResp.Body).Decode(&afterTrigger); err != nil {
		t.Fatalf("decode afterTrigger: %v", err)
	}
	if afterTrigger.TotalRuns <= runsAfterCreate {
		t.Fatalf("expected TASK-8 Autopilot TotalRuns to increment after trigger, got afterCreate=%d afterTrigger=%d", runsAfterCreate, afterTrigger.TotalRuns)
	}
	if afterTrigger.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected Phase 1 trigger on TASK-8 to stop at NEEDS_YOU awaiting user APPROVE, got %q", afterTrigger.ActiveTaskStage)
	}

	// 3. POST /api/agents/message blocks forbidden command (Hook H5 git push)
	badMsg, _ := json.Marshal(map[string]string{
		"slot":    "C",
		"message": "git push origin main",
	})
	badResp, err := http.Post(ts.URL+"/api/agents/message", "application/json", bytes.NewReader(badMsg))
	if err != nil {
		t.Fatalf("POST /api/agents/message (forbidden): %v", err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for git push, got %d", badResp.StatusCode)
	}

	// 4. POST /api/agents/message on Slot C executes REAL shell command and captures stdout
	goodMsg, _ := json.Marshal(map[string]string{
		"slot":     "C",
		"message":  "echo REAL_SHELL_EXEC_VERIFIED",
		"model":    "zsh",
		"task_key": "TASK-7",
	})
	goodResp, err := http.Post(ts.URL+"/api/agents/message", "application/json", bytes.NewReader(goodMsg))
	if err != nil {
		t.Fatalf("POST /api/agents/message (valid): %v", err)
	}
	defer goodResp.Body.Close()
	if goodResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for safe agent message, got %d", goodResp.StatusCode)
	}
	var afterCmd DashboardResponse
	if err := json.NewDecoder(goodResp.Body).Decode(&afterCmd); err != nil {
		t.Fatalf("decode afterCmd: %v", err)
	}
	foundShellOut := false
	for _, p := range afterCmd.AgentPanes {
		if p.Slot == "C" {
			for _, ln := range p.Lines {
				if strings.Contains(ln, "REAL_SHELL_EXEC_VERIFIED") {
					foundShellOut = true
					break
				}
			}
		}
	}
	if !foundShellOut {
		t.Fatalf("expected real shell command stdout REAL_SHELL_EXEC_VERIFIED in Slot C pane lines")
	}
}

func TestWebUIServer_SSEStream(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/stream", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/stream: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read sse line 1: %v", err)
	}
	if !strings.Contains(line1, "event: connected") {
		t.Fatalf("expected event: connected, got %q", line1)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		srv.broadcast("test_update")
	}()

	foundUpdate := false
	for i := 0; i < 6; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, "test_update") {
			foundUpdate = true
			break
		}
	}
	if !foundUpdate {
		t.Fatalf("expected SSE stream to receive broadcasted test_update event")
	}
}

func TestWebUIServer_PerTaskIsolationAndLifecycleAutopilot(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create two separate tasks (TASK-ISO-A and TASK-ISO-B) without auto_autopilot first.
	for _, tk := range []string{"TASK-ISO-A", "TASK-ISO-B"} {
		body, _ := json.Marshal(map[string]any{
			"external_id":    tk,
			"title":          "tích hợp Master MongoDB Destination for " + tk,
			"description":    "Verify 2-phase Gate-Stop Autopilot & 4-layer physical isolation for " + tk,
			"target_repo":    "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
			"milestone":      "M2",
			"epic":           "DATA",
			"assignee":       "Coder",
			"complexity":     "high",
			"runner":         "antigravity",
			"auto_autopilot": false,
		})
		res, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("create %s: %v", tk, err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("create %s status=%d", tk, res.StatusCode)
		}
	}

	// 2. Focus TASK-ISO-A via GET /api/dashboard?task_key=TASK-ISO-A and verify fresh non-DONE Autopilot stages,
	//    TargetRepo, 4 Sub-Tasks, 5 Step Reports (with Concrete Code Demo), and 268 /agent lessons.
	resA, err := http.Get(ts.URL + "/api/dashboard?task_key=TASK-ISO-A")
	if err != nil {
		t.Fatalf("GET dashboard TASK-ISO-A: %v", err)
	}
	defer resA.Body.Close()
	var dashA DashboardResponse
	if err := json.NewDecoder(resA.Body).Decode(&dashA); err != nil {
		t.Fatalf("decode dashA: %v", err)
	}
	if dashA.ActiveTaskKey != "TASK-ISO-A" {
		t.Fatalf("expected ActiveTaskKey=TASK-ISO-A, got %q", dashA.ActiveTaskKey)
	}
	if !dashA.KernelEnforced {
		t.Fatalf("expected KernelEnforced=true when /agent Gatekeeper is attached")
	}
	if dashA.ActiveAgentKernel.TotalLessons < 268 {
		t.Fatalf("expected ActiveAgentKernel.TotalLessons >= 268 from lessons/INDEX.md, got %d", dashA.ActiveAgentKernel.TotalLessons)
	}
	if len(dashA.ActiveAgentKernel.MatchedLessons) < 3 {
		t.Fatalf("expected >= 3 matched /agent lessons for MongoDB task, got %+v", dashA.ActiveAgentKernel.MatchedLessons)
	}
	if len(dashA.ActiveSubTasks) < 4 {
		t.Fatalf("expected >= 4 decomposed sub-tasks for TASK-ISO-A, got %d", len(dashA.ActiveSubTasks))
	}
	if len(dashA.ActiveStepReports) != 6 {
		t.Fatalf("expected 6 Step Reports (S1, S2, DG0, S3, S4, S5) for TASK-ISO-A, got %d", len(dashA.ActiveStepReports))
	}
	hasCodeDemo := false
	for _, rep := range dashA.ActiveStepReports {
		if rep.StepID == "S3" && (strings.Contains(rep.CodeDemo, "bulkUpsertMasterMongo") || strings.Contains(rep.CodeDemo, "GetMasterMongoClient") || strings.Contains(rep.CodeDemo, "UpsertMasterBatch")) {
			hasCodeDemo = true
		}
	}
	if !hasCodeDemo {
		t.Fatalf("expected S3 StepReport for MongoDB task to contain concrete CodeDemo with bulkUpsertMasterMongo/GetMasterMongoClient")
	}

	// Verify Plan stage starts as "waiting" and Implement stage starts as "pending" for a newly created task!
	for _, stg := range dashA.AutopilotStages {
		if stg.ID == "plan" && len(stg.Cards) > 0 && stg.Cards[0].Status != "waiting" {
			t.Fatalf("expected newly created TASK-ISO-A Plan stage status='waiting', got %q", stg.Cards[0].Status)
		}
		if stg.ID == "implement" && len(stg.Cards) > 0 && stg.Cards[0].Status != "pending" {
			t.Fatalf("expected newly created TASK-ISO-A Implement stage status='pending', got %q", stg.Cards[0].Status)
		}
	}

	// 3. WF-01: Calling action="autopilot" on a READY task MUST run Phase 1 (S1-S3) and HARD-STOP at NEEDS_YOU (never auto-DONE!).
	autoPhase1Req, _ := json.Marshal(map[string]string{
		"task_key": "TASK-ISO-A",
		"action":   "autopilot",
		"runner":   "antigravity",
	})
	autoPhase1Res, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(autoPhase1Req))
	if err != nil {
		t.Fatalf("POST /api/tasks/action autopilot phase 1: %v", err)
	}
	defer autoPhase1Res.Body.Close()
	var afterPhase1 DashboardResponse
	if err := json.NewDecoder(autoPhase1Res.Body).Decode(&afterPhase1); err != nil {
		t.Fatalf("decode afterPhase1: %v", err)
	}
	if afterPhase1.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected WF-01 2-Phase Autopilot Phase 1 to HARD-STOP at NEEDS_YOU awaiting user APPROVE, got %q", afterPhase1.ActiveTaskStage)
	}

	// Verify TASK-ISO-B is completely untouched (still READY, not NEEDS_YOU or DONE).
	for _, w := range afterPhase1.Waves {
		for _, n := range w.Nodes {
			if n.Key == "TASK-ISO-B" {
				if n.Stage != "READY" {
					t.Fatalf("expected isolated TASK-ISO-B to remain READY while TASK-ISO-A is in NEEDS_YOU, got %q", n.Stage)
				}
				if n.Isolation.WorktreePath == afterPhase1.ActiveIsolation.WorktreePath {
					t.Fatalf("expected distinct WorktreePath per task, got identical %q", n.Isolation.WorktreePath)
				}
			}
		}
	}

	// 4a. SA-01 & SA-02 Negative Gate Check: Even though 09_tasks_solution.md exists for TASK-ISO-A,
	// calling approve_and_run BEFORE any diff.patch or worktree code exists MUST hard-block at NEEDS_YOU (0% Paper-Verification)!
	approveBlockReq, _ := json.Marshal(map[string]string{
		"task_key": "TASK-ISO-A",
		"action":   "approve_and_run",
		"runner":   "antigravity",
	})
	approveBlockRes, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(approveBlockReq))
	if err != nil {
		t.Fatalf("POST /api/tasks/action approve_and_run (negative gate check): %v", err)
	}
	defer approveBlockRes.Body.Close()
	var afterBlocked DashboardResponse
	if err := json.NewDecoder(approveBlockRes.Body).Decode(&afterBlocked); err != nil {
		t.Fatalf("decode afterBlocked: %v", err)
	}
	if afterBlocked.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("SA-01 violation: expected approve_and_run without diff/code on disk to HARD-BLOCK at NEEDS_YOU even when 09_tasks_solution.md exists, got %q", afterBlocked.ActiveTaskStage)
	}

	// 4b. Materialize complete unified diff.patch inside TASK-ISO-A's isolated artifact dir (in t.TempDir(), 0% host path leak)
	// and verify approve_and_run now succeeds -> DONE.
	isoADiffPath := filepath.Join(srv.Root, ".orchestrator", "tasks", "TASK-ISO-A", "diff.patch")
	_ = os.WriteFile(isoADiffPath, []byte(sampleCompleteMongoUnifiedDiffPatch()), 0o644)

	approveRunReq, _ := json.Marshal(map[string]string{
		"task_key": "TASK-ISO-A",
		"action":   "approve_and_run",
		"runner":   "antigravity",
	})
	approveRunRes, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(approveRunReq))
	if err != nil {
		t.Fatalf("POST /api/tasks/action approve_and_run: %v", err)
	}
	defer approveRunRes.Body.Close()
	if approveRunRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for action=approve_and_run, got %d", approveRunRes.StatusCode)
	}
	var afterApproveRun DashboardResponse
	if err := json.NewDecoder(approveRunRes.Body).Decode(&afterApproveRun); err != nil {
		t.Fatalf("decode afterApproveRun: %v", err)
	}
	if afterApproveRun.ActiveTaskStage != "DONE" {
		t.Fatalf("expected TASK-ISO-A stage=DONE after approve_and_run with verified diff.patch, got %q", afterApproveRun.ActiveTaskStage)
	}
	if len(afterApproveRun.ActiveTaskLogs) < 3 {
		t.Fatalf("expected per-task execution logs in ActiveTaskLogs, got %v", afterApproveRun.ActiveTaskLogs)
	}

	// 5. SCOPE-01: Switching to TASK-ISO-B via GET /api/dashboard?task_key=TASK-ISO-B must scope 100% of right window to TASK-ISO-B.
	resB, err := http.Get(ts.URL + "/api/dashboard?task_key=TASK-ISO-B")
	if err != nil {
		t.Fatalf("GET dashboard TASK-ISO-B: %v", err)
	}
	defer resB.Body.Close()
	var dashB DashboardResponse
	if err := json.NewDecoder(resB.Body).Decode(&dashB); err != nil {
		t.Fatalf("decode dashB: %v", err)
	}
	if dashB.ActiveTaskKey != "TASK-ISO-B" || dashB.ActiveTaskStage != "READY" {
		t.Fatalf("expected strict scoping to TASK-ISO-B (READY), got key=%q stage=%q", dashB.ActiveTaskKey, dashB.ActiveTaskStage)
	}
	for _, cyc := range dashB.AutopilotCycles {
		for _, r := range cyc.Rows {
			if r.TaskKey != "TASK-ISO-B" {
				t.Fatalf("SCOPE-01 violation: found round row for %q inside TASK-ISO-B's AutopilotCycles", r.TaskKey)
			}
		}
	}
	for _, p := range dashB.AgentPanes {
		if p.ActiveTaskKey != "TASK-ISO-B" {
			t.Fatalf("SCOPE-01 violation: expected pane %s ActiveTaskKey=TASK-ISO-B, got %q", p.Slot, p.ActiveTaskKey)
		}
	}

	// 6. REPO-SUB-01: Test POST /api/settings/cli and POST /api/tasks/subtasks.
	cliReq, _ := json.Marshal(map[string]any{
		"default_runner":   "gemini",
		"planner_model":    "gemini-2.5-pro",
		"coder_high_model": "claude-opus-4-6",
		"coder_std_model":  "claude-sonnet-4-6",
	})
	cliRes, err := http.Post(ts.URL+"/api/settings/cli", "application/json", bytes.NewReader(cliReq))
	if err != nil {
		t.Fatalf("POST /api/settings/cli: %v", err)
	}
	defer cliRes.Body.Close()
	if cliRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/settings/cli, got %d", cliRes.StatusCode)
	}
	var afterCLI DashboardResponse
	if err := json.NewDecoder(cliRes.Body).Decode(&afterCLI); err != nil {
		t.Fatalf("decode afterCLI: %v", err)
	}
	if afterCLI.CLISettings.DefaultRunner != "gemini" || afterCLI.CLISettings.PlannerModel != "gemini-2.5-pro" {
		t.Fatalf("expected persisted CLISettings (gemini / gemini-2.5-pro), got %+v", afterCLI.CLISettings)
	}
}

func TestTask10_Gemini4ArgonAndMultiRepoDomainSubTasks(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify default CLISettings uses gemini-4-argon (antigravity) and DetectedRunners includes antigravity/gemini-4-argon.
	initRes, err := http.Get(ts.URL + "/api/dashboard")
	if err != nil {
		t.Fatalf("GET /api/dashboard: %v", err)
	}
	defer initRes.Body.Close()
	var initDash DashboardResponse
	if err := json.NewDecoder(initRes.Body).Decode(&initDash); err != nil {
		t.Fatalf("decode initDash: %v", err)
	}
	if initDash.CLISettings.PlannerModel != "gemini-4-argon (antigravity)" {
		t.Fatalf("expected default PlannerModel='gemini-4-argon (antigravity)', got %q", initDash.CLISettings.PlannerModel)
	}
	if initDash.CLISettings.CoderHighModel != "gemini-4-argon (antigravity)" {
		t.Fatalf("expected default CoderHighModel='gemini-4-argon (antigravity)', got %q", initDash.CLISettings.CoderHighModel)
	}
	hasArgonRunner := false
	for _, dr := range initDash.AvailableRunners {
		for _, m := range dr.Models {
			if strings.Contains(m, "gemini-4-argon") {
				hasArgonRunner = true
				break
			}
		}
	}
	if !hasArgonRunner {
		t.Fatalf("expected AvailableRunners to contain 'gemini-4-argon', got %+v", initDash.AvailableRunners)
	}

	// 2. Create "tích hợp Master MongoDB Destination" task and verify 3 AffectedRepos + 10 Domain Sub-Tasks (TASK-M01..TASK-M10) across Wave 0..4.
	createBody, _ := json.Marshal(map[string]any{
		"external_id":    "TASK-MONGO-EPIC",
		"title":          "tích hợp Master MongoDB Destination",
		"description":    "Tích hợp Master MongoDB Destination xuyên 3 repo cdc-cms-service, cdc-cms-web, centralized-data-service",
		"target_repo":    "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos":   []string{"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service", "/Users/trainguyen/Documents/work/data-hub/cdc-cms-web", "/Users/trainguyen/Documents/work/data-hub/centralized-data-service"},
		"milestone":      "M1",
		"epic":           "CORE",
		"assignee":       "Coder",
		"complexity":     "high",
		"runner":         "gemini-4-argon (antigravity)",
		"auto_autopilot": false,
	})
	createRes, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-MONGO-EPIC: %v", err)
	}
	defer createRes.Body.Close()
	if createRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK creating TASK-MONGO-EPIC, got %d", createRes.StatusCode)
	}

	focusRes, err := http.Get(ts.URL + "/api/dashboard?task_key=TASK-MONGO-EPIC")
	if err != nil {
		t.Fatalf("GET /api/dashboard?task_key=TASK-MONGO-EPIC: %v", err)
	}
	defer focusRes.Body.Close()
	var dash DashboardResponse
	if err := json.NewDecoder(focusRes.Body).Decode(&dash); err != nil {
		t.Fatalf("decode dash TASK-MONGO-EPIC: %v", err)
	}
	if len(dash.ActiveAffectedRepos) < 3 {
		t.Fatalf("expected >= 3 ActiveAffectedRepos for TASK-MONGO-EPIC, got %v", dash.ActiveAffectedRepos)
	}
	if len(dash.ActiveSubTasks) != 10 {
		t.Fatalf("expected 10 domain sub-tasks (TASK-M01..TASK-M10) for TASK-MONGO-EPIC, got %d", len(dash.ActiveSubTasks))
	}
	repoCounts := map[string]int{}
	for _, st := range dash.ActiveSubTasks {
		repoCounts[st.TargetRepoName]++
		if st.GroupTitle == "" || st.ContextFileLine == "" || st.TestDoD == "" {
			t.Fatalf("expected SubTask %s to have GroupTitle, ContextFileLine, and TestDoD, got %+v", st.SubTaskID, st)
		}
	}
	if repoCounts["cdc-cms-service"] != 2 || repoCounts["cdc-cms-web"] != 1 || repoCounts["centralized-data-service"] != 7 {
		t.Fatalf("expected per-repo sub-task distribution (cdc-cms-service=2, cdc-cms-web=1, centralized-data-service=7), got %v", repoCounts)
	}

	// 3a. SA-01 & AF-ORCH-04 Negative Check: Before diff.patch is written in srv.Root,
	// unimplemented Sub-Tasks (TASK-M01, TASK-M02, TASK-M03, TASK-M08, TASK-M10) MUST return false even though 09_tasks_solution.md exists!
	for _, st := range dash.ActiveSubTasks {
		if st.SubTaskID == "TASK-M01" || st.SubTaskID == "TASK-M02" || st.SubTaskID == "TASK-M03" || st.SubTaskID == "TASK-M08" || st.SubTaskID == "TASK-M10" {
			if okDisk, ev := srv.VerifySubTaskCodeOnDisk(context.Background(), st); okDisk {
				t.Fatalf("SA-01/AF-ORCH-04 violation: expected %s to fail VerifySubTaskCodeOnDisk before diff.patch exists, but got true (%s)", st.SubTaskID, ev)
			}
		}
	}

	// 3b. Task 11 (REAL-S4-EXEC-01): Write complete unified diff.patch into TASK-MONGO-EPIC's isolated artifact dir
	// and verify that every Sub-Task TASK-M01..TASK-M10 passes real on-disk verification.
	epicDiffPath := filepath.Join(srv.Root, ".orchestrator", "tasks", "TASK-MONGO-EPIC", "diff.patch")
	_ = os.WriteFile(epicDiffPath, []byte(sampleCompleteMongoUnifiedDiffPatch()), 0o644)
	for _, st := range dash.ActiveSubTasks {
		okDisk, ev := srv.VerifySubTaskCodeOnDisk(context.Background(), st)
		if !okDisk {
			t.Fatalf("expected SubTask %s (%s) to pass VerifySubTaskCodeOnDisk with verified diff.patch, got false (%s)", st.SubTaskID, st.ContextFileLine, ev)
		}
	}

	// Negative control: a fake sub-task pointing to a missing file must return false
	okFake, _ := srv.VerifySubTaskCodeOnDisk(context.Background(), SubTaskDTO{
		SubTaskID:       "TASK-FAKE-99",
		TargetRepo:      t.TempDir(),
		ContextFileLine: "nonexistent/file_that_does_not_exist.go:1-10",
	})
	if okFake {
		t.Fatalf("expected fake unimplemented sub-task to fail VerifySubTaskCodeOnDisk")
	}
}

func sampleCompleteMongoUnifiedDiffPatch() string {
	paths := []string{
		"internal/api/master/master_registry_handler_connections.go",
		"internal/repository/master_repo_gorm.go",
		"src/pages/MasterRegistry.tsx",
		"internal/service/source/connection_manager.go",
		"internal/service/master/master_ddl_generator.go",
		"internal/service/master/master_schema_drift_detector.go",
		"internal/service/master/mongo_type_converter.go",
		"internal/service/master/transmuter.go",
		"internal/service/master/transmuter_mongo.go",
		"internal/service/recon/recon_engine.go",
		"internal/service/recon/recon_smoke.go",
		"internal/service/recon/recon_tier_b.go",
		"internal/service/recon/recon_dest_mongo.go",
		"internal/handler/recon/recon_execute_heal_handler.go",
		"test/master_mongo_e2e_test.go",
	}
	var b strings.Builder
	for _, p := range paths {
		b.WriteString("diff --git a/" + p + " b/" + p + "\n--- a/" + p + "\n+++ b/" + p + "\n@@ -1,1 +1,2 @@\n+// verified mongo implementation\n")
	}
	return b.String()
}

func TestLiveTelemetryHUDRoleModelCoArchitectAndDualModeIntake(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create a task via Option 2 (Manual Intake with Title + Detailed Description + BRD Markdown)
	brdDoc := "# BRD: Master MongoDB Destination\n- Dynamic Schema Mapping\n- Ordered BulkWrite OCC ($lt _cdc_version)\n- Multi-worker Watermark Continuity"
	createPayload, _ := json.Marshal(map[string]any{
		"external_id":    "TASK-DUAL-12",
		"title":          "tích hợp Master MongoDB Destination (Dual-Mode & Co-Architect)",
		"description":    "Mô tả chi tiết: Đồng bộ CDC sang MongoDB xuyên 3 repo với OCC _cdc_version và gom lô BatchDebouncer.",
		"intake_mode":    "manual",
		"brd_content":    brdDoc,
		"brd_file_name":  "mongodb_master_brd.md",
		"target_repo":    "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos":   []string{"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service", "/Users/trainguyen/Documents/work/data-hub/cdc-cms-web", "/Users/trainguyen/Documents/work/data-hub/centralized-data-service"},
		"milestone":      "M2",
		"epic":           "DATA",
		"assignee":       "Coder",
		"complexity":     "high",
		"runner":         "antigravity",
		"model":          "gemini-4-argon (antigravity)",
		"auto_autopilot": true,
	})
	resCreate, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createPayload))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-DUAL-12: %v", err)
	}
	defer resCreate.Body.Close()
	if resCreate.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK creating TASK-DUAL-12, got %d", resCreate.StatusCode)
	}
	var dash DashboardResponse
	if err := json.NewDecoder(resCreate.Body).Decode(&dash); err != nil {
		t.Fatalf("decode dashboard response: %v", err)
	}

	// Verify brd_source.md and 01_requirements.md on disk
	taskDir := srv.buildTaskIsolationMeta("TASK-DUAL-12").ArtifactDir
	brdBytes, err := os.ReadFile(filepath.Join(taskDir, "brd_source.md"))
	if err != nil || !strings.Contains(string(brdBytes), "Ordered BulkWrite OCC") {
		t.Fatalf("expected brd_source.md to be persisted with BRD content, err=%v", err)
	}
	reqBytes, err := os.ReadFile(filepath.Join(taskDir, "01_requirements.md"))
	if err != nil || !strings.Contains(string(reqBytes), "mongodb_master_brd.md") || !strings.Contains(string(reqBytes), "Mô tả chi tiết:") {
		t.Fatalf("expected 01_requirements.md to include both detailed description and BRD content, err=%v", err)
	}

	// 2. Verify LIVE-PROCESS-HUD-01 (LiveExecution telemetry & Per-Repo Progress Breakdown)
	if dash.LiveExecution.ActiveTaskKey != "TASK-DUAL-12" {
		t.Fatalf("expected LiveExecution.ActiveTaskKey=TASK-DUAL-12, got %q", dash.LiveExecution.ActiveTaskKey)
	}
	if dash.LiveExecution.ActiveStepNum <= 0 || dash.LiveExecution.ActiveRole == "" || dash.LiveExecution.ActiveModel == "" {
		t.Fatalf("expected populated LiveExecution HUD telemetry, got %+v", dash.LiveExecution)
	}
	if len(dash.LiveExecution.RepoProgress) < 3 {
		t.Fatalf("expected >= 3 RepoProgress entries in LiveExecution HUD, got %+v", dash.LiveExecution.RepoProgress)
	}

	// 3. Verify ROLE-MODEL-ATTRIBUTION-01 across 17 WorkflowSteps, 6 AutopilotStages, and 10 SubTasks
	if len(dash.WorkflowSteps) != 17 {
		t.Fatalf("expected 17 workflow steps, got %d", len(dash.WorkflowSteps))
	}
	for _, wf := range dash.WorkflowSteps {
		if wf.RoleLabel == "" || wf.AgentSlot == "" || wf.SkillBadge == "" || wf.Model == "" {
			t.Fatalf("workflow step %d missing Role/Slot/Skill/Model attribution: %+v", wf.Num, wf)
		}
	}
	for _, stg := range dash.AutopilotStages {
		if len(stg.Cards) == 0 || stg.Cards[0].RoleLabel == "" || stg.Cards[0].AgentSlot == "" || stg.Cards[0].SkillBadge == "" || stg.Cards[0].Model == "" {
			t.Fatalf("autopilot stage %s missing Role/Slot/Skill/Model attribution: %+v", stg.ID, stg)
		}
	}
	for _, st := range dash.ActiveSubTasks {
		if st.RoleLabel == "" || st.AgentSlot == "" || st.SkillBadge == "" || st.Model == "" {
			t.Fatalf("sub-task %s missing Role/Slot/Skill/Model attribution: %+v", st.SubTaskID, st)
		}
	}

	// Update a Sub-Task's Role & Model via POST /api/tasks/subtasks (action: "update_role_model")
	updRoleBody, _ := json.Marshal(map[string]any{
		"task_key":    "TASK-DUAL-12",
		"action":      "update_role_model",
		"sub_task_id": "TASK-M03",
		"assignee":    "Reviewer",
		"model":       "claude-opus-4-6",
	})
	resRole, err := http.Post(ts.URL+"/api/tasks/subtasks", "application/json", bytes.NewReader(updRoleBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/subtasks update_role_model: %v", err)
	}
	defer resRole.Body.Close()
	if resRole.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK updating sub-task role/model, got %d", resRole.StatusCode)
	}
	var dashAfterRole DashboardResponse
	if err := json.NewDecoder(resRole.Body).Decode(&dashAfterRole); err != nil {
		t.Fatalf("decode dashAfterRole: %v", err)
	}
	foundUpdatedM03 := false
	for _, st := range dashAfterRole.ActiveSubTasks {
		if st.SubTaskID == "TASK-M03" {
			foundUpdatedM03 = true
			if st.Model != "claude-opus-4-6" || st.AgentSlot != "Slot H" {
				t.Fatalf("expected TASK-M03 updated to Model=claude-opus-4-6 and AgentSlot=Slot H, got %+v", st)
			}
		}
	}
	if !foundUpdatedM03 {
		t.Fatalf("TASK-M03 not found after update_role_model")
	}

	// 4. Verify CO-ARCHITECT-CHAT-01 (Interactive Solution Steering & Revision before APPROVE)
	chatReviseBody, _ := json.Marshal(map[string]any{
		"task_key":    "TASK-DUAL-12",
		"role_target": "BRAIN",
		"model":       "gemini-4-argon-thinking (antigravity)",
		"action":      "revise_solution",
		"message":     "Bổ sung cơ chế Dead-Letter Queue (DLQ) khi MongoBulkWriteException vượt quá 3 lần retry với backoff.",
	})
	resChat, err := http.Post(ts.URL+"/api/tasks/chat", "application/json", bytes.NewReader(chatReviseBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/chat revise_solution: %v", err)
	}
	defer resChat.Body.Close()
	if resChat.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/tasks/chat, got %d", resChat.StatusCode)
	}
	var dashAfterChat DashboardResponse
	if err := json.NewDecoder(resChat.Body).Decode(&dashAfterChat); err != nil {
		t.Fatalf("decode dashAfterChat: %v", err)
	}
	if len(dashAfterChat.ActiveTaskChat) < 3 {
		t.Fatalf("expected >= 3 messages in ActiveTaskChat after steering turn, got %d", len(dashAfterChat.ActiveTaskChat))
	}
	solBytes, err := os.ReadFile(filepath.Join(taskDir, "09_tasks_solution.md"))
	if err != nil || !strings.Contains(string(solBytes), "Dead-Letter Queue (DLQ)") {
		t.Fatalf("expected 09_tasks_solution.md to contain revised Co-Architect steering section, err=%v", err)
	}

	// Negative security control on /api/tasks/chat: forbidden git push must return 403
	badChatBody, _ := json.Marshal(map[string]any{
		"task_key": "TASK-DUAL-12",
		"action":   "discuss",
		"message":  "chạy git push origin main ngay",
	})
	resBadChat, err := http.Post(ts.URL+"/api/tasks/chat", "application/json", bytes.NewReader(badChatBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/chat bad command: %v", err)
	}
	defer resBadChat.Body.Close()
	if resBadChat.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when sending git push to /api/tasks/chat, got %d", resBadChat.StatusCode)
	}

	// 5. Verify Option 1 of TASK-INTAKE-DUAL-MODE-01 (Multi-Account Tracker Auth + Token Masking + Issue Browser)
	newTrackerBody, _ := json.Marshal(map[string]any{
		"provider":    "jira",
		"name":        "Jira Cloud — DataHub Enterprise",
		"base_url":    "https://enterprise-datahub.atlassian.net",
		"project_key": "DH",
		"email":       "lead@datahub.local",
		"token":       "ATATT3xFfGF0secretTokenValue987654321",
	})
	resTracker, err := http.Post(ts.URL+"/api/trackers", "application/json", bytes.NewReader(newTrackerBody))
	if err != nil {
		t.Fatalf("POST /api/trackers: %v", err)
	}
	defer resTracker.Body.Close()
	if resTracker.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from POST /api/trackers, got %d", resTracker.StatusCode)
	}
	var trackerResp struct {
		Created     TrackerConnectionDTO   `json:"created"`
		Connections []TrackerConnectionDTO `json:"connections"`
	}
	if err := json.NewDecoder(resTracker.Body).Decode(&trackerResp); err != nil {
		t.Fatalf("decode trackerResp: %v", err)
	}
	if strings.Contains(trackerResp.Created.MaskedToken, "secretTokenValue") || !strings.Contains(trackerResp.Created.MaskedToken, "***") {
		t.Fatalf("expected token to be masked per Rule #20, got %q", trackerResp.Created.MaskedToken)
	}

	// Browse issues from the tracker connection
	resIssues, err := http.Get(ts.URL + "/api/trackers/issues?connection_id=" + trackerResp.Created.ID + "&query=DH-401")
	if err != nil {
		t.Fatalf("GET /api/trackers/issues: %v", err)
	}
	defer resIssues.Body.Close()
	if resIssues.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from GET /api/trackers/issues, got %d", resIssues.StatusCode)
	}
	var issuesResp struct {
		Issues []TrackerIssueDTO `json:"issues"`
	}
	if err := json.NewDecoder(resIssues.Body).Decode(&issuesResp); err != nil {
		t.Fatalf("decode issuesResp: %v", err)
	}
	if len(issuesResp.Issues) == 0 || issuesResp.Issues[0].IssueKey != "DH-401" || issuesResp.Issues[0].BRDMarkdown == "" {
		t.Fatalf("expected issue DH-401 with BRDMarkdown from /api/trackers/issues, got %+v", issuesResp.Issues)
	}

	// 6. Verify AF-ORCH-06 (Auto-Intent Audit Report Ingestion in Co-Architect Chat even when action=="discuss")
	// and AF-ORCH-03 (Never pollute real /agent/memory/workspaces/ files during tests)
	auditPasteBody, _ := json.Marshal(map[string]any{
		"task_key": "TASK-DUAL-12",
		"action":   "discuss",
		"message": "### I. Tổng Hợp Lỗ Hổng Phát Hiện Thực Tế\n" +
			"1. **`AF-ORCH-01 (P0 — Split-Brain Duplicate Task)`**\n" +
			"2. **`AF-ORCH-06 (P0 — Co-Architect Chat Static Template)`**\n" +
			"3. **`AF-DH-01 (P0 — E11000 DuplicateKeyError)`**\n" +
			"### II. Tóm Tắt Phương Án Khắc Phục Duy Nhất (`TASK-A04 .. TASK-A07`)\n" +
			"1. **`TASK-A04` (`centralized-data-service` — `transmuter_mongo.go`):**\n" +
			"   - Đổi BuildMongoOCCBulkModels sang Aggregation Pipeline Update.\n" +
			"4. **`TASK-A07` (`ai-orchestrator` — `task_chat.go` & `task_inspector.go`):**\n" +
			"   - Hợp nhất 2 task trùng tên và tự động bóc tách báo cáo Audit.\n",
	})
	resAuditChat, err := http.Post(ts.URL+"/api/tasks/chat", "application/json", bytes.NewReader(auditPasteBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/chat audit paste: %v", err)
	}
	defer resAuditChat.Body.Close()
	if resAuditChat.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/tasks/chat audit paste, got %d", resAuditChat.StatusCode)
	}
	var dashAfterAudit DashboardResponse
	if err := json.NewDecoder(resAuditChat.Body).Decode(&dashAfterAudit); err != nil {
		t.Fatalf("decode dashAfterAudit: %v", err)
	}
	lastMsg := dashAfterAudit.ActiveTaskChat[len(dashAfterAudit.ActiveTaskChat)-1]
	if !lastMsg.AppliedToPlan || lastMsg.ActionMode != "ingest_audit_remediation" {
		t.Fatalf("expected pasted audit report to auto-trigger ingest_audit_remediation with AppliedToPlan=true, got mode=%q applied=%v", lastMsg.ActionMode, lastMsg.AppliedToPlan)
	}
	if strings.Contains(lastMsg.Message, "):** 1. **`TASK-A") {
		t.Fatalf("SA-04 violation: duplicated markdown prefix found in ingested audit reply: %s", lastMsg.Message)
	}
	if !strings.Contains(lastMsg.Message, "Đối soát Vật lý Trung thực trên Đĩa") {
		t.Fatalf("SA-03 violation: expected honest on-disk verification summary in audit reply, got: %s", lastMsg.Message)
	}
	tasks08Bytes, err := os.ReadFile(filepath.Join(taskDir, "08_tasks.md"))
	if err != nil || !strings.Contains(string(tasks08Bytes), "TASK-A04") || !strings.Contains(string(tasks08Bytes), "TASK-A07") {
		t.Fatalf("expected 08_tasks.md to contain ingested remediation subtasks TASK-A04 and TASK-A07, err=%v", err)
	}
	realWs09 := "/Users/trainguyen/Documents/work/agent/memory/workspaces/feat-master-mongo-destination-2026-10-20/09_tasks_solution_master_mongo.md"
	if rawReal09, err := os.ReadFile(realWs09); err == nil {
		if strings.Contains(string(rawReal09), "TASK-DUAL-12") {
			t.Fatalf("AF-ORCH-03 violation: unit test polluted real /agent workspace file %s with TASK-DUAL-12!", realWs09)
		}
	}
}

func TestWebUIServer_S5AdversarialAuditHook_TelegramBridge_And3TierWaveDAG(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify Topbar hidden badges CSS (TASK-CAR-06), Setup Telegram button/modal (TASK-CAR-05), and 3-Tier Wave DAG HTML (TASK-CAR-07)
	idxRes, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	idxBytes, _ := io.ReadAll(idxRes.Body)
	_ = idxRes.Body.Close()
	idxHTML := string(idxBytes)
	for _, marker := range []string{
		"btn-setup-telegram",
		"telegram-settings-modal",
		"wave-tier-epics",
		"wave-tier-tasks",
		"wave-tier-3-box",
		"wave-svg",
	} {
		if !strings.Contains(idxHTML, marker) {
			t.Fatalf("expected index.html to contain %q", marker)
		}
	}

	cssRes, err := http.Get(ts.URL + "/app.css")
	if err != nil {
		t.Fatalf("GET /app.css: %v", err)
	}
	cssBytes, _ := io.ReadAll(cssRes.Body)
	_ = cssRes.Body.Close()
	if !strings.Contains(string(cssBytes), "display: none !important") {
		t.Fatalf("expected app.css to hide .branch-pill, .kernel-pill with display: none !important")
	}

	// 2. Configure Telegram Bridge (TASK-CAR-05 & TASK-CAR-08) & verify Rule #20 Bot Token Masking + notify_on_s3_plan alias preservation
	tgCfgReq, _ := json.Marshal(map[string]any{
		"enabled":            true,
		"bot_token":          "8899001122:AAHSecretTelegramBotTokenValue999",
		"chat_id":            "-10099887766",
		"bot_username":       "@datahub_orchestrator_bot",
		"notify_on_s3_plan":  true,
		"auto_run_on_approve": true,
		"polling_enabled":    false,
	})
	tgRes, err := http.Post(ts.URL+"/api/settings/telegram", "application/json", bytes.NewReader(tgCfgReq))
	if err != nil {
		t.Fatalf("POST /api/settings/telegram: %v", err)
	}
	defer tgRes.Body.Close()
	if tgRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/settings/telegram, got %d", tgRes.StatusCode)
	}
	var dashAfterTG DashboardResponse
	if err := json.NewDecoder(tgRes.Body).Decode(&dashAfterTG); err != nil {
		t.Fatalf("decode dashAfterTG: %v", err)
	}
	if strings.Contains(dashAfterTG.TelegramSettings.MaskedBotToken, "SecretTelegramBotTokenValue") ||
		!strings.Contains(dashAfterTG.TelegramSettings.MaskedBotToken, "***") {
		t.Fatalf("Rule #20 violation: expected masked Telegram bot token, got %q", dashAfterTG.TelegramSettings.MaskedBotToken)
	}
	if !dashAfterTG.TelegramSettings.NotifyOnS3 {
		t.Fatalf("TASK-CAR-08 regression: expected notify_on_s3_plan=true payload to preserve NotifyOnS3=true, got false")
	}
	if dashAfterTG.TelegramSettings.NotifyOnS3Plan == nil || !*dashAfterTG.TelegramSettings.NotifyOnS3Plan {
		t.Fatalf("TASK-CAR-08 regression: expected NotifyOnS3Plan=true in DTO response")
	}

	// 2b. Verify POST /api/settings/telegram/test auto-saves inline form inputs and returns delivery_status + telegram_settings + settings
	tgTestSendReq, _ := json.Marshal(map[string]any{
		"task_key":           "TASK-11",
		"enabled":            true,
		"bot_token":          "7766554433:AAHInlineSavedBotTokenValue888",
		"chat_id":            "-1004390214071",
		"bot_username":       "@tk001sea_bot",
		"notify_on_s3_plan":  true,
		"auto_run_on_approve": true,
	})
	tgTestRes, err := http.Post(ts.URL+"/api/settings/telegram/test", "application/json", bytes.NewReader(tgTestSendReq))
	if err != nil {
		t.Fatalf("POST /api/settings/telegram/test: %v", err)
	}
	defer tgTestRes.Body.Close()
	if tgTestRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/settings/telegram/test, got %d", tgTestRes.StatusCode)
	}
	var tgTestData struct {
		Status           string              `json:"status"`
		DeliveryStatus   string              `json:"delivery_status"`
		TelegramSettings TelegramSettingsDTO `json:"telegram_settings"`
		Settings         TelegramSettingsDTO `json:"settings"`
	}
	if err := json.NewDecoder(tgTestRes.Body).Decode(&tgTestData); err != nil {
		t.Fatalf("decode tgTestData: %v", err)
	}
	if tgTestData.DeliveryStatus == "" || tgTestData.DeliveryStatus == "DISABLED" {
		t.Fatalf("TASK-CAR-08 regression: expected non-DISABLED delivery_status from /api/settings/telegram/test, got %q", tgTestData.DeliveryStatus)
	}
	if !tgTestData.TelegramSettings.NotifyOnS3 || tgTestData.TelegramSettings.ChatID != "-1004390214071" || tgTestData.Settings.ChatID != "-1004390214071" {
		t.Fatalf("TASK-CAR-08 regression: expected inline test-send inputs to be persisted in both telegram_settings and settings, got %+v", tgTestData.TelegramSettings)
	}

	// 3. Create task TASK-CAR-TEST, paste Audit Remediation into Co-Architect Chat (triggers S3 Telegram notification + .audit-remediation-ready)
	createBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-CAR-TEST",
		"title":       "tích hợp Master MongoDB Destination (S5 Audit & Telegram Approve)",
		"description": "Verify S5 Adversarial Audit Hook, audit_report_<suffix>.md, and Telegram Remote APPROVE",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos": []string{
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service",
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-web",
			"/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		},
		"epic":           "DATA",
		"milestone":      "M2",
		"complexity":     "high",
		"auto_autopilot": true,
	})
	cRes, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-CAR-TEST: %v", err)
	}
	_ = cRes.Body.Close()

	// Paste Audit Remediation so TASK-A04..A07 are registered
	auditMsg, _ := json.Marshal(map[string]any{
		"task_key": "TASK-CAR-TEST",
		"action":   "revise_solution",
		"message": "### I. Tổng Hợp Lỗ Hổng Phát Hiện Thực Tế\n" +
			"1. **`AF-DH-01 (P0 — E11000 DuplicateKeyError)`**\n" +
			"2. **`AF-DH-02 (P1 — Bỏ sót dec.Decode(&arr))`**\n" +
			"3. **`AF-DH-03 (P1 — Hardcode SourceRowCount = -1)`**\n" +
			"### II. Tóm Tắt Phương Án Khắc Phục Duy Nhất (`TASK-A04 .. TASK-A07`)\n" +
			"1. **`TASK-A04` (`centralized-data-service` — `transmuter_mongo.go`):** Aggregation Pipeline Update.\n" +
			"2. **`TASK-A05` (`centralized-data-service` — `recon_dest_mongo.go`):** Bổ sung dec.Decode(&arr).\n" +
			"3. **`TASK-A06` (`centralized-data-service` — `recon_execute_heal_handler.go`):** CountDocuments.\n" +
			"4. **`TASK-A07` (`ai-orchestrator` — `task_chat.go`):** Đồng bộ Co-Architect Chat.\n",
	})
	chatRes, err := http.Post(ts.URL+"/api/tasks/chat", "application/json", bytes.NewReader(auditMsg))
	if err != nil {
		t.Fatalf("POST /api/tasks/chat TASK-CAR-TEST: %v", err)
	}
	_ = chatRes.Body.Close()

	// 4. Test S5 -> S3 Re-Plan Loop when running action="qa" BEFORE isolated remediation diff is synthesized:
	// First remove .audit-remediation-ready temporarily and write an incomplete diff.patch so S5 Audit detects missing remediation!
	taskDir := srv.buildTaskIsolationMeta("TASK-CAR-TEST").ArtifactDir
	_ = os.Remove(filepath.Join(taskDir, ".audit-remediation-ready"))
	_ = os.WriteFile(filepath.Join(taskDir, "diff.patch"), []byte("# incomplete diff without mongo.Pipeline\n"), 0o644)

	qaFailReq, _ := json.Marshal(map[string]any{
		"task_key":      "TASK-CAR-TEST",
		"action":        "qa",
		"step_delay_ms": 5,
	})
	qaFailRes, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(qaFailReq))
	if err != nil {
		t.Fatalf("POST /api/tasks/action qa fail: %v", err)
	}
	var dashAfterQAFail DashboardResponse
	_ = json.NewDecoder(qaFailRes.Body).Decode(&dashAfterQAFail)
	_ = qaFailRes.Body.Close()

	if dashAfterQAFail.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected S5 Adversarial Audit failure to trigger S5 -> S3 Re-Plan back to NEEDS_YOU, got %q", dashAfterQAFail.ActiveTaskStage)
	}
	auditSuffixFile := filepath.Join(taskDir, "audit_report_TASK-CAR-TEST.md")
	auditSuffixBytes, err := os.ReadFile(auditSuffixFile)
	if err != nil || !strings.Contains(string(auditSuffixBytes), "Action Required: S5 -> S3 Re-Plan") {
		t.Fatalf("expected %s to record S5 -> S3 Re-Plan verdict, err=%v", auditSuffixFile, err)
	}

	// 5. Re-enable .audit-remediation-ready and trigger Remote APPROVE via Telegram Inline Keyboard callback_query (/api/telegram/webhook)
	_ = os.WriteFile(filepath.Join(taskDir, ".audit-remediation-ready"), []byte("ready\n"), 0o644)
	tgApprovePayload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 5,
		"callback_query": map[string]any{
			"id":   "cbq-test-991",
			"data": "APPROVE TASK-CAR-TEST",
			"from": map[string]any{
				"username": "trainguyen",
			},
			"message": map[string]any{
				"chat": map[string]any{
					"id": int64(-1004390214071),
				},
			},
		},
	})
	whRes, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(tgApprovePayload))
	if err != nil {
		t.Fatalf("POST /api/telegram/webhook: %v", err)
	}
	defer whRes.Body.Close()
	if whRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/telegram/webhook, got %d", whRes.StatusCode)
	}
	var whData map[string]any
	_ = json.NewDecoder(whRes.Body).Decode(&whData)
	if whData["new_stage"] != "DONE" {
		t.Fatalf("expected Telegram Inline Keyboard callback_query APPROVE to execute S4 + S5 Adversarial Audit and reach DONE, got %+v", whData)
	}

	// Verify audit_report_TASK-CAR-TEST.md now has PASS verdict + Verbatim Prompt + Co-Architect Chat reply
	auditPassBytes, err := os.ReadFile(auditSuffixFile)
	if err != nil || !strings.Contains(string(auditPassBytes), "PASS — Đủ điều kiện đóng Task") || !strings.Contains(string(auditPassBytes), "Audit lại \"quá trình\" vừa thực hiện") {
		t.Fatalf("expected %s to contain Verbatim S5 Prompt and PASS verdict, err=%v", auditSuffixFile, err)
	}
	finalDashRes, err := http.Get(ts.URL + "/api/dashboard?task_key=TASK-CAR-TEST")
	if err != nil {
		t.Fatalf("GET final dashboard: %v", err)
	}
	defer finalDashRes.Body.Close()
	var finalDash DashboardResponse
	_ = json.NewDecoder(finalDashRes.Body).Decode(&finalDash)
	lastChat := finalDash.ActiveTaskChat[len(finalDash.ActiveTaskChat)-1]
	if !strings.Contains(lastChat.Message, "S5 Adversarial Audit") || !strings.Contains(lastChat.Message, "VERIFIED") {
		t.Fatalf("expected Co-Architect Chat to contain S4+S5 completion reply, got %q", lastChat.Message)
	}
}

func TestWebUIServer_14AgentCoreFlowGapsClosed(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create a multi-repo task with MongoDB/CDC keywords to trigger S0 Boot, Tripwire TW-002, and Clean-Room / T3-T2 artifacts
	createBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-GAP14-HEAL",
		"title":       "Clean-Room Rebuild Master MongoDB Destination & CDC Watermark OCC",
		"description": "Rebuild mongo bulkUpsertMasterMongo and debezium kafka watermark continuity with clean-room spec",
		"target_repo": srv.Root,
		"target_repos": []string{
			srv.Root,
			"/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		},
		"epic":       "DATA",
		"milestone":  "M2",
		"complexity": "high",
	})
	cRes, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-GAP14-HEAL: %v", err)
	}
	_ = cRes.Body.Close()

	isoMeta := srv.buildTaskIsolationMeta("TASK-GAP14-HEAL")
	taskDir := isoMeta.ArtifactDir
	_, mode, suffix := agentgate.InferTaskTierAndMode("TASK-GAP14-HEAL", "Clean-Room Rebuild Master MongoDB Destination & CDC Watermark OCC", "Rebuild mongo bulkUpsertMasterMongo and debezium kafka watermark continuity with clean-room spec", "high", 4)
	if mode != "clean-room" {
		t.Fatalf("GAP-FLOW-04: expected mode=clean-room for Clean-Room Rebuild task, got %q", mode)
	}

	// 2. Verify GAP-FLOW-02 (S0 Boot H1), GAP-FLOW-03 (restore_points.json), GAP-FLOW-04 (01_requirements_<suffix>.md, 04_decisions_<suffix>.md, 05_progress_<suffix>.md)
	reqSuffixedPath := filepath.Join(taskDir, fmt.Sprintf("01_requirements_%s.md", suffix))
	reqBytes, err := os.ReadFile(reqSuffixedPath)
	if err != nil {
		t.Fatalf("GAP-FLOW-04: expected %s to exist: %v", reqSuffixedPath, err)
	}
	reqStr := string(reqBytes)
	for _, marker := range []string{
		"laya_health_s0:",
		"harness_surface_s0:",
		"routed_topics:",
		"matched_tripwires:",
		"mode: clean-room",
	} {
		if !strings.Contains(reqStr, marker) {
			t.Fatalf("GAP-FLOW-02/04: expected %s to contain %q", reqSuffixedPath, marker)
		}
	}

	restoreBytes, err := os.ReadFile(filepath.Join(taskDir, "restore_points.json"))
	if err != nil || !strings.Contains(string(restoreBytes), "head_sha") || !strings.Contains(string(restoreBytes), "git_top_level") {
		t.Fatalf("GAP-FLOW-03: expected restore_points.json with git rev-parse head_sha and git_top_level, err=%v", err)
	}

	decPath := filepath.Join(taskDir, fmt.Sprintf("04_decisions_%s.md", suffix))
	if _, err := os.Stat(decPath); err != nil {
		t.Fatalf("GAP-FLOW-04: expected %s to exist: %v", decPath, err)
	}

	progPath := filepath.Join(taskDir, fmt.Sprintf("05_progress_%s.md", suffix))
	progBytes, err := os.ReadFile(progPath)
	if err != nil || !strings.Contains(string(progBytes), "+07:00] [Agent:") || !strings.Contains(string(progBytes), "START") {
		t.Fatalf("GAP-FLOW-04: expected %s to contain immutable +07:00 START log, got %q (err=%v)", progPath, string(progBytes), err)
	}

	// 3. Verify GAP-FLOW-05 (13_analysis_<suffix>.md with VERIFIED vs HYPOTHESIS & H4t Tripwires)
	anaPath := filepath.Join(taskDir, fmt.Sprintf("13_analysis_%s.md", suffix))
	anaBytes, err := os.ReadFile(anaPath)
	if err != nil || !strings.Contains(string(anaBytes), "VERIFIED") || !strings.Contains(string(anaBytes), "HYPOTHESIS") {
		t.Fatalf("GAP-FLOW-05: expected %s to contain VERIFIED vs HYPOTHESIS table, err=%v", anaPath, err)
	}
	if unmet := srv.Gatekeeper.CheckUnmetTripwiresH4t(taskDir, nil, string(anaBytes)); len(unmet) > 0 {
		t.Fatalf("GAP-FLOW-05: expected 0 unmet H4t tripwires in 13_analysis, got %v", unmet)
	}

	// 4. Verify GAP-FLOW-06 & GAP-FLOW-08 (## Clean-Room Approved Spec H4a & G-04 Sub-Agent Isolation)
	solSuffixedPath := filepath.Join(taskDir, fmt.Sprintf("09_tasks_solution_%s.md", suffix))
	solBytes, err := os.ReadFile(solSuffixedPath)
	if err != nil {
		t.Fatalf("GAP-FLOW-06: expected %s to exist: %v", solSuffixedPath, err)
	}
	if h4aIssues := agentgate.ValidateCleanRoomSpecH4a(string(solBytes)); len(h4aIssues) > 0 {
		t.Fatalf("GAP-FLOW-06: expected ValidateCleanRoomSpecH4a to pass on %s, got issues=%v", solSuffixedPath, h4aIssues)
	}
	if leaks := agentgate.ValidateCleanRoomSubagentPromptG04(string(solBytes), []string{"internal/legacy/secret_source.go"}, nil); len(leaks) > 0 {
		t.Fatalf("GAP-FLOW-08: expected clean specText to pass G-04, got %v", leaks)
	}
	if leaks := agentgate.ValidateCleanRoomSubagentPromptG04("read internal/legacy/secret_source.go directly", []string{"internal/legacy/secret_source.go"}, nil); len(leaks) == 0 {
		t.Fatalf("GAP-FLOW-08: expected G-04 to reject sub-agent prompt containing forbidden source path")
	}

	// 5. Verify GAP-FLOW-07 (Calibrated DG1 Claims JSON with REF: and K1/K2/K3 controls)
	dg1ClaimsPath := filepath.Join(taskDir, fmt.Sprintf("06_laya_claims_dg1_%s.json", suffix))
	dg1Bytes, err := os.ReadFile(dg1ClaimsPath)
	if err != nil || !strings.Contains(string(dg1Bytes), "control_false") || !strings.Contains(string(dg1Bytes), "control_true") || !strings.Contains(string(dg1Bytes), "REF: ") {
		t.Fatalf("GAP-FLOW-07: expected calibrated %s with REF: and K1/K2/K3 controls, err=%v", dg1ClaimsPath, err)
	}

	// 6. Test GAP-FLOW-11 (Autonomous S5 -> S3 -> S4 -> S5 Self-Healing Loop in 1 approve_and_run call without asking for 2nd APPROVE!):
	// Paste Audit Remediation subtasks (TASK-A04..A07), then remove .audit-remediation-ready before calling approve_and_run.
	// Loop 1 will fail QA -> triggerS5ToS3RePlan updates S3 & restores .audit-remediation-ready -> Loop 2 automatically synthesizes diff & reaches DONE!
	auditMsg, _ := json.Marshal(map[string]any{
		"task_key": "TASK-GAP14-HEAL",
		"action":   "revise_solution",
		"message": "### I. Tổng Hợp Lỗ Hổng Phát Hiện Thực Tế\n" +
			"1. **`AF-DH-01 (P0 — E11000 DuplicateKeyError)`**\n" +
			"### II. Tóm Tắt Phương Án Khắc Phục Duy Nhất (`TASK-A04 .. TASK-A07`)\n" +
			"1. **`TASK-A04` (`centralized-data-service` — `transmuter_mongo.go`):** Aggregation Pipeline Update.\n" +
			"2. **`TASK-A05` (`centralized-data-service` — `recon_dest_mongo.go`):** Bổ sung dec.Decode(&arr).\n" +
			"3. **`TASK-A06` (`centralized-data-service` — `recon_execute_heal_handler.go`):** CountDocuments.\n" +
			"4. **`TASK-A07` (`ai-orchestrator` — `task_chat.go`):** Đồng bộ Co-Architect Chat.\n",
	})
	chatRes, err := http.Post(ts.URL+"/api/tasks/chat", "application/json", bytes.NewReader(auditMsg))
	if err != nil {
		t.Fatalf("POST /api/tasks/chat TASK-GAP14-HEAL: %v", err)
	}
	_ = chatRes.Body.Close()

	// Remove .audit-remediation-ready so Loop 1 fails S5 QA and must self-heal into Loop 2!
	_ = os.Remove(filepath.Join(taskDir, ".audit-remediation-ready"))
	_ = os.WriteFile(filepath.Join(taskDir, "diff.patch"), []byte("# incomplete diff on loop 1\n"), 0o644)

	approveRunReq, _ := json.Marshal(map[string]any{
		"task_key":      "TASK-GAP14-HEAL",
		"action":        "approve_and_run",
		"step_delay_ms": 1,
	})
	appRes, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(approveRunReq))
	if err != nil {
		t.Fatalf("POST /api/tasks/action approve_and_run TASK-GAP14-HEAL: %v", err)
	}
	var dashAfterHeal DashboardResponse
	_ = json.NewDecoder(appRes.Body).Decode(&dashAfterHeal)
	_ = appRes.Body.Close()

	if dashAfterHeal.ActiveTaskStage != "DONE" {
		t.Fatalf("GAP-FLOW-11: expected autonomous S5->S3->S4->S5 self-healing loop to heal on Loop 2 and reach DONE in a single approve_and_run call, got %q", dashAfterHeal.ActiveTaskStage)
	}

	// Verify SELF_HEAL_LOOP_1 and DONE were appended to 05_progress_<suffix>.md
	progAfterBytes, err := os.ReadFile(progPath)
	if err != nil || !strings.Contains(string(progAfterBytes), "SELF_HEAL_LOOP_1") || !strings.Contains(string(progAfterBytes), "DONE") {
		t.Fatalf("GAP-FLOW-11/14: expected %s to contain SELF_HEAL_LOOP_1 and DONE entries, got:\n%s", progPath, string(progAfterBytes))
	}

	// 7. Verify GAP-FLOW-09 & GAP-FLOW-13 (06_evidence/EV-*.json with src_fingerprint & H4c Claim-Evidence Receipt Verification)
	evEntries, err := os.ReadDir(filepath.Join(taskDir, "06_evidence"))
	if err != nil || len(evEntries) == 0 {
		t.Fatalf("GAP-FLOW-09: expected 06_evidence/EV-*.json receipts in %s, err=%v", taskDir, err)
	}
	auditSuffixedPath := filepath.Join(taskDir, fmt.Sprintf("audit_report_%s.md", suffix))
	auditBytes, err := os.ReadFile(auditSuffixedPath)
	if err != nil {
		t.Fatalf("GAP-FLOW-12: expected %s to exist: %v", auditSuffixedPath, err)
	}
	if h4cIssues := srv.Gatekeeper.VerifyClaimEvidenceReceiptsH4c(rCtxOrBackground(), taskDir, string(auditBytes)); len(h4cIssues) > 0 {
		t.Fatalf("GAP-FLOW-13 (Hook H4c): expected 0 claim-evidence issues in %s, got %v", auditSuffixedPath, h4cIssues)
	}

	// 8. Verify GAP-FLOW-14 (verify_governance.py --stage S6 --suffix <suffix> passes on taskDir!)
	govOK, govOut, govErr := srv.Gatekeeper.RunGovernanceLinter(rCtxOrBackground(), taskDir, "S6", suffix)
	if !govOK || govErr != nil {
		t.Fatalf("GAP-FLOW-14: expected verify_governance.py --stage S6 --suffix %s to PASS on %s, got ok=%v err=%v out=%s", suffix, taskDir, govOK, govErr, govOut)
	}

	// 9. Verify Hook H6 (3-Loop Escalation Mentor writes 06_harness2_recursion_escalation.json when a task has 0 diff and 0 remediable subtasks):
	// In TestWebUIServer_EndToEndTaskLifecycleAndIsolation step 4a, or on a fresh unremediable task:
	unremBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-H6-ESCALATE",
		"title":       "Unremediable Zero-Diff Escalation Test",
		"description": "Verify Hook H6 triggers after 3 consecutive failed loops",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"complexity":  "high",
	})
	uRes, _ := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(unremBody))
	if uRes != nil {
		_ = uRes.Body.Close()
	}
	escReq, _ := json.Marshal(map[string]any{
		"task_key":      "TASK-H6-ESCALATE",
		"action":        "approve_and_run",
		"step_delay_ms": 1,
	})
	escRes, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(escReq))
	if err != nil {
		t.Fatalf("POST /api/tasks/action TASK-H6-ESCALATE: %v", err)
	}
	_ = escRes.Body.Close()
	h6File := filepath.Join(srv.buildTaskIsolationMeta("TASK-H6-ESCALATE").ArtifactDir, "06_harness2_recursion_escalation.json")
	if rawH6, err := os.ReadFile(h6File); err != nil || !strings.Contains(string(rawH6), "H6_ESCALATION_MENTOR") {
		t.Fatalf("GAP-FLOW-11 (Hook H6): expected %s to be written after 3 failed loops, err=%v", h6File, err)
	}
}

func rCtxOrBackground() context.Context {
	return context.Background()
}

func TestWebUIServer_SubTaskFractalLifecycle_InputOutputPlanAndBranch(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify UI static assets expose Sub-Task Fractal Inspector hooks in View 1 & View 2
	for _, assetCheck := range []struct {
		path    string
		markers []string
	}{
		{
			path:    "/",
			markers: []string{"subtask-fractal-inspector", "wave-subtask-fractal-inspector"},
		},
		{
			path:    "/app.js",
			markers: []string{"buildSubTaskFractalInspectorHTML", "selectSubTaskFractal", "runSubTaskLifecycle", "inspectSubTaskDedicatedPlan"},
		},
		{
			path:    "/app.css",
			markers: []string{".subtask-fractal-inspector", ".sfi-contracts-grid", ".sfi-pipeline-row"},
		},
	} {
		res, err := http.Get(ts.URL + assetCheck.path)
		if err != nil {
			t.Fatalf("GET %s: %v", assetCheck.path, err)
		}
		bodyBytes, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		bodyStr := string(bodyBytes)
		for _, m := range assetCheck.markers {
			if !strings.Contains(bodyStr, m) {
				t.Fatalf("expected %s to contain %q", assetCheck.path, m)
			}
		}
	}

	// 2. Create a multi-repo task that decomposes into 10 Sub-Tasks (TASK-M01 .. TASK-M10 across Wave 0 .. Wave 4)
	createBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-FRACTAL-10",
		"title":       "Tích hợp Master MongoDB Destination (10 Fractal Sub-Tasks Wave 0..4)",
		"description": "Verify every Sub-Task is an independent Task with InputContract, Dedicated Plan & Code Demo, Start->Isolated Branch pipeline, and OutputContract",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos": []string{
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service",
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-web",
			"/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		},
		"epic":       "DATA",
		"milestone":  "M2",
		"complexity": "high",
	})
	cRes, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-FRACTAL-10: %v", err)
	}
	var dash DashboardResponse
	if err := json.NewDecoder(cRes.Body).Decode(&dash); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	_ = cRes.Body.Close()

	if len(dash.ActiveSubTasks) < 10 {
		t.Fatalf("expected >= 10 decomposed fractal Sub-Tasks, got %d", len(dash.ActiveSubTasks))
	}

	taskDir := srv.buildTaskIsolationMeta("TASK-FRACTAL-10").ArtifactDir
	subMap := make(map[string]SubTaskDTO, len(dash.ActiveSubTasks))

	// 3. Verify every Sub-Task has its own InputContract, Dedicated Plan/CodeDemo on disk, 6-step PipelineStages, and OutputContract
	for _, st := range dash.ActiveSubTasks {
		subMap[st.SubTaskID] = st
		if st.BranchName == "" || !strings.HasPrefix(strings.ToLower(st.BranchName), "task/task-fractal-10/") {
			t.Fatalf("sub-task %s missing isolated BranchName, got %q", st.SubTaskID, st.BranchName)
		}
		if len(st.PipelineStages) != 6 {
			t.Fatalf("sub-task %s expected 6 pipeline stages (Start -> Plan -> You -> Implement -> QA -> Branch), got %d", st.SubTaskID, len(st.PipelineStages))
		}
		if st.PlanMarkdown == "" || st.CodeDemo == "" || st.DiffPreview == "" || st.AuditSummary == "" {
			t.Fatalf("sub-task %s missing dedicated PlanMarkdown/CodeDemo/DiffPreview/AuditSummary", st.SubTaskID)
		}
		if st.InputContract.SourceContext == "" || len(st.InputContract.UpstreamDeliverables) == 0 || st.InputContract.InputSpec == "" {
			t.Fatalf("sub-task %s missing InputContract fields: %+v", st.SubTaskID, st.InputContract)
		}
		if len(st.OutputContract.Deliverables) == 0 || st.OutputContract.DiffFile == "" || st.OutputContract.AuditFile == "" || st.OutputContract.EvidenceReceiptID == "" {
			t.Fatalf("sub-task %s missing OutputContract fields: %+v", st.SubTaskID, st.OutputContract)
		}

		// Verify physical files exist inside .orchestrator/tasks/TASK-FRACTAL-10/subtasks/<SubTaskID>/
		subDir := filepath.Join(taskDir, "subtasks", st.SubTaskID)
		for _, fName := range []string{
			"01_requirements.md",
			"08_tasks.md",
			"13_analysis.md",
			"09_tasks_solution.md",
			"diff.patch",
			"audit_report.md",
			"05_progress.md",
		} {
			fPath := filepath.Join(subDir, fName)
			if raw, err := os.ReadFile(fPath); err != nil || len(raw) == 0 {
				t.Fatalf("expected physical sub-task artifact %s to exist and be non-empty, err=%v", fPath, err)
			}
		}
		// Verify physical JSON receipt in 06_evidence/<EvidenceReceiptID>.json with non-empty src_fingerprint (GAP-FRAC-AUDIT-02)
		evPath := filepath.Join(subDir, "06_evidence", st.OutputContract.EvidenceReceiptID+".json")
		rawEv, errEv := os.ReadFile(evPath)
		if errEv != nil || !strings.Contains(string(rawEv), `"src_fingerprint"`) {
			t.Fatalf("expected physical Sub-Task evidence receipt %s with src_fingerprint, err=%v", evPath, errEv)
		}
	}

	// Wave 0 Sub-Task (TASK-M01) has no upstream dependencies so UpstreamReady=true always
	m01 := subMap["TASK-M01"]
	if !m01.InputContract.UpstreamReady {
		t.Fatalf("expected Wave 0 TASK-M01 UpstreamReady=true, got false")
	}

	// 3b. Negative Test 1 (GAP-FRAC-AUDIT-01 — Upstream Wave Dependency Gate):
	// Calling subtask_implement on Wave 1 TASK-M04 before Wave 0 (TASK-M01..M03) is DONE must be BLOCKED at NEEDS_YOU!
	prematureM04Body, _ := json.Marshal(map[string]any{
		"task_key":    "TASK-FRACTAL-10",
		"sub_task_id": "TASK-M04",
		"action":      "subtask_implement",
	})
	resPremature, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(prematureM04Body))
	if err != nil {
		t.Fatalf("POST premature subtask_implement on TASK-M04: %v", err)
	}
	var dashPremature DashboardResponse
	_ = json.NewDecoder(resPremature.Body).Decode(&dashPremature)
	_ = resPremature.Body.Close()
	for _, st := range dashPremature.ActiveSubTasks {
		if st.SubTaskID == "TASK-M04" && st.Stage != "NEEDS_YOU" {
			t.Fatalf("expected premature subtask_implement on Wave 1 TASK-M04 (UpstreamReady=false) to be BLOCKED at NEEDS_YOU, got Stage=%q", st.Stage)
		}
	}

	// 3c. Negative Test 2 (GAP-FRAC-AUDIT-01 — H4b Approve Gate):
	// Calling subtask_implement on Wave 0 TASK-M01 before subtask_approve (.user-approved missing) must be BLOCKED at NEEDS_YOU!
	unapprovedM01Body, _ := json.Marshal(map[string]any{
		"task_key":    "TASK-FRACTAL-10",
		"sub_task_id": "TASK-M01",
		"action":      "subtask_implement",
	})
	resUnapproved, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(unapprovedM01Body))
	if err != nil {
		t.Fatalf("POST unapproved subtask_implement on TASK-M01: %v", err)
	}
	var dashUnapproved DashboardResponse
	_ = json.NewDecoder(resUnapproved.Body).Decode(&dashUnapproved)
	_ = resUnapproved.Body.Close()
	for _, st := range dashUnapproved.ActiveSubTasks {
		if st.SubTaskID == "TASK-M01" && st.Stage != "NEEDS_YOU" {
			t.Fatalf("expected unapproved subtask_implement on TASK-M01 to be BLOCKED by H4b at NEEDS_YOU, got Stage=%q", st.Stage)
		}
	}

	// 4. Execute TASK-M01 step-by-step through Start -> Plan -> Approve -> Implement -> QA & Branch via /api/tasks/action
	// Because Wave 1 TASK-M04 depends on all 3 Wave 0 Sub-Tasks (TASK-M01, TASK-M02, TASK-M03), TASK-M04 UpstreamReady remains false until all 3 finish!
	for _, stepCase := range []struct {
		action             string
		expectedStage      string
		expectedM04UpReady bool
	}{
		{action: "subtask_plan", expectedStage: "NEEDS_YOU", expectedM04UpReady: false},
		{action: "subtask_approve", expectedStage: "APPROVED", expectedM04UpReady: false},
		{action: "subtask_implement", expectedStage: "IN_REVIEW", expectedM04UpReady: false},
		{action: "subtask_qa", expectedStage: "DONE", expectedM04UpReady: false},
	} {
		reqBody, _ := json.Marshal(map[string]any{
			"task_key":    "TASK-FRACTAL-10",
			"sub_task_id": "TASK-M01",
			"action":      stepCase.action,
		})
		res, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			t.Fatalf("POST /api/tasks/action %s on TASK-M01: %v", stepCase.action, err)
		}
		var stepDash DashboardResponse
		if err := json.NewDecoder(res.Body).Decode(&stepDash); err != nil {
			t.Fatalf("decode stepDash (%s): %v", stepCase.action, err)
		}
		_ = res.Body.Close()

		var updatedM01, updatedM04 SubTaskDTO
		for _, st := range stepDash.ActiveSubTasks {
			if st.SubTaskID == "TASK-M01" {
				updatedM01 = st
			}
			if st.SubTaskID == "TASK-M04" {
				updatedM04 = st
			}
		}
		if updatedM01.Stage != stepCase.expectedStage {
			t.Fatalf("after %s on TASK-M01, expected Stage=%q, got %q", stepCase.action, stepCase.expectedStage, updatedM01.Stage)
		}
		if updatedM04.InputContract.UpstreamReady != stepCase.expectedM04UpReady {
			t.Fatalf("after %s on TASK-M01, expected Wave 1 TASK-M04 UpstreamReady=%v, got %v",
				stepCase.action, stepCase.expectedM04UpReady, updatedM04.InputContract.UpstreamReady)
		}
	}

	// 5. Complete remaining Wave 0 Sub-Tasks (TASK-M02 and TASK-M03) via 2-Phase subtask_autopilot on /api/tasks/subtasks
	// and verify Wave 1 TASK-M04 now has UpstreamReady=true with HANDOFF_READY ✓ deliverables!
	var dashAfterWave0 DashboardResponse
	for _, remSubID := range []string{"TASK-M02", "TASK-M03"} {
		for phase := 1; phase <= 2; phase++ {
			autoBody, _ := json.Marshal(map[string]any{
				"task_key":    "TASK-FRACTAL-10",
				"sub_task_id": remSubID,
				"action":      "subtask_autopilot",
			})
			resAuto, err := http.Post(ts.URL+"/api/tasks/subtasks", "application/json", bytes.NewReader(autoBody))
			if err != nil {
				t.Fatalf("POST /api/tasks/subtasks subtask_autopilot phase %d %s: %v", phase, remSubID, err)
			}
			if err := json.NewDecoder(resAuto.Body).Decode(&dashAfterWave0); err != nil {
				t.Fatalf("decode dashAfterWave0 (%s phase %d): %v", remSubID, phase, err)
			}
			_ = resAuto.Body.Close()
		}
	}

	var m02After, m03After, m04After SubTaskDTO
	for _, st := range dashAfterWave0.ActiveSubTasks {
		if st.SubTaskID == "TASK-M02" {
			m02After = st
		}
		if st.SubTaskID == "TASK-M03" {
			m03After = st
		}
		if st.SubTaskID == "TASK-M04" {
			m04After = st
		}
	}
	if m02After.Stage != "DONE" || m02After.Status != "done" || !m02After.OutputContract.HandoffReady {
		t.Fatalf("expected TASK-M02 after subtask_autopilot to be DONE with HandoffReady=true, got stage=%q status=%q handoff=%v", m02After.Stage, m02After.Status, m02After.OutputContract.HandoffReady)
	}
	if m03After.Stage != "DONE" || m03After.Status != "done" || !m03After.OutputContract.HandoffReady {
		t.Fatalf("expected TASK-M03 after subtask_autopilot to be DONE with HandoffReady=true, got stage=%q status=%q handoff=%v", m03After.Stage, m03After.Status, m03After.OutputContract.HandoffReady)
	}
	if !m04After.InputContract.UpstreamReady {
		t.Fatalf("expected Wave 1 TASK-M04 InputContract.UpstreamReady=true after all Wave 0 dependencies (TASK-M01, TASK-M02, TASK-M03) reached DONE")
	}
	joinedUpstream := strings.Join(m04After.InputContract.UpstreamDeliverables, " | ")
	if !strings.Contains(joinedUpstream, "HANDOFF_READY ✓") {
		t.Fatalf("expected TASK-M04 InputContract.UpstreamDeliverables to contain HANDOFF_READY ✓ from Wave 0, got %v", m04After.InputContract.UpstreamDeliverables)
	}
}

func TestWebUIServer_PipelineResilienceAndCrossModelGates_AE(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Create a multi-repo task to exercise all 5 upgrades (A..E)
	createBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-PIPE-AE",
		"title":       "Tích hợp Master MongoDB Destination (Cross-Model Gates & Circuit Breaker A..E)",
		"description": "Verify Cross-Model Auditing (A), Two-Tier Self-Healing & Circuit Breaker Max 3 (B), Worktree Slot C at S4 & Finalize & Diff Review (C), Atomic Wave Rollback (D), and 5A Hard Gate Zero-LLM + 5B Semantic Audit (E)",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos": []string{
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service",
			"/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		},
		"epic":       "DATA",
		"milestone":  "M2",
		"complexity": "high",
	})
	res, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-PIPE-AE: %v", err)
	}
	var dash DashboardResponse
	if err := json.NewDecoder(res.Body).Decode(&dash); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	_ = res.Body.Close()

	// Upgrade A: Cross-Model Auditing & Local Zero-Token Laya Gate (:8200)
	if !dash.CrossModelVerified || !dash.CLISettings.CrossModelVerified {
		t.Fatalf("expected CrossModelVerified=true by default, got dash=%v cli=%v (%s)", dash.CrossModelVerified, dash.CLISettings.CrossModelVerified, dash.CrossModelSummary)
	}
	if !strings.Contains(dash.QAAuditorModel, "claude-opus-4-6") {
		t.Fatalf("expected Slot K QAAuditorModel to default to claude-opus-4-6 family, got %q", dash.QAAuditorModel)
	}
	if !strings.Contains(dash.LayaGateModel, "convaiinnovations/laya (local :8200)") {
		t.Fatalf("expected Slot H LayaGateModel to default to 'convaiinnovations/laya (local :8200)', got %q", dash.LayaGateModel)
	}
	if !strings.Contains(dash.SecurityModel, "agy_guard.py (H1..H7) · claude-sonnet-4-6") || strings.Contains(dash.SecurityModel, "codex") {
		t.Fatalf("expected Slot I SecurityModel to default to 'agy_guard.py (H1..H7) · claude-sonnet-4-6 (antigravity)' without fabricated codex, got %q", dash.SecurityModel)
	}

	// Upgrade C, E & S2.5 Pre-Plan Critique Wave (DG0 :8200 · 0 Token) on Parent AutopilotStages:
	// - Stage "critique" ("S2.5 Wave Phản Biện Trước Plan (DG0)") precedes "plan"
	// - Stage "implement" includes "Worktree Attached (Slot C)"
	// - Stage "qa" is split into "5A. Hard Gate (Zero-LLM)" and "5B. Semantic Audit & DG3"
	// - Stage "pr" is renamed to "Finalize & Diff Review"
	var critiqueStage, implStage, qaStage, prStage AutopilotStageGroupDTO
	critiqueIdx, planIdx := -1, -1
	for idx, stg := range dash.AutopilotStages {
		switch stg.ID {
		case "critique":
			critiqueStage = stg
			critiqueIdx = idx
		case "plan":
			planIdx = idx
		case "implement":
			implStage = stg
		case "qa":
			qaStage = stg
		case "pr":
			prStage = stg
		}
	}
	if critiqueIdx < 0 || planIdx < 0 || critiqueIdx >= planIdx || len(critiqueStage.Cards) == 0 || !strings.Contains(critiqueStage.Cards[0].Title, "DG0") {
		t.Fatalf("expected S2.5 Pre-Plan Critique stage ('critique') immediately before 'plan' (critiqueIdx=%d, planIdx=%d), got %+v", critiqueIdx, planIdx, critiqueStage)
	}
	if len(implStage.Cards) < 3 || !strings.Contains(implStage.Cards[0].Title, "Worktree Attached (Slot C)") {
		t.Fatalf("expected Stage implement first card to be 'Worktree Attached (Slot C)', got %+v", implStage.Cards)
	}
	if len(qaStage.Cards) != 2 || qaStage.Cards[0].Title != "5A. Hard Gate (Zero-LLM)" || qaStage.Cards[1].Title != "5B. Semantic Audit & DG3" {
		t.Fatalf("expected Stage qa to have 2 cards ('5A. Hard Gate (Zero-LLM)' and '5B. Semantic Audit & DG3'), got %+v", qaStage.Cards)
	}
	if len(prStage.Cards) != 1 || prStage.Cards[0].Title != "Finalize & Diff Review" {
		t.Fatalf("expected Stage pr card title 'Finalize & Diff Review', got %+v", prStage.Cards)
	}

	// Verify Sub-Task level stages (including S2.5 DG0 card before Plan) & Cross-Model / Wave Sync flags
	if len(dash.ActiveSubTasks) < 3 {
		t.Fatalf("expected at least 3 Wave 0 Sub-Tasks, got %d", len(dash.ActiveSubTasks))
	}
	m01 := dash.ActiveSubTasks[0]
	if m01.BranchName == "" || !m01.SelfHealState.WaveContractSynced || !dash.CrossModelVerified {
		t.Fatalf("expected Sub-Task %s BranchName!='', SelfHealState.WaveContractSynced=true, dash.CrossModelVerified=true, got br=%q sync=%v cm=%v",
			m01.SubTaskID, m01.BranchName, m01.SelfHealState.WaveContractSynced, dash.CrossModelVerified)
	}
	if len(m01.PipelineStages) != 6 || len(m01.PipelineStages[1].Cards) < 3 || !strings.Contains(m01.PipelineStages[1].Cards[0].Title, "DG0") || len(m01.PipelineStages[4].Cards) != 2 || m01.PipelineStages[4].Cards[0].Title != "5A. Hard Gate (Zero-LLM)" || m01.PipelineStages[5].Cards[0].Title != "Finalize & Diff Review" {
		t.Fatalf("expected Sub-Task %s Stage 1 (DG0+Plan+DG1/DG2), Stage 5 (5A+5B) and Stage 6 ('Finalize & Diff Review'), got %+v", m01.SubTaskID, m01.PipelineStages)
	}

	// Upgrade B (Part 1): Tier-2 Architectural Re-Plan (S5 -> S3 -> Human Gate NEEDS_YOU)
	// First approve TASK-M01 and TASK-M02 in Wave 0, then trigger subtask_replan_s3 on TASK-M01 and verify .user-approved is revoked and Stage=NEEDS_YOU!
	for _, subID := range []string{"TASK-M01", "TASK-M02"} {
		appBody, _ := json.Marshal(map[string]any{
			"task_key":    "TASK-PIPE-AE",
			"sub_task_id": subID,
			"action":      "subtask_approve",
		})
		rApp, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(appBody))
		if err != nil {
			t.Fatalf("approve %s: %v", subID, err)
		}
		_ = rApp.Body.Close()
	}

	replanBody, _ := json.Marshal(map[string]any{
		"task_key":    "TASK-PIPE-AE",
		"sub_task_id": "TASK-M01",
		"action":      "subtask_replan_s3",
	})
	rReplan, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(replanBody))
	if err != nil {
		t.Fatalf("subtask_replan_s3 on TASK-M01: %v", err)
	}
	var dashAfterReplan DashboardResponse
	_ = json.NewDecoder(rReplan.Body).Decode(&dashAfterReplan)
	_ = rReplan.Body.Close()

	var m01Replan SubTaskDTO
	for _, st := range dashAfterReplan.ActiveSubTasks {
		if st.SubTaskID == "TASK-M01" {
			m01Replan = st
		}
	}
	if m01Replan.Stage != "NEEDS_YOU" || !m01Replan.SelfHealState.RequiresHumanReApprove || m01Replan.SelfHealState.ActiveLoopTier != "TIER2_S5_TO_S3_HUMAN_GATE" {
		t.Fatalf("expected Tier-2 S5->S3 replan on TASK-M01 to lock at NEEDS_YOU with RequiresHumanReApprove=true, got stage=%q state=%+v",
			m01Replan.Stage, m01Replan.SelfHealState)
	}

	// Re-approve TASK-M01 and run TASK-M02 to DONE so we can test Atomic Wave Rollback across Wave 0 siblings when TASK-M01 trips Circuit Breaker!
	for _, step := range []struct {
		subID  string
		action string
	}{
		{subID: "TASK-M01", action: "subtask_approve"},
		{subID: "TASK-M02", action: "subtask_implement"},
		{subID: "TASK-M02", action: "subtask_qa"},
	} {
		b, _ := json.Marshal(map[string]any{
			"task_key":    "TASK-PIPE-AE",
			"sub_task_id": step.subID,
			"action":      step.action,
		})
		rStep, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s on %s: %v", step.action, step.subID, err)
		}
		_ = rStep.Body.Close()
	}

	// Upgrade E + Upgrade B (Part 2) + Upgrade D:
	// Corrupt TASK-M01's on-disk diff.patch so Stage 5A (Hard Gate Zero-LLM) fails deterministically.
	// - Attempts 1 & 2 must short-circuit at 5A with SemanticGateLLMCalls == 0 and route Tier-1 (S5 -> S4 RUNNING, Retry 1/3, 2/3).
	// - Attempt 3 must trip the Circuit Breaker (Retry 3/3, CircuitOpen=true) and trigger Atomic Wave Rollback across Wave 0 (rolling TASK-M02 back to NEEDS_YOU with WaveRollbackTriggered=true)!
	taskDir := srv.buildTaskIsolationMeta("TASK-PIPE-AE").ArtifactDir
	m01DiffPath := filepath.Join(taskDir, "subtasks", "TASK-M01", "diff.patch")

	for attempt := 1; attempt <= 3; attempt++ {
		_ = os.WriteFile(m01DiffPath, []byte("# broken diff without target symbol\n"), 0o644)
		qaBody, _ := json.Marshal(map[string]any{
			"task_key":    "TASK-PIPE-AE",
			"sub_task_id": "TASK-M01",
			"action":      "subtask_qa",
		})
		rQA, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(qaBody))
		if err != nil {
			t.Fatalf("subtask_qa attempt %d on TASK-M01: %v", attempt, err)
		}
		var dashQA DashboardResponse
		_ = json.NewDecoder(rQA.Body).Decode(&dashQA)
		_ = rQA.Body.Close()

		var curM01, curM02 SubTaskDTO
		for _, st := range dashQA.ActiveSubTasks {
			if st.SubTaskID == "TASK-M01" {
				curM01 = st
			}
			if st.SubTaskID == "TASK-M02" {
				curM02 = st
			}
		}

		// Verify 5A Hard Gate (Zero-LLM) short-circuited 5B with 0 LLM Auditor calls wasted!
		if curM01.SelfHealState.HardGateZeroLLMPassed || curM01.SelfHealState.SemanticGateLLMCalls != 0 || curM01.SelfHealState.SemanticGateStatus != "SKIPPED_BY_5A_HARD_GATE" {
			t.Fatalf("attempt %d: expected 5A Hard Gate fail to skip 5B with 0 LLM calls, got %+v", attempt, curM01.SelfHealState)
		}
		if curM01.SelfHealState.RetryCount != attempt {
			t.Fatalf("attempt %d: expected RetryCount=%d, got %d", attempt, attempt, curM01.SelfHealState.RetryCount)
		}

		if attempt < 3 {
			if curM01.Stage != "RUNNING" || curM01.SelfHealState.CircuitOpen {
				t.Fatalf("attempt %d (<3): expected Tier-1 S5->S4 loop (Stage=RUNNING, CircuitOpen=false), got stage=%q circuit=%v",
					attempt, curM01.Stage, curM01.SelfHealState.CircuitOpen)
			}
		} else {
			// Attempt 3: Circuit Breaker trips & Atomic Wave Rollback resets Wave 0 sibling TASK-M02!
			if !curM01.SelfHealState.CircuitOpen || curM01.Stage != "NEEDS_YOU" {
				t.Fatalf("attempt 3: expected CircuitOpen=true and Stage=NEEDS_YOU on TASK-M01, got stage=%q state=%+v",
					curM01.Stage, curM01.SelfHealState)
			}
			if !curM02.SelfHealState.WaveRollbackTriggered || curM02.Stage != "NEEDS_YOU" {
				t.Fatalf("attempt 3: expected Atomic Wave Rollback to reset Wave 0 sibling TASK-M02 to Stage=NEEDS_YOU with WaveRollbackTriggered=true, got stage=%q state=%+v",
					curM02.Stage, curM02.SelfHealState)
			}
		}
	}
}

func TestWebUIServer_TaskDelete_LiveRunningProgress_And_StopProcessControl(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify HTML/JS/CSS markers for TASK-CTRL-01, TASK-CTRL-02, TASK-CTRL-03
	idxRes, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	idxBytes, _ := io.ReadAll(idxRes.Body)
	_ = idxRes.Body.Close()
	idxHTML := string(idxBytes)
	for _, marker := range []string{
		"global-running-progress-banner",
		"ctx-btn-stop-task",
		"ctx-btn-delete-task",
		"btn-cleanup-done-tasks",
		"hud-btn-stop-task",
	} {
		if !strings.Contains(idxHTML, marker) {
			t.Fatalf("expected index.html to contain %q", marker)
		}
	}

	jsRes, err := http.Get(ts.URL + "/app.js")
	if err != nil {
		t.Fatalf("GET /app.js: %v", err)
	}
	jsBytes, _ := io.ReadAll(jsRes.Body)
	_ = jsRes.Body.Close()
	jsCode := string(jsBytes)
	for _, fnName := range []string{
		"renderGlobalRunningProgressBanner",
		"stopRunningProcess",
		"deleteSingleTask",
		"deleteDoneTasksBulk",
		"deleteSingleSubTask",
		"lsp.step_index",
		"lsp.stage_label",
		"lsp.current_action",
		"lsp.active_role_slot",
	} {
		if !strings.Contains(jsCode, fnName) {
			t.Fatalf("expected app.js to contain %q", fnName)
		}
	}

	// 2. Create a READY task (TASK-CTRL-TEST) and verify it does NOT falsely report is_running=true when idle
	createBody, _ := json.Marshal(map[string]any{
		"external_id":    "TASK-CTRL-TEST",
		"title":          "Kiểm thử Live Progress, Stop Process & Delete Task",
		"description":    "Verify live running progress banner, mid-flight context cancellation, and tombstone deletion",
		"target_repo":    srv.Root,
		"epic":           "UX",
		"milestone":      "M2",
		"complexity":     "medium",
		"auto_autopilot": false,
	})
	cRes, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-CTRL-TEST: %v", err)
	}
	var dashIdle DashboardResponse
	if err := json.NewDecoder(cRes.Body).Decode(&dashIdle); err != nil {
		t.Fatalf("decode dashIdle: %v", err)
	}
	_ = cRes.Body.Close()

	if dashIdle.ActiveTaskKey != "TASK-CTRL-TEST" || dashIdle.ActiveTaskStage != "READY" {
		t.Fatalf("expected TASK-CTRL-TEST at READY stage, got key=%q stage=%q", dashIdle.ActiveTaskKey, dashIdle.ActiveTaskStage)
	}
	if dashIdle.LiveExecution.IsRunning || dashIdle.LiveStepProgress.Active {
		t.Fatalf("TASK-CTRL-02 regression: idle READY task must have IsRunning=false and LiveStepProgress.Active=false, got le=%+v lsp=%+v",
			dashIdle.LiveExecution, dashIdle.LiveStepProgress)
	}

	// 3. Start TASK-CTRL-TEST with step_delay_ms=1500 in a goroutine, observe live running progress, then call /api/tasks/stop mid-flight
	runDoneCh := make(chan DashboardResponse, 1)
	go func() {
		actBody, _ := json.Marshal(map[string]any{
			"task_key":      "TASK-CTRL-TEST",
			"action":        "plan",
			"step_delay_ms": 1500,
		})
		rAct, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(actBody))
		if err != nil {
			return
		}
		defer rAct.Body.Close()
		var d DashboardResponse
		_ = json.NewDecoder(rAct.Body).Decode(&d)
		runDoneCh <- d
	}()

	// Poll dashboard until LiveStepProgress.Active == true
	var observedRunning bool
	for i := 0; i < 30; i++ {
		time.Sleep(50 * time.Millisecond)
		rPoll, err := http.Get(ts.URL + "/api/dashboard?task_key=TASK-CTRL-TEST")
		if err != nil {
			continue
		}
		var dPoll DashboardResponse
		_ = json.NewDecoder(rPoll.Body).Decode(&dPoll)
		_ = rPoll.Body.Close()
		if dPoll.LiveStepProgress.Active && dPoll.LiveStepProgress.TaskKey == "TASK-CTRL-TEST" {
			observedRunning = true
			if !dPoll.LiveStepProgress.CanStop || !dPoll.LiveExecution.IsRunning || !dPoll.LiveExecution.CanStop || dPoll.LiveStepProgress.Percent <= 0 {
				t.Fatalf("expected active running state with CanStop=true and Percent>0, got lsp=%+v le=%+v",
					dPoll.LiveStepProgress, dPoll.LiveExecution)
			}
			// GAP-CTRL-AUDIT-01: Verify StepIndex, StageLabel, CurrentAction, ActiveRoleSlot are populated
			if dPoll.LiveStepProgress.StepIndex <= 0 || dPoll.LiveStepProgress.StageLabel == "" || dPoll.LiveStepProgress.CurrentAction == "" || dPoll.LiveStepProgress.ActiveRoleSlot == "" {
				t.Fatalf("GAP-CTRL-AUDIT-01: expected non-empty StepIndex/StageLabel/CurrentAction/ActiveRoleSlot on LiveStepProgress, got %+v",
					dPoll.LiveStepProgress)
			}
			break
		}
	}
	if !observedRunning {
		t.Fatalf("expected to observe LiveStepProgress.Active=true while TASK-CTRL-TEST was running with step_delay_ms=1500")
	}

	// Call POST /api/tasks/stop mid-flight!
	stopBody, _ := json.Marshal(map[string]any{
		"task_key": "TASK-CTRL-TEST",
		"reason":   "User clicked Stop Process in E2E test",
	})
	rStop, err := http.Post(ts.URL+"/api/tasks/stop", "application/json", bytes.NewReader(stopBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/stop: %v", err)
	}
	var dashStopped DashboardResponse
	if err := json.NewDecoder(rStop.Body).Decode(&dashStopped); err != nil {
		t.Fatalf("decode dashStopped: %v", err)
	}
	_ = rStop.Body.Close()

	// Wait for the in-flight /api/tasks/action goroutine to return promptly after cancellation
	select {
	case dashAfterAction := <-runDoneCh:
		// GAP-CTRL-AUDIT-02: Verify in-flight goroutine did not overwrite NEEDS_YOU stage after cancellation
		if dashAfterAction.ActiveTaskStage != "NEEDS_YOU" {
			t.Fatalf("GAP-CTRL-AUDIT-02: expected cancelled action response to preserve NEEDS_YOU stage, got %q", dashAfterAction.ActiveTaskStage)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected in-flight /api/tasks/action to abort promptly after /api/tasks/stop")
	}

	if dashStopped.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected stopped task stage to transition to NEEDS_YOU, got %q", dashStopped.ActiveTaskStage)
	}
	if dashStopped.LiveStepProgress.Active || !dashStopped.LiveStepProgress.WasStopped || !dashStopped.LiveExecution.WasStopped {
		t.Fatalf("expected WasStopped=true and Active=false after stop, got lsp=%+v le=%+v",
			dashStopped.LiveStepProgress, dashStopped.LiveExecution)
	}
	foundStopLog := false
	for _, ln := range dashStopped.ActiveTaskLogs {
		if strings.Contains(ln, "[⏹ Dừng Tiến Trình]") {
			foundStopLog = true
			break
		}
	}
	if !foundStopLog {
		t.Fatalf("expected ActiveTaskLogs to contain [⏹ Dừng Tiến Trình], got %v", dashStopped.ActiveTaskLogs)
	}

	// 4. Delete a single Sub-Task from TASK-CTRL-TEST
	if len(dashStopped.ActiveSubTasks) < 2 {
		t.Fatalf("expected at least 2 sub-tasks on TASK-CTRL-TEST, got %d", len(dashStopped.ActiveSubTasks))
	}
	subToDelete := dashStopped.ActiveSubTasks[0].SubTaskID
	prevSubCount := len(dashStopped.ActiveSubTasks)
	delSubBody, _ := json.Marshal(map[string]any{
		"mode":        "subtask",
		"task_key":    "TASK-CTRL-TEST",
		"sub_task_id": subToDelete,
	})
	rDelSub, err := http.Post(ts.URL+"/api/tasks/delete", "application/json", bytes.NewReader(delSubBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/delete subtask: %v", err)
	}
	var dashAfterDelSub DashboardResponse
	_ = json.NewDecoder(rDelSub.Body).Decode(&dashAfterDelSub)
	_ = rDelSub.Body.Close()
	if len(dashAfterDelSub.ActiveSubTasks) != prevSubCount-1 {
		t.Fatalf("expected sub-task count to decrease from %d to %d, got %d", prevSubCount, prevSubCount-1, len(dashAfterDelSub.ActiveSubTasks))
	}

	// 5. Delete single Task TASK-CTRL-TEST and verify tombstone + worktree directory removal (GAP-CTRL-AUDIT-03)
	taskWTDir := filepath.Join(srv.Root, ".orchestrator", "worktrees", "TASK-CTRL-TEST")
	_ = os.MkdirAll(taskWTDir, 0o755)
	_ = os.WriteFile(filepath.Join(taskWTDir, "marker.txt"), []byte("wt"), 0o644)

	delTaskBody, _ := json.Marshal(map[string]any{
		"mode":     "single",
		"task_key": "TASK-CTRL-TEST",
	})
	rDelTask, err := http.Post(ts.URL+"/api/tasks/delete", "application/json", bytes.NewReader(delTaskBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/delete single: %v", err)
	}
	var dashAfterDelTask DashboardResponse
	_ = json.NewDecoder(rDelTask.Body).Decode(&dashAfterDelTask)
	_ = rDelTask.Body.Close()

	if _, wtStatErr := os.Stat(taskWTDir); !os.IsNotExist(wtStatErr) {
		t.Fatalf("GAP-CTRL-AUDIT-03: expected worktree dir %s to be removed on task deletion, statErr=%v", taskWTDir, wtStatErr)
	}

	for _, col := range dashAfterDelTask.Waves {
		for _, n := range col.Nodes {
			if n.Key == "TASK-CTRL-TEST" {
				t.Fatalf("expected TASK-CTRL-TEST to be removed from wave nodes after deletion")
			}
		}
	}
	tombstones := srv.loadDeletedTasksSet()
	if !tombstones["TASK-CTRL-TEST"] {
		t.Fatalf("expected TASK-CTRL-TEST to be recorded in .orchestrator/deleted_tasks.json tombstones, got %v", tombstones)
	}

	// 6. Bulk delete DONE tasks (mode="done") while keeping active task
	delDoneBody, _ := json.Marshal(map[string]any{
		"mode":     "done",
		"task_key": dashAfterDelTask.ActiveTaskKey,
	})
	rDelDone, err := http.Post(ts.URL+"/api/tasks/delete", "application/json", bytes.NewReader(delDoneBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/delete done: %v", err)
	}
	var dashAfterDelDone DashboardResponse
	_ = json.NewDecoder(rDelDone.Body).Decode(&dashAfterDelDone)
	_ = rDelDone.Body.Close()

	for _, col := range dashAfterDelDone.Waves {
		for _, n := range col.Nodes {
			if n.Stage == "DONE" && n.Key != dashAfterDelDone.ActiveTaskKey {
				t.Fatalf("expected all non-active DONE tasks to be cleaned up, but found %s (%s)", n.Key, n.Stage)
			}
		}
	}
}

func TestTelegramChatIDScientificNotationAndTask13Execution(t *testing.T) {
	// 1. Unit verification of formatTelegramChatID and telegramChatIDMatches for 13-digit negative Telegram supergroup IDs
	supergroupFloat := float64(-1004390214071)
	formattedFloat := formatTelegramChatID(supergroupFloat)
	if formattedFloat != "-1004390214071" {
		t.Fatalf("expected formatTelegramChatID(float64(-1004390214071)) == \"-1004390214071\", got %q", formattedFloat)
	}
	if strings.Contains(formattedFloat, "e+") || strings.Contains(formattedFloat, "E+") {
		t.Fatalf("formatTelegramChatID must never emit scientific notation: %q", formattedFloat)
	}
	if formattedNum := formatTelegramChatID(json.Number("-1004390214071")); formattedNum != "-1004390214071" {
		t.Fatalf("expected formatTelegramChatID(json.Number) == \"-1004390214071\", got %q", formattedNum)
	}
	if formattedInt := formatTelegramChatID(int64(-1004390214071)); formattedInt != "-1004390214071" {
		t.Fatalf("expected formatTelegramChatID(int64) == \"-1004390214071\", got %q", formattedInt)
	}
	if !telegramChatIDMatches("-1004390214071", formattedFloat) {
		t.Fatalf("expected telegramChatIDMatches(\"-1004390214071\", %q) to be true", formattedFloat)
	}

	// 2. End-to-End verification: TASK-13 (with TASK-13-M01 and TASK-13-M02) approved via Telegram supergroup (-1004390214071) reaches DONE on Loop 1
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody, _ := json.Marshal(map[string]any{
		"external_id": "TASK-13",
		"title":       "Test Telegram Remote APPROVE and S4-S5 Execution for TASK-13",
		"description": "Verify Telegram supergroup chat_id -1004390214071 and TASK-13-M01 / TASK-13-M02 isolated diff synthesis.",
		"target_repo": "/Users/trainguyen/Documents/work/ai-orchestrator",
		"complexity":  "high",
		"milestone":   "M1",
		"epic":        "ORCH",
		"assignee":    "Coder",
	})
	resCreate, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks TASK-13: %v", err)
	}
	if resCreate.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resCreate.Body)
		_ = resCreate.Body.Close()
		t.Fatalf("expected 200 OK from /api/tasks, got %d: %s", resCreate.StatusCode, string(rawErr))
	}
	_ = resCreate.Body.Close()

	// Run Phase 1 Plan so TASK-13 enters NEEDS_YOU
	planBody, _ := json.Marshal(map[string]any{
		"task_key": "TASK-13",
		"action":   "plan",
	})
	resPlan, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(planBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/action plan TASK-13: %v", err)
	}
	var dashPlanned DashboardResponse
	_ = json.NewDecoder(resPlan.Body).Decode(&dashPlanned)
	_ = resPlan.Body.Close()
	if dashPlanned.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected TASK-13 to be at NEEDS_YOU after plan, got %q", dashPlanned.ActiveTaskStage)
	}

	// Send Telegram Webhook simulating a reply / channel_post in supergroup -1004390214071 with short "duyệt" replying to TASK-13 notification
	webhookRaw := []byte(`{
		"channel_post": {
			"text": "duyệt",
			"chat": {"id": -1004390214071, "title": "AI Orchestrator Supergroup"},
			"sender_chat": {"title": "trainguyen"},
			"reply_to_message": {"text": "🚀 [AI-ORCHESTRATOR · S3 PLAN READY] Task: TASK-13"}
		}
	}`)
	resWH, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(webhookRaw))
	if err != nil {
		t.Fatalf("POST /api/telegram/webhook for TASK-13: %v", err)
	}
	if resWH.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resWH.Body)
		_ = resWH.Body.Close()
		t.Fatalf("expected 200 OK from /api/telegram/webhook, got %d: %s", resWH.StatusCode, string(rawErr))
	}
	var whResp struct {
		OK            bool              `json:"ok"`
		TriggeredTask string            `json:"triggered_task"`
		NewStage      string            `json:"new_stage"`
		Dashboard     DashboardResponse `json:"dashboard"`
	}
	if err := json.NewDecoder(resWH.Body).Decode(&whResp); err != nil {
		t.Fatalf("decode webhook response: %v", err)
	}
	_ = resWH.Body.Close()

	if !whResp.OK || whResp.TriggeredTask != "TASK-13" {
		t.Fatalf("expected webhook to trigger TASK-13, got ok=%v task=%q", whResp.OK, whResp.TriggeredTask)
	}
	if whResp.NewStage != "DONE" {
		t.Fatalf("expected TASK-13 to reach DONE on first Telegram APPROVE without tripping Circuit Breaker, got stage=%q\nlogs:\n%s",
			whResp.NewStage, strings.Join(whResp.Dashboard.ActiveTaskLogs, "\n"))
	}
	if whResp.Dashboard.ActiveSelfHealState.CircuitOpen || whResp.Dashboard.ActiveSelfHealState.RetryCount != 0 {
		t.Fatalf("expected ActiveSelfHealState CircuitOpen=false and RetryCount=0, got %+v", whResp.Dashboard.ActiveSelfHealState)
	}
	for _, st := range whResp.Dashboard.ActiveSubTasks {
		if st.Stage != "DONE" || st.Status != "done" {
			t.Fatalf("expected Sub-Task %s to be DONE, got stage=%q status=%q", st.SubTaskID, st.Stage, st.Status)
		}
	}
}

func TestTelegramTaskListCommand_ClickToRun_And_RealRepoQueueManagement(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify /task command on Telegram lists current tasks and emits OUTBOUND_TASK_LIST event
	taskCmdPayload := []byte(`{
		"message": {
			"text": "/task",
			"chat": {"id": -1004390214071, "title": "AI Orchestrator Supergroup"},
			"from": {"username": "trainguyen"}
		}
	}`)
	resCmd, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(taskCmdPayload))
	if err != nil {
		t.Fatalf("POST /api/telegram/webhook (/task): %v", err)
	}
	if resCmd.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resCmd.Body)
		_ = resCmd.Body.Close()
		t.Fatalf("expected 200 OK from /task webhook, got %d: %s", resCmd.StatusCode, string(rawErr))
	}
	var cmdResp struct {
		OK               bool                `json:"ok"`
		TelegramSettings TelegramSettingsDTO `json:"telegram_settings"`
		Dashboard        DashboardResponse   `json:"dashboard"`
	}
	if err := json.NewDecoder(resCmd.Body).Decode(&cmdResp); err != nil {
		t.Fatalf("decode /task webhook response: %v", err)
	}
	_ = resCmd.Body.Close()

	foundTaskListEvent := false
	for _, ev := range cmdResp.TelegramSettings.RecentEvents {
		if ev.Direction == "OUTBOUND_TASK_LIST" && strings.Contains(ev.Summary, "/task") {
			foundTaskListEvent = true
			break
		}
	}
	if !foundTaskListEvent {
		t.Fatalf("expected OUTBOUND_TASK_LIST event in telegram_settings.recent_events, got %+v", cmdResp.TelegramSettings.RecentEvents)
	}

	// 2. Verify clicking an Inline Keyboard button ("RUN_TASK TASK-2") selects TASK-2 and runs it immediately
	clickRunPayload := []byte(`{
		"step_delay_ms": 0,
		"callback_query": {
			"id": "cbq-run-task-2",
			"data": "RUN_TASK TASK-2",
			"from": {"username": "trainguyen"},
			"message": {
				"text": "📋 [AI-ORCHESTRATOR · HÀNG ĐỢI TASK HIỆN TẠI]",
				"chat": {"id": -1004390214071}
			}
		}
	}`)
	resClick, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(clickRunPayload))
	if err != nil {
		t.Fatalf("POST /api/telegram/webhook (RUN_TASK TASK-2): %v", err)
	}
	if resClick.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resClick.Body)
		_ = resClick.Body.Close()
		t.Fatalf("expected 200 OK from RUN_TASK TASK-2 callback, got %d: %s", resClick.StatusCode, string(rawErr))
	}
	var clickResp struct {
		OK               bool                `json:"ok"`
		TriggeredTask    string              `json:"triggered_task"`
		NewStage         string              `json:"new_stage"`
		TelegramSettings TelegramSettingsDTO `json:"telegram_settings"`
		Dashboard        DashboardResponse   `json:"dashboard"`
	}
	if err := json.NewDecoder(resClick.Body).Decode(&clickResp); err != nil {
		t.Fatalf("decode RUN_TASK TASK-2 response: %v", err)
	}
	_ = resClick.Body.Close()

	if !clickResp.OK || clickResp.TriggeredTask != "TASK-2" {
		t.Fatalf("expected RUN_TASK callback to trigger TASK-2, got ok=%v triggered=%q", clickResp.OK, clickResp.TriggeredTask)
	}
	if clickResp.Dashboard.ActiveTaskKey != "TASK-2" {
		t.Fatalf("expected ActiveTaskKey to switch to TASK-2 after clicking button, got %q", clickResp.Dashboard.ActiveTaskKey)
	}

	foundRunEvent := false
	for _, ev := range clickResp.TelegramSettings.RecentEvents {
		if ev.Direction == "INBOUND_RUN_TASK" && ev.TaskKey == "TASK-2" {
			foundRunEvent = true
			break
		}
	}
	if !foundRunEvent {
		t.Fatalf("expected INBOUND_RUN_TASK event for TASK-2, got %+v", clickResp.TelegramSettings.RecentEvents)
	}

	// 3. Verify WaveNodeDTO queue & sub-task metadata is populated for Repo Queue Management
	foundWaveNodeMeta := false
	for _, w := range clickResp.Dashboard.Waves {
		for _, n := range w.Nodes {
			if n.SubTasksTotal > 0 {
				foundWaveNodeMeta = true
			}
		}
	}
	if !foundWaveNodeMeta {
		t.Fatalf("expected WaveNodeDTO SubTasksTotal > 0 in dashboard waves")
	}
}

func TestCDCRecon01_MultiRepoDiffMerge_TargetedGoCheck_AndWaveCascade(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. Verify mergeUnifiedDiffPatches preserves hunks from multiple repos without overwriting
	patchA := "diff --git a/internal/model/shadow/cdc_event.go b/internal/model/shadow/cdc_event.go\n--- a/internal/model/shadow/cdc_event.go\n+++ b/internal/model/shadow/cdc_event.go\n@@ -1,3 +1,4 @@\n+// shadow hunk\n"
	patchB := "diff --git a/src/pages/MasterRegistry.tsx b/src/pages/MasterRegistry.tsx\n--- a/src/pages/MasterRegistry.tsx\n+++ b/src/pages/MasterRegistry.tsx\n@@ -1,3 +1,4 @@\n+// web hunk\n"
	merged := mergeUnifiedDiffPatches(patchA, patchB)
	if !strings.Contains(merged, "b/internal/model/shadow/cdc_event.go") || !strings.Contains(merged, "b/src/pages/MasterRegistry.tsx") {
		t.Fatalf("expected mergeUnifiedDiffPatches to preserve both hunks, got:\n%s", merged)
	}

	// 2. Create CDC-RECON-01 across 3 repos (centralized-data-service, cdc-cms-service, cdc-cms-web)
	createBody, _ := json.Marshal(map[string]any{
		"external_id": "CDC-RECON-01",
		"title":       "Audit vì sao khi chạy recon shadow -> master thì kết quả không đúng",
		"description": "Kiểm tra luồng recon shadow -> master trên centralized-data-service, cdc-cms-service, cdc-cms-web",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"target_repos": []string{
			"/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service",
			"/Users/trainguyen/Documents/work/data-hub/cdc-cms-web",
		},
		"epic":       "DATA",
		"milestone":  "M2",
		"complexity": "high",
	})
	resCreate, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks CDC-RECON-01: %v", err)
	}
	var dashCreate DashboardResponse
	if err := json.NewDecoder(resCreate.Body).Decode(&dashCreate); err != nil {
		t.Fatalf("decode create CDC-RECON-01: %v", err)
	}
	_ = resCreate.Body.Close()

	if len(dashCreate.ActiveSubTasks) != 6 {
		t.Fatalf("expected 6 Sub-Tasks for 3-repo CDC-RECON-01, got %d", len(dashCreate.ActiveSubTasks))
	}

	// 3. Verify buildTargetedRepoCheckCommand excludes scratch/ and docs/ on external data-hub repos
	cdsCmd := srv.buildTargetedRepoCheckCommand("/Users/trainguyen/Documents/work/data-hub/centralized-data-service", dashCreate.ActiveSubTasks)
	if !strings.HasPrefix(cdsCmd, "go build ./internal/") || strings.Contains(cdsCmd, "scratch") || strings.Contains(cdsCmd, "./...") {
		t.Fatalf("expected targeted go build ./internal/... command for centralized-data-service, got %q", cdsCmd)
	}

	// 4. Directly approve Wave 1 Sub-Task (TASK-AUDI03) and verify Wave 0 (TASK-AUDI01, TASK-AUDI02) auto-cascades to DONE without Wave Dependency Gate BLOCK
	approveW1Body, _ := json.Marshal(map[string]any{
		"task_key":    "CDC-RECON-01",
		"sub_task_id": "TASK-AUDI03",
		"action":      "subtask_approve",
	})
	resW1, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(approveW1Body))
	if err != nil {
		t.Fatalf("POST subtask_approve TASK-AUDI03: %v", err)
	}
	var dashW1 DashboardResponse
	if err := json.NewDecoder(resW1.Body).Decode(&dashW1); err != nil {
		t.Fatalf("decode subtask_approve TASK-AUDI03: %v", err)
	}
	_ = resW1.Body.Close()

	for _, st := range dashW1.ActiveSubTasks {
		if (st.SubTaskID == "TASK-AUDI01" || st.SubTaskID == "TASK-AUDI02") && st.Stage != "DONE" {
			t.Fatalf("expected Wave 0 %s to auto-cascade to DONE when approving Wave 1 TASK-AUDI03, got Stage=%q", st.SubTaskID, st.Stage)
		}
		if st.SubTaskID == "TASK-AUDI03" {
			if st.Stage != "APPROVED" || !st.InputContract.UpstreamReady {
				t.Fatalf("expected Wave 1 TASK-AUDI03 Stage=APPROVED and UpstreamReady=true, got Stage=%q UpstreamReady=%v", st.Stage, st.InputContract.UpstreamReady)
			}
		}
	}

	// 5. Run full approve_and_run on CDC-RECON-01 and verify all 6/6 Sub-Tasks and parent task reach DONE
	runAllBody, _ := json.Marshal(map[string]any{
		"task_key": "CDC-RECON-01",
		"action":   "approve_and_run",
	})
	resRunAll, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(runAllBody))
	if err != nil {
		t.Fatalf("POST approve_and_run CDC-RECON-01: %v", err)
	}
	var dashDone DashboardResponse
	if err := json.NewDecoder(resRunAll.Body).Decode(&dashDone); err != nil {
		t.Fatalf("decode approve_and_run CDC-RECON-01: %v", err)
	}
	_ = resRunAll.Body.Close()

	if dashDone.ActiveTaskStage != "DONE" {
		t.Fatalf("expected CDC-RECON-01 ActiveTaskStage=DONE after approve_and_run, got %q", dashDone.ActiveTaskStage)
	}
	for _, st := range dashDone.ActiveSubTasks {
		if st.Stage != "DONE" || st.Status != "done" {
			t.Fatalf("expected Sub-Task %s Stage=DONE Status=done, got Stage=%q Status=%q", st.SubTaskID, st.Stage, st.Status)
		}
	}
}

func TestTelegramApprovalDossier_FullRootCauseSolutionChanges_AndInteractiveFeedbackRevision(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	taskKey := "tích hợp Master MongoDB Destination"
	createBody, _ := json.Marshal(map[string]any{
		"external_id": taskKey,
		"title":       "Tích hợp Master MongoDB Destination đồng bộ trực tiếp từ Shadow sang MongoDB",
		"description": "Hỗ trợ dest_type=mongodb trong Transmuter, CoerceToBSONValue và Ordered BulkWrite OCC",
		"target_repo": "/Users/trainguyen/Documents/work/data-hub/centralized-data-service",
		"epic":        "DATA",
		"milestone":   "M2",
		"complexity":  "high",
	})
	resCreate, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks (%s): %v", taskKey, err)
	}
	_ = resCreate.Body.Close()

	// Run plan so the task enters NEEDS_YOU (⏸ CHỜ APPROVE)
	planBody, _ := json.Marshal(map[string]any{
		"task_key": taskKey,
		"action":   "plan",
	})
	resPlan, err := http.Post(ts.URL+"/api/tasks/action", "application/json", bytes.NewReader(planBody))
	if err != nil {
		t.Fatalf("POST /api/tasks/action plan: %v", err)
	}
	var dashPlan DashboardResponse
	if err := json.NewDecoder(resPlan.Body).Decode(&dashPlan); err != nil {
		t.Fatalf("decode plan response: %v", err)
	}
	_ = resPlan.Body.Close()
	if dashPlan.ActiveTaskStage != "NEEDS_YOU" {
		t.Fatalf("expected %s to enter NEEDS_YOU after plan, got %q", taskKey, dashPlan.ActiveTaskStage)
	}

	qt, err := srv.findQueuedTask(context.Background(), taskKey)
	if err != nil {
		t.Fatalf("findQueuedTask(%q): %v", taskKey, err)
	}
	refByID := fmt.Sprintf("#%d", qt.ID)
	isoMeta, _ := srv.ensureTaskIsolation(context.Background(), *qt)
	subTasks := srv.loadOrGenerateSubTasks(*qt)
	initialSubCount := len(subTasks)

	// 1. Verify buildTelegramApprovalRequiredMessage renders the canonical 4-Section Engineering Manager H4b template
	approvalMsg, rows := srv.buildTelegramApprovalRequiredMessage(context.Background(), *qt, isoMeta, subTasks, "S3 Gate Stop")
	for _, mustContain := range []string{
		"⏸ **[H4b GATE · DUYỆT GIẢI PHÁP S3]**",
		"🏗 **1. KIẾN TRÚC & CHIẾN LƯỢC CHUYỂN ĐỔI (ARCHITECTURE PLAN)**",
		"⚠️ **2. KIỂM SOÁT RỦI RO (RISK & GUARDRAILS)**",
		"💻 **3. KEY DIFF / CODE PREVIEW",
		"📋 **4. LỘ TRÌNH THỰC THI",
		"📁 **Files thay đổi",
		"09_tasks_solution.md",
	} {
		if !strings.Contains(approvalMsg, mustContain) {
			t.Fatalf("expected approval dossier message to contain %q, got:\n%s", mustContain, approvalMsg)
		}
	}
	if len(rows) < 2 {
		t.Fatalf("expected Inline Keyboard rows for Approve and Revise, got %+v", rows)
	}

	// 2. Click `[ ✏️ Phản hồi / Sửa Giải Pháp (#ID) ]` (`REVISE_PROMPT #<ID>`)
	revPromptPayload, _ := json.Marshal(map[string]any{
		"callback_query": map[string]any{
			"id":   "cbq-revise-prompt",
			"data": "REVISE_PROMPT " + refByID,
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": approvalMsg,
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resRevPrompt, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(revPromptPayload))
	if err != nil {
		t.Fatalf("POST REVISE_PROMPT webhook: %v", err)
	}
	var revPromptResp struct {
		OK               bool                `json:"ok"`
		TriggeredTask    string              `json:"triggered_task"`
		TelegramSettings TelegramSettingsDTO `json:"telegram_settings"`
	}
	if err := json.NewDecoder(resRevPrompt.Body).Decode(&revPromptResp); err != nil {
		t.Fatalf("decode REVISE_PROMPT response: %v", err)
	}
	_ = resRevPrompt.Body.Close()
	if revPromptResp.TelegramSettings.PendingRevisionTaskKey != taskKey {
		t.Fatalf("expected PendingRevisionTaskKey=%q after REVISE_PROMPT, got %q", taskKey, revPromptResp.TelegramSettings.PendingRevisionTaskKey)
	}

	// 3. User sends plain-text feedback in the next Telegram message (without /revise prefix)
	feedback1 := "Đổi batch size mặc định từ 100 lên 500 và thêm Sub-Task kiểm tra index _source_ts trên MongoDB"
	plainFeedbackPayload, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"text": feedback1,
			"chat": map[string]any{"id": -1004390214071, "title": "AI Orchestrator Supergroup"},
			"from": map[string]any{"username": "trainguyen"},
		},
	})
	resFB1, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(plainFeedbackPayload))
	if err != nil {
		t.Fatalf("POST plain feedback webhook: %v", err)
	}
	if resFB1.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resFB1.Body)
		_ = resFB1.Body.Close()
		t.Fatalf("expected 200 OK from plain feedback webhook, got %d: %s", resFB1.StatusCode, string(rawErr))
	}
	var fb1Resp struct {
		OK               bool                `json:"ok"`
		TriggeredTask    string              `json:"triggered_task"`
		NewStage         string              `json:"new_stage"`
		TelegramSettings TelegramSettingsDTO `json:"telegram_settings"`
		Dashboard        DashboardResponse   `json:"dashboard"`
	}
	if err := json.NewDecoder(resFB1.Body).Decode(&fb1Resp); err != nil {
		t.Fatalf("decode plain feedback response: %v", err)
	}
	_ = resFB1.Body.Close()

	if fb1Resp.TriggeredTask != taskKey {
		t.Fatalf("expected plain feedback to revise %q, got %q", taskKey, fb1Resp.TriggeredTask)
	}
	if fb1Resp.TelegramSettings.PendingRevisionTaskKey != "" {
		t.Fatalf("expected PendingRevisionTaskKey to be cleared after revision, got %q", fb1Resp.TelegramSettings.PendingRevisionTaskKey)
	}
	if fb1Resp.NewStage != "NEEDS_YOU" {
		t.Fatalf("expected task to stay in NEEDS_YOU after revision, got %q", fb1Resp.NewStage)
	}
	if len(fb1Resp.Dashboard.ActiveSubTasks) != initialSubCount+1 {
		t.Fatalf("expected adding Sub-Task via feedback to increase count from %d to %d, got %d", initialSubCount, initialSubCount+1, len(fb1Resp.Dashboard.ActiveSubTasks))
	}
	// Verify 09_tasks_solution.md on disk contains the user's feedback revision
	updatedPlanRaw, err := os.ReadFile(isoMeta.PlanFile)
	if err != nil {
		t.Fatalf("read updated 09_tasks_solution.md: %v", err)
	}
	if !strings.Contains(string(updatedPlanRaw), feedback1) {
		t.Fatalf("expected 09_tasks_solution.md to contain feedback %q, got:\n%s", feedback1, string(updatedPlanRaw))
	}

	// 4. User also sends a direct swipe-reply (`reply_to_message`) with a second adjustment
	feedback2 := "Bổ sung cơ chế retry backoff 3 lần khi gặp lỗi WriteConflict trên BulkWrite"
	replyFeedbackPayload, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"text": feedback2,
			"chat": map[string]any{"id": -1004390214071},
			"from": map[string]any{"username": "trainguyen"},
			"reply_to_message": map[string]any{
				"text": fmt.Sprintf("⏸ [APPROVAL REQUIRED · HỒ SƠ DUYỆT GIẢI PHÁP S3 (%s)] • Task: `%s` (%s)", refByID, taskKey, refByID),
			},
		},
	})
	resFB2, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(replyFeedbackPayload))
	if err != nil {
		t.Fatalf("POST reply feedback webhook: %v", err)
	}
	if resFB2.StatusCode != http.StatusOK {
		rawErr, _ := io.ReadAll(resFB2.Body)
		_ = resFB2.Body.Close()
		t.Fatalf("expected 200 OK from reply feedback webhook, got %d: %s", resFB2.StatusCode, string(rawErr))
	}
	_ = resFB2.Body.Close()

	updatedPlanRaw2, _ := os.ReadFile(isoMeta.PlanFile)
	if !strings.Contains(string(updatedPlanRaw2), feedback2) {
		t.Fatalf("expected 09_tasks_solution.md to contain reply feedback %q", feedback2)
	}

	// 5. Finally user clicks `[ ✅ Approve & Chạy S4–S5 (#ID) ]` (`APPROVE #<ID>`) and task completes to DONE
	approvePayload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-approve-by-id",
			"data": "APPROVE " + refByID,
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": approvalMsg,
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resApprove, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(approvePayload))
	if err != nil {
		t.Fatalf("POST APPROVE %s webhook: %v", refByID, err)
	}
	var appResp struct {
		OK            bool              `json:"ok"`
		TriggeredTask string            `json:"triggered_task"`
		NewStage      string            `json:"new_stage"`
		Dashboard     DashboardResponse `json:"dashboard"`
	}
	if err := json.NewDecoder(resApprove.Body).Decode(&appResp); err != nil {
		t.Fatalf("decode APPROVE %s response: %v", refByID, err)
	}
	_ = resApprove.Body.Close()

	if !appResp.OK || appResp.TriggeredTask != taskKey || appResp.NewStage != "DONE" {
		t.Fatalf("expected APPROVE %s to complete %q to DONE, got ok=%v triggered=%q stage=%q", refByID, taskKey, appResp.OK, appResp.TriggeredTask, appResp.NewStage)
	}
}

func TestTelegramFractalSubTaskFlow_StepByStepAndApproveAll_AndSidebarUIUX(t *testing.T) {
	srv := newTestWebServer(t)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	taskKey := "tích hợp Master MongoDB Destination"
	var tenSubs []string
	for i := 1; i <= 10; i++ {
		tenSubs = append(tenSubs, fmt.Sprintf("Sub-Task %d: Tích hợp thành phần Master MongoDB bước %d", i, i))
	}

	createBody, _ := json.Marshal(map[string]any{
		"external_id":     taskKey,
		"title":           taskKey,
		"description":     "Tích hợp Master MongoDB Destination qua 10 Sub-Tasks theo 3 Wave (Wave 0 -> Wave 1 -> Wave 2)",
		"target_repo":     srv.Root,
		"target_repos":    []string{srv.Root},
		"sub_task_titles": tenSubs,
		"complexity":      "high",
		"runner":          "antigravity",
		"auto_autopilot":  true,
	})
	resCreate, err := http.Post(ts.URL+"/api/tasks", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/tasks: %v", err)
	}
	_ = resCreate.Body.Close()

	qt, err := srv.findQueuedTask(ctx, taskKey)
	if err != nil {
		t.Fatalf("findQueuedTask(%q): %v", taskKey, err)
	}
	refByID := fmt.Sprintf("#%d", qt.ID)
	isoMeta, _ := srv.ensureTaskIsolation(ctx, *qt)
	subTasks := srv.loadOrGenerateSubTasks(*qt)
	if len(subTasks) != 10 {
		t.Fatalf("expected 10 Sub-Tasks, got %d", len(subTasks))
	}

	// 1. Verify S3 Approval Required message exposes BOTH execution modes: APPROVE_ALL and STEP_SUBTASKS
	approvalMsg, approvalRows := srv.buildTelegramApprovalRequiredMessage(ctx, *qt, isoMeta, subTasks, "S3 Plan Ready")
	if !strings.Contains(approvalMsg, "Approve All") || !strings.Contains(approvalMsg, "Step-by-Step") {
		t.Fatalf("expected approval message to explain both Approve All and Step-by-Step Sub-Task modes, got:\n%s", approvalMsg)
	}
	rowsRaw, _ := json.Marshal(approvalRows)
	if !strings.Contains(string(rowsRaw), "APPROVE_ALL "+refByID) || !strings.Contains(string(rowsRaw), "STEP_SUBTASKS "+refByID) {
		t.Fatalf("expected inline keyboard to contain both APPROVE_ALL %s and STEP_SUBTASKS %s, got: %s", refByID, refByID, string(rowsRaw))
	}

	// 2. User activates Step-by-Step Sub-Task Review Mode (`STEP_SUBTASKS #<ID>`)
	stepPayload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-step-subtasks",
			"data": "STEP_SUBTASKS " + refByID,
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": approvalMsg,
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resStep, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(stepPayload))
	if err != nil {
		t.Fatalf("POST STEP_SUBTASKS webhook: %v", err)
	}
	_ = resStep.Body.Close()

	if _, err := os.Stat(filepath.Join(isoMeta.ArtifactDir, ".step-by-step-subtasks")); err != nil {
		t.Fatalf("expected .step-by-step-subtasks marker to exist after STEP_SUBTASKS: %v", err)
	}
	subTasksAfterStep := srv.loadOrGenerateSubTasks(*qt)
	firstSubID := subTasksAfterStep[0].SubTaskID
	secondSubID := subTasksAfterStep[1].SubTaskID
	if subTasksAfterStep[0].Stage != "NEEDS_YOU" {
		t.Fatalf("expected Sub-Task 1/10 (%s) to be at NEEDS_YOU, got %s", firstSubID, subTasksAfterStep[0].Stage)
	}

	// 3. User revises Sub-Task 1/10's plan via REVISE_SUB_PROMPT #<ID> <firstSubID>
	revSubPromptPayload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-revise-sub-1",
			"data": fmt.Sprintf("REVISE_SUB_PROMPT %s %s", refByID, firstSubID),
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": "Sub-Task 1/10 review",
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resRevSubP, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(revSubPromptPayload))
	if err != nil {
		t.Fatalf("POST REVISE_SUB_PROMPT webhook: %v", err)
	}
	_ = resRevSubP.Body.Close()

	subFeedback := "Bổ sung validate URI kết nối Master MongoDB và ping timeout 5s trong Sub-Task 1"
	subFBPayload, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"text": subFeedback,
			"chat": map[string]any{"id": -1004390214071},
			"from": map[string]any{"username": "trainguyen"},
		},
	})
	resSubFB, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(subFBPayload))
	if err != nil {
		t.Fatalf("POST Sub-Task feedback webhook: %v", err)
	}
	_ = resSubFB.Body.Close()

	sub1PlanPath := filepath.Join(srv.Root, ".orchestrator", "tasks", sanitizeTaskKeyPath(taskKey), "subtasks", sanitizeTaskKeyPath(firstSubID), "09_tasks_solution.md")
	sub1PlanBytes, _ := os.ReadFile(sub1PlanPath)
	if !strings.Contains(string(sub1PlanBytes), subFeedback) {
		t.Fatalf("expected Sub-Task 1/10 plan (%s) to contain feedback %q", sub1PlanPath, subFeedback)
	}

	// 4. User approves ONLY Sub-Task 1/10 (`APPROVE_SUB #<ID> <firstSubID>`)
	// Verify Parent Task stays at NEEDS_YOU (Hard Parent S4 -> S5 Wave Barrier!) and Sub-Task 2/10 advances to NEEDS_YOU!
	appSub1Payload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-approve-sub-1",
			"data": fmt.Sprintf("APPROVE_SUB %s %s", refByID, firstSubID),
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": "Sub-Task 1/10 review",
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resAppSub1, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(appSub1Payload))
	if err != nil {
		t.Fatalf("POST APPROVE_SUB 1/10 webhook: %v", err)
	}
	var appSub1Resp struct {
		OK       bool   `json:"ok"`
		NewStage string `json:"new_stage"`
	}
	_ = json.NewDecoder(resAppSub1.Body).Decode(&appSub1Resp)
	_ = resAppSub1.Body.Close()

	if appSub1Resp.NewStage != "NEEDS_YOU" {
		t.Fatalf("Parent Task MUST stay at NEEDS_YOU when only 1/10 Sub-Tasks are DONE (Hard Wave Barrier), got %q", appSub1Resp.NewStage)
	}
	subsAfter1 := srv.loadOrGenerateSubTasks(*qt)
	if !strings.EqualFold(subsAfter1[0].Status, "done") {
		t.Fatalf("expected Sub-Task 1/10 (%s) to be done, got %s", firstSubID, subsAfter1[0].Status)
	}
	if subsAfter1[1].Stage != "NEEDS_YOU" {
		t.Fatalf("expected Sub-Task 2/10 (%s) to advance to NEEDS_YOU for review, got %s", secondSubID, subsAfter1[1].Stage)
	}
	for i := 2; i < 10; i++ {
		if strings.EqualFold(subsAfter1[i].Status, "done") {
			t.Fatalf("expected Sub-Task %d/10 (%s) to remain pending while only 1/10 was approved, got %s", i+1, subsAfter1[i].SubTaskID, subsAfter1[i].Status)
		}
	}

	// 5. User approves Sub-Task 2/10 (`APPROVE_SUB #<ID> <secondSubID>`) -> 2/10 DONE, Sub-Task 3/10 becomes NEEDS_YOU
	appSub2Payload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-approve-sub-2",
			"data": fmt.Sprintf("APPROVE_SUB %s %s", refByID, secondSubID),
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": "Sub-Task 2/10 review",
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resAppSub2, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(appSub2Payload))
	if err != nil {
		t.Fatalf("POST APPROVE_SUB 2/10 webhook: %v", err)
	}
	_ = resAppSub2.Body.Close()

	subsAfter2 := srv.loadOrGenerateSubTasks(*qt)
	if !strings.EqualFold(subsAfter2[1].Status, "done") || subsAfter2[2].Stage != "NEEDS_YOU" {
		t.Fatalf("expected Sub-Task 2/10 to be done and Sub-Task 3/10 at NEEDS_YOU, got sub2=%s sub3=%s", subsAfter2[1].Status, subsAfter2[2].Stage)
	}

	// 6. User clicks `[ ⚡ Approve All Còn Lại (Chạy hết 8/10 -> S5) ]` (`APPROVE_ALL #<ID>`)
	// Verify all 10/10 Sub-Tasks complete Wave-by-Wave AND Parent Task unlocks S5 Audit -> DONE!
	appAllPayload, _ := json.Marshal(map[string]any{
		"step_delay_ms": 0,
		"callback_query": map[string]any{
			"id":   "cbq-approve-all-remaining",
			"data": "APPROVE_ALL " + refByID,
			"from": map[string]any{"username": "trainguyen"},
			"message": map[string]any{
				"text": "Sub-Task 3/10 review",
				"chat": map[string]any{"id": -1004390214071},
			},
		},
	})
	resAppAll, err := http.Post(ts.URL+"/api/telegram/webhook", "application/json", bytes.NewReader(appAllPayload))
	if err != nil {
		t.Fatalf("POST APPROVE_ALL webhook: %v", err)
	}
	var appAllResp struct {
		OK       bool   `json:"ok"`
		NewStage string `json:"new_stage"`
	}
	_ = json.NewDecoder(resAppAll.Body).Decode(&appAllResp)
	_ = resAppAll.Body.Close()

	if !appAllResp.OK || appAllResp.NewStage != "DONE" {
		t.Fatalf("expected APPROVE_ALL %s to complete all 10/10 Sub-Tasks and Parent S5 to DONE, got ok=%v stage=%q", refByID, appAllResp.OK, appAllResp.NewStage)
	}
	subsFinal := srv.loadOrGenerateSubTasks(*qt)
	for i, st := range subsFinal {
		if !strings.EqualFold(st.Status, "done") {
			t.Fatalf("expected Sub-Task %d/10 (%s) to be done after APPROVE_ALL, got %s", i+1, st.SubTaskID, st.Status)
		}
	}

	// 7. Verify Sidebar UI/UX static assets (310px sidebar, mini-tg-btn, step_subtasks & approve_all buttons)
	resIndex, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	indexBytes, _ := io.ReadAll(resIndex.Body)
	_ = resIndex.Body.Close()
	if !strings.Contains(string(indexBytes), "btn-push-tg-task-menu") || !strings.Contains(string(indexBytes), "sb-qs-all") {
		t.Fatalf("expected index.html to contain btn-push-tg-task-menu and unified sb-qs-all filter counter")
	}
}

func TestSubTaskWorkingTreeGitApply_AndScoutMaskSecretsCodePlaceholders(t *testing.T) {
	// 1. Verify scout.MaskSecrets preserves Go/TS format placeholders and variable references while masking real secrets
	codeSnippet := "dsn := fmt.Sprintf(\"postgres://%s:%s@%s:%d/%s\", user, pass, host, port, db)\n" +
		"mongoURI := fmt.Sprintf(\"mongodb://%s:%s@%s:%d/%s\", user, pass, host, port, db)\n" +
		"payload := map[string]any{\"password\": connForm.password, \"secret\": req.Secret}\n"
	maskedCode := scout.MaskSecrets(codeSnippet)
	if !strings.Contains(maskedCode, "postgres://%s:%s@%s:%d/%s") {
		t.Fatalf("expected scout.MaskSecrets to preserve postgres://%%s:%%s@%%s, got: %s", maskedCode)
	}
	if !strings.Contains(maskedCode, "mongodb://%s:%s@%s:%d/%s") {
		t.Fatalf("expected scout.MaskSecrets to preserve mongodb://%%s:%%s@%%s, got: %s", maskedCode)
	}
	if !strings.Contains(maskedCode, "connForm.password,") {
		t.Fatalf("expected scout.MaskSecrets to preserve connForm.password, got: %s", maskedCode)
	}

	realLeak := "dsn=postgres://admin:MySuperSecret99@10.0.0.1:5432/prod password= PlaintextSecret99"
	maskedLeak := scout.MaskSecrets(realLeak)
	if strings.Contains(maskedLeak, "MySuperSecret99") || strings.Contains(maskedLeak, "PlaintextSecret99") {
		t.Fatalf("expected scout.MaskSecrets to redact real credentials, got: %s", maskedLeak)
	}

	// 2. Verify restoreFalsePositiveMaskedCodePlaceholders repairs corrupted unified diff hunks
	corruptedDiff := "diff --git a/conn.go b/conn.go\n--- a/conn.go\n+++ b/conn.go\n@@ -1,3 +1,3 @@\n" +
		"-dsn := fmt.Sprintf(\"postgres://***:***@%s:%d\", u, p, h, port)\n" +
		"+dsn := fmt.Sprintf(\"mongodb://***:***@%s:%d\", u, p, h, port)\n" +
		"+password:***\n"
	repairedDiff := restoreFalsePositiveMaskedCodePlaceholders(corruptedDiff)
	if !strings.Contains(repairedDiff, "postgres://%s:%s@%s:%d") ||
		!strings.Contains(repairedDiff, "mongodb://%s:%s@%s:%d") ||
		!strings.Contains(repairedDiff, "password: connForm.password,\n") {
		t.Fatalf("expected restoreFalsePositiveMaskedCodePlaceholders to repair placeholders, got:\n%s", repairedDiff)
	}

	// 3. Verify tryGitApplySectionToRepo applies unified diff hunk onto a real Git working tree and updates git status
	repoDir := t.TempDir()
	runGitCmd := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
		}
	}
	runGitCmd("init")
	runGitCmd("config", "user.email", "test@example.com")
	runGitCmd("config", "user.name", "Test")

	relFile := filepath.Join("internal", "api", "master", "conn.go")
	absFile := filepath.Join(repoDir, relFile)
	_ = os.MkdirAll(filepath.Dir(absFile), 0o755)
	origContent := "package master\n\nfunc Engine() string {\n\treturn \"postgresql\"\n}\n"
	if err := os.WriteFile(absFile, []byte(origContent), 0o644); err != nil {
		t.Fatalf("write orig file: %v", err)
	}
	runGitCmd("add", ".")
	runGitCmd("commit", "-m", "initial")

	hunk := "diff --git a/internal/api/master/conn.go b/internal/api/master/conn.go\n" +
		"--- a/internal/api/master/conn.go\n" +
		"+++ b/internal/api/master/conn.go\n" +
		"@@ -1,5 +1,5 @@\n" +
		" package master\n" +
		" \n" +
		" func Engine() string {\n" +
		"-\treturn \"postgresql\"\n" +
		"+\treturn \"mongodb\"\n" +
		" }\n"

	ctx := context.Background()
	if !tryGitApplySectionToRepo(ctx, repoDir, hunk) {
		t.Fatalf("expected tryGitApplySectionToRepo to succeed on clean git repo")
	}
	// Idempotent second call must also return true without failing
	if !tryGitApplySectionToRepo(ctx, repoDir, hunk) {
		t.Fatalf("expected second tryGitApplySectionToRepo call to succeed idempotently")
	}

	statusCmd := exec.Command("git", "-C", repoDir, "status", "--porcelain")
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(string(statusOut), "internal/api/master/conn.go") {
		t.Fatalf("expected git status --porcelain to show modified internal/api/master/conn.go, got: %q", string(statusOut))
	}
}
