package webui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/config"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
)

func TestArchitectLLMBridge_SkipDistAndBoilerplateAndRePlanV1V2(t *testing.T) {
	tmpRoot := t.TempDir()
	fakeExportRepo := filepath.Join(tmpRoot, "centralized-export-service")
	if err := os.MkdirAll(filepath.Join(fakeExportRepo, "dist", "models", "trans-his"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fakeExportRepo, "models", "trans-his"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fakeExportRepo, "dbHandler"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Write compiled JS boilerplate inside dist/ (must be 100% skipped by scanner!)
	distJS := `"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {});
};
class TransHisModel {}
`
	if err := os.WriteFile(filepath.Join(fakeExportRepo, "dist", "models", "trans-his", "trans-his.model.js"), []byte(distJS), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Write real TypeScript source files
	srcModelTS := `import mongoose from "mongoose";
export class TransHisModel {
  public async findTransHisV1(filter: Record<string, any>) {
    return [];
  }
}
`
	if err := os.WriteFile(filepath.Join(fakeExportRepo, "models", "trans-his", "trans-his.model.ts"), []byte(srcModelTS), 0o644); err != nil {
		t.Fatal(err)
	}
	srcPgTS := `import { Pool } from "pg";
export class PostgresTransHisV2Repository {
  public async queryTransHisV2(pool: Pool, filter: Record<string, any>) {
    return [];
  }
}
`
	if err := os.WriteFile(filepath.Join(fakeExportRepo, "package.json"), []byte(`{"name":"centralized-export-service","scripts":{"build":"tsc","test":"jest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeExportRepo, "dbHandler", "postgres.ts"), []byte(srcPgTS), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify detectTargetRepoLanguageAndDoD returns typescript + npm run build && npm test for package.json repo
	fenceLang, dodCmd := detectTargetRepoLanguageAndDoD(fakeExportRepo)
	if fenceLang != "typescript" || dodCmd != "npm run build && npm test" {
		t.Fatalf("expected typescript + 'npm run build && npm test', got fence=%q dod=%q", fenceLang, dodCmd)
	}
	// Verify normalizeRelativeRepoFiles strips repo prefix and :line and deduplicates via Set
	dedupedFiles := normalizeRelativeRepoFiles([]string{
		"centralized-export-service/models/trans-his/trans-his.model.ts:10 (class TransHisModel {)",
		"models/trans-his/trans-his.model.ts:25",
		"centralized-export-service/dbHandler/postgres.ts:6",
		"dbHandler/postgres.ts",
	}, "centralized-export-service")
	if len(dedupedFiles) != 2 || dedupedFiles[0] != "models/trans-his/trans-his.model.ts" || dedupedFiles[1] != "dbHandler/postgres.ts" {
		t.Fatalf("unexpected dedupedFiles: %#v", dedupedFiles)
	}

	qStore, err := queue.Open(filepath.Join(tmpRoot, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer qStore.Close()

	srv := NewServer(tmpRoot, "test-ws", "main", config.Config{}, qStore, nil, nil, 326)
	ctx := context.Background()

	taskID, _, err := qStore.EnqueueRaw(ctx, queue.RawTask{
		ExternalID:  "TASK-13",
		Source:      "vibe-ui",
		Title:       "con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret.",
		Description: "con centralizezed-export hiện đang dùng mongo trans_his. giờ tao muốn chuyển qua posgret.",
		Severity:    "high",
		Labels:      []string{"repo:" + fakeExportRepo},
		Complexity:  "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = qStore.UpdateStage(ctx, taskID, queue.StageDone, "gemini-4-argon (antigravity)", "")

	qt, err := srv.findQueuedTask(ctx, "TASK-13")
	if err != nil {
		t.Fatal(err)
	}

	// Verify scanner never touches dist/ or function adopt(value)
	hits, _ := srv.inspectTargetRepoSymbols(fakeExportRepo, qt.ExternalID, qt.Title, qt.Description)
	if len(hits) == 0 {
		t.Fatal("expected at least 1 symbol hit from real TypeScript source")
	}
	for _, h := range hits {
		if strings.Contains(h.RelPath, "dist/") || strings.Contains(h.Signature, "function adopt(") {
			t.Fatalf("scanner leaked dist/ or compiler boilerplate: %+v", h)
		}
	}

	// Verify classifyChatActionMode detects architectural steering V1/V2 prompt as revise_solution
	userSteeringPrompt := "hiện tại mày đang làm là bỏ cái export type cũ đi, đè lên nó. giải pháp của anh là tạo 1 cái type v2 qua postgrest, còn v1 vẫn là cái cũ"
	mode := classifyChatActionMode(userSteeringPrompt, "discuss")
	if mode != "revise_solution" {
		t.Fatalf("expected classifyChatActionMode=%q, got %q", "revise_solution", mode)
	}

	isoMeta, err := srv.ensureTaskIsolation(ctx, *qt)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate stale DONE artifacts prior to steering
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, ".user-approved"), []byte("approved\n"), 0o644)
	_ = os.WriteFile(filepath.Join(isoMeta.ArtifactDir, "diff.patch"), []byte("diff --git a/dist/models/trans-his/trans-his.model.js b/dist/models/trans-his/trans-his.model.js\n"), 0o644)

	cliCfg := srv.loadCLISettings()
	reply, planFile, updatedSubs := srv.executeArchitectReviseSolution(
		ctx, qt, isoMeta, nil, userSteeringPrompt,
		"🧠 BRAIN · Architect", "Slot A", "/brain-delegate", "gemini-4-argon (antigravity)",
		[]string{"L018", "L030"}, cliCfg,
	)

	// 1. Task must be re-opened to NEEDS_YOU (not DONE!)
	reloadedQT, err := srv.findQueuedTask(ctx, "TASK-13")
	if err != nil {
		t.Fatal(err)
	}
	if reloadedQT.Stage != queue.StageNeedsYou {
		t.Fatalf("expected TASK-13 stage to be re-opened to NEEDS_YOU, got %q", reloadedQT.Stage)
	}
	if _, err := os.Stat(filepath.Join(isoMeta.ArtifactDir, ".user-approved")); !os.IsNotExist(err) {
		t.Fatal("expected .user-approved to be revoked on architectural re-plan")
	}

	// 2. All Sub-Tasks must be reset to NEEDS_YOU (0/N DONE) and use dynamic npm run build && npm test DoD
	if len(updatedSubs) == 0 {
		t.Fatal("expected non-empty updated sub-tasks")
	}
	for _, st := range updatedSubs {
		if strings.EqualFold(st.Status, "done") || strings.EqualFold(st.Stage, "DONE") {
			t.Fatalf("sub-task %s must not remain DONE after architectural re-plan: status=%q stage=%q", st.SubTaskID, st.Status, st.Stage)
		}
		if strings.Contains(st.ContextFileLine, "dist/") || strings.Contains(st.ContextFileLine, "function adopt(") {
			t.Fatalf("sub-task %s still references dist/ or boilerplate: %q", st.SubTaskID, st.ContextFileLine)
		}
		if st.TestDoD != "npm run build && npm test" {
			t.Fatalf("expected TypeScript TestDoD 'npm run build && npm test', got %q", st.TestDoD)
		}
	}

	// 3. Reply and 09_tasks_solution.md must include Upfront Summary, address V1 (Mongo) + V2 (PostgresDB with pgPool only), use ```typescript fence, and contain zero Meta-System Noise
	for _, forbidden := range []string{
		"dist/models/",
		"function adopt(value)",
		"Kịch Bản Tái Hiện Lỗi",
		"VerifySubTaskCodeOnDisk",
		"NATS/Kafka Dead Letter Queue",
		"Triển khai kiến trúc & nghiệp vụ cốt lõi cho",
		"```go",
		"Đã quét trực tiếp",
		"private readonly knexPg",
		"Knex/PG Pool",
		"Stage `DONE` | Tất cả Sub-Tasks hiện tại đã hoàn tất (`100% DONE`)",
	} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("architect reply contains forbidden garbage %q:\n%s", forbidden, reply)
		}
	}
	for _, required := range []string{
		"### 📌 Tóm tắt & Trả lời trực tiếp (Summary)",
		"PostgresDB",
		"executeTransHisExport",
		"```typescript",
		"**Stage:** `NEEDS_YOU`",
		"**Sub-Tasks:** `0/2 DONE`",
	} {
		if !strings.Contains(reply, required) {
			t.Fatalf("architect reply missing required content %q:\n%s", required, reply)
		}
	}

	rawPlan, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawPlan), "```typescript") ||
		!strings.Contains(string(rawPlan), "### 📌 Tóm tắt & Trả lời trực tiếp (Summary)") ||
		strings.Contains(string(rawPlan), "Đã quét trực tiếp") ||
		strings.Contains(string(rawPlan), "dist/models/") ||
		strings.Contains(string(rawPlan), "Knex/PG Pool") {
		t.Fatalf("09_tasks_solution.md missing ```typescript / Summary or contains Meta-System Noise / Knex/PG Pool:\n%s", string(rawPlan))
	}
}

func TestRealRepoCentralizedExportService_AndTask13OnDiskRegeneration(t *testing.T) {
	realExportRepo := "/Users/trainguyen/Documents/work/centralized-export-service"
	if st, err := os.Stat(realExportRepo); err != nil || !st.IsDir() {
		t.Fatalf("real repo %s not found on disk: %v", realExportRepo, err)
	}
	fenceLang, dodCmd := detectTargetRepoLanguageAndDoD(realExportRepo)
	if fenceLang != "typescript" || dodCmd != "npm run build && npm test" {
		t.Fatalf("real repo %s: expected typescript + 'npm run build && npm test', got fence=%q dod=%q", realExportRepo, fenceLang, dodCmd)
	}

	orchRoot := "/Users/trainguyen/Documents/work/ai-orchestrator"
	qPath := filepath.Join(orchRoot, ".orchestrator", "queue.db")
	qStore, err := queue.Open(qPath)
	if err != nil {
		t.Fatal(err)
	}
	defer qStore.Close()

	srv := NewServer(orchRoot, "ai-orchestrator", "main", config.Config{}, qStore, nil, nil, 326)
	ctx := context.Background()

	qt, err := srv.findQueuedTask(ctx, "TASK-13")
	if err != nil {
		t.Fatal(err)
	}
	hits, totalFiles := srv.inspectTargetRepoSymbols(realExportRepo, qt.ExternalID, qt.Title, qt.Description)
	if len(hits) == 0 || totalFiles == 0 {
		t.Fatalf("expected non-empty symbol hits on real repo %s (hits=%d totalFiles=%d)", realExportRepo, len(hits), totalFiles)
	}
	for _, h := range hits {
		if strings.Contains(h.RelPath, "dist/") || strings.Contains(h.Signature, "function adopt(") || strings.HasSuffix(h.RelPath, ".d.ts") {
			t.Fatalf("real repo scan leaked build artifact or boilerplate: %+v", h)
		}
	}

	isoMeta, err := srv.ensureTaskIsolation(ctx, *qt)
	if err != nil {
		t.Fatal(err)
	}
	cliCfg := srv.loadCLISettings()
	userSteeringPrompt := "hiện tại mày đang làm là bỏ cái export type cũ đi, đè lên nó. giải pháp của anh là tạo 1 cái type v2 qua postgrest, còn v1 vẫn là cái cũ"
	reply, planFile, updatedSubs := srv.executeArchitectReviseSolution(
		ctx, qt, isoMeta, nil, userSteeringPrompt,
		"🧠 BRAIN · Architect", "Slot A", "/brain-delegate", "gemini-4-argon (antigravity)",
		[]string{"L001", "L018", "L030"}, cliCfg,
	)
	if len(updatedSubs) == 0 {
		t.Fatal("expected updated sub-tasks for real TASK-13")
	}
	for _, st := range updatedSubs {
		if st.TestDoD != "npm run build && npm test" {
			t.Fatalf("real TASK-13 sub-task %s expected TestDoD='npm run build && npm test', got %q", st.SubTaskID, st.TestDoD)
		}
	}
	planBytes, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Knex/PG Pool", "private readonly knexPg", "dist/models/", "function adopt(", "```go", "Đã quét trực tiếp"} {
		if strings.Contains(reply, forbidden) || strings.Contains(string(planBytes), forbidden) {
			t.Fatalf("real TASK-13 reply or 09_tasks_solution.md contains forbidden string %q", forbidden)
		}
	}
}

func TestCDC01IsolationAndNoWorkspaceContamination(t *testing.T) {
	tmpRoot := t.TempDir()
	fakeDataHub := filepath.Join(filepath.Dir(tmpRoot), "data-hub")
	_ = fakeDataHub

	fakeCDS := filepath.Join(tmpRoot, "centralized-data-service")
	if err := os.MkdirAll(filepath.Join(fakeCDS, "internal", "service", "master"), 0o755); err != nil {
		t.Fatal(err)
	}
	transmuterSrc := `package master
type Transmuter struct{}
func (t *Transmuter) RunIncrementalTransmute() error {
	return nil
}
`
	if err := os.WriteFile(filepath.Join(fakeCDS, "internal", "service", "master", "transmuter.go"), []byte(transmuterSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	qStore, err := queue.Open(filepath.Join(tmpRoot, "queue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer qStore.Close()

	srv := NewServer(tmpRoot, "test-ws", "main", config.Config{}, qStore, nil, nil, 326)
	ctx := context.Background()

	// Create cdc-01 whose description mentions "[mongodb]" and "master_payment_bill_service"
	_, _, err = qStore.EnqueueRaw(ctx, queue.RawTask{
		ExternalID:  "cdc-01",
		Source:      "vibe-ui",
		Title:       "kiểm tra cho tao vì sao transmute bị thiếu",
		Description: "kiểm tra cho tao vì sao transmute bị thiếu\nDatabase đích: [mongodb] traitestpbs / payment-bill-service\nBảng Master đích: master_payment_bill_service",
		Severity:    "high",
		Labels:      []string{"repo:" + fakeCDS},
		Complexity:  "high",
	})
	if err != nil {
		t.Fatal(err)
	}

	qt, err := srv.findQueuedTask(ctx, "cdc-01")
	if err != nil {
		t.Fatal(err)
	}

	// 1. FindMatchingAgentWorkspace must NEVER match feat-master-mongo-destination-2026-10-20 for cdc-01
	wsDir, t08, s09, _ := srv.FindMatchingAgentWorkspace(*qt)
	if wsDir != "" || t08 != "" || s09 != "" {
		t.Fatalf("expected FindMatchingAgentWorkspace to return empty for cdc-01, got wsDir=%q t08=%q s09=%q", wsDir, t08, s09)
	}

	// 2. Simulate previously contaminated subtasks.json with TASK-M01..M02 on disk
	isoMeta, err := srv.ensureTaskIsolation(ctx, *qt)
	if err != nil {
		t.Fatal(err)
	}
	contaminatedJSON := `[{"sub_task_id":"TASK-M01","parent_key":"cdc-01","title":"Mở rộng Domain Model & API Master Connection hỗ trợ MongoDB","target_repo_name":"cdc-cms-service","context_file_line":"cdc-cms-service/internal/domain/master.go:10"}]`
	_ = os.WriteFile(srv.subTasksFilePath("cdc-01"), []byte(contaminatedJSON), 0o644)

	// 3. loadOrGenerateSubTasks must detect contamination and decompose clean CDC-01-S01..S02 sub-tasks
	subs := srv.loadOrGenerateSubTasks(*qt)
	if len(subs) == 0 {
		t.Fatal("expected non-empty sub-tasks for cdc-01")
	}
	for _, st := range subs {
		if strings.HasPrefix(st.SubTaskID, "TASK-M") || strings.Contains(st.Title, "Master Connection hỗ trợ MongoDB") {
			t.Fatalf("cdc-01 sub-task is still contaminated with Master Mongo task: %+v", st)
		}
	}

	// 4. renderContextualDiscussReply must never emit "Phương án khắc phục: Workspace**:" or "feat-master-mongo-destination-2026-10-20"
	reply := srv.renderContextualDiscussReply(
		*qt, isoMeta, subs,
		"sao task của tao là kiểm tra vì sao transmute bị thiếu, mày lại làm cái gì vậy",
		"🧠 BRAIN · Architect", "Slot A", "/brain-delegate", "gemini-4-argon (antigravity)",
		nil,
	)
	for _, forbidden := range []string{
		"feat-master-mongo-destination-2026-10-20",
		"Phương án khắc phục: Workspace**:",
		"- **Phương án khắc phục:** ",
		"TASK-M01",
		"Master Connection hỗ trợ MongoDB",
	} {
		if strings.Contains(reply, forbidden) {
			t.Fatalf("cdc-01 discuss reply contains forbidden contamination %q:\n%s", forbidden, reply)
		}
	}

	// 5. findQueuedTask must NEVER silently fall back to another task (e.g. TASK-13) when an explicit taskKey is missing
	if wrongQT, err := srv.findQueuedTask(ctx, "missing-task-999"); err == nil {
		t.Fatalf("expected findQueuedTask('missing-task-999') to return error, but got task %q", wrongQT.ExternalID)
	}
}

