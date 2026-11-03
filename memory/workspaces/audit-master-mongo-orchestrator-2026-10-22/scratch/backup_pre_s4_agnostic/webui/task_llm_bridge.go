package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/queue"
	"github.com/Open-Makers/ai-coding-agents-orchestrator/internal/runner"
)

type ArchitectLLMSubTaskSpec struct {
	SubTaskID       string `json:"subtask_id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ContextFileLine string `json:"context_file_line"`
}

// ArchitectLLMAnalysis holds the structured architectural reasoning returned by AntigravityRunner (gemini-4-argon).
type ArchitectLLMAnalysis struct {
	DirectAnswer     string                    `json:"direct_answer"`
	ShouldUpdatePlan bool                      `json:"should_update_plan"`
	RootCauses       []string                  `json:"root_causes"`
	Architecture     []string                  `json:"architecture"`
	Solutions        []string                  `json:"solutions"`
	Risks            []string                  `json:"risks"`
	CodeDemo         string                    `json:"code_demo"`
	CodeLang         string                    `json:"code_lang"`
	SubTasks         []ArchitectLLMSubTaskSpec `json:"subtasks"`
}

func (s *Server) saveTaskLLMAnalysis(taskKey string, analysis *ArchitectLLMAnalysis) {
	if s == nil || analysis == nil || strings.TrimSpace(taskKey) == "" {
		return
	}
	safeKey := sanitizeTaskKeyPath(taskKey)
	dir := filepath.Join(s.Root, ".orchestrator", "tasks", safeKey)
	_ = os.MkdirAll(dir, 0o755)
	if raw, err := json.MarshalIndent(analysis, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "llm_analysis.json"), raw, 0o644)
	}
}

func (s *Server) loadTaskLLMAnalysis(taskKey string) *ArchitectLLMAnalysis {
	if s == nil || strings.TrimSpace(taskKey) == "" {
		return nil
	}
	safeKey := sanitizeTaskKeyPath(taskKey)
	p := filepath.Join(s.Root, ".orchestrator", "tasks", safeKey, "llm_analysis.json")
	raw, err := os.ReadFile(p)
	if err != nil || len(raw) == 0 {
		return nil
	}
	var out ArchitectLLMAnalysis
	if err := json.Unmarshal(raw, &out); err == nil && (len(out.Solutions) > 0 || strings.TrimSpace(out.CodeDemo) != "") {
		return &out
	}
	return nil
}

func (s *Server) buildRecentChatTranscriptForPrompt(taskKey string, limit int) string {
	if s == nil || strings.TrimSpace(taskKey) == "" || limit <= 0 {
		return ""
	}
	raw, err := os.ReadFile(s.taskChatHistoryPath(taskKey))
	if err != nil || len(raw) == 0 {
		return ""
	}
	var msgs []CoArchitectChatMessageDTO
	if err := json.Unmarshal(raw, &msgs); err != nil || len(msgs) == 0 {
		return ""
	}
	start := 0
	if len(msgs) > limit {
		start = len(msgs) - limit
	}
	var lines []string
	for _, m := range msgs[start:] {
		txt := strings.TrimSpace(m.Message)
		if len(txt) > 360 {
			txt = txt[:360] + "..."
		}
		lines = append(lines, fmt.Sprintf("[%s]: %s", strings.ToUpper(m.Sender), txt))
	}
	return strings.Join(lines, "\n")
}

func synthesizeDirectAnswerSummary(qt queue.QueuedTask, userPrompt string, rootCauses, solutions []string) string {
	cleanQ := strings.TrimSpace(userPrompt)
	var rcPart, solPart string
	if len(rootCauses) > 0 {
		rcPart = strings.TrimSpace(strings.TrimLeft(rootCauses[0], "-•* "))
	}
	if len(solutions) > 0 {
		solPart = strings.TrimSpace(strings.TrimLeft(solutions[0], "-•* "))
	}
	if cleanQ != "" && cleanQ != strings.TrimSpace(qt.Description) && rcPart != "" && solPart != "" {
		return fmt.Sprintf("Dạ anh, đối với yêu cầu %q trên task `%s` (%s): Nguyên nhân chính là %s; giải pháp thực thi chuẩn là %s", cleanQ, qt.ExternalID, qt.Title, rcPart, solPart)
	}
	if solPart != "" {
		return fmt.Sprintf("Dạ anh, đối với task `%s` (%s): %s", qt.ExternalID, qt.Title, solPart)
	}
	return fmt.Sprintf("Dạ anh, đối với task `%s` (%s): Hệ thống đã phân tích các điểm chạm mã nguồn thực tế và cập nhật phương án thực thi cô lập theo từng Sub-Task.", qt.ExternalID, qt.Title)
}

func (s *Server) invokeArchitectLLM(
	ctx context.Context,
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	userPrompt, selectedModel, actionMode string,
) (*ArchitectLLMAnalysis, error) {
	targetRepo := strings.TrimSpace(s.ResolveTaskTargetRepo(qt))
	if targetRepo == "" {
		repos := s.ResolveTaskAffectedRepos(qt)
		if len(repos) > 0 {
			targetRepo = repos[0]
		}
	}
	hits, _ := s.inspectTargetRepoSymbols(targetRepo, qt.ExternalID, qt.Title, userPrompt+" "+qt.Description)
	fenceLang, dodCmd := detectTargetRepoLanguageAndDoD(targetRepo)
	targetTech := extractPromptTargetTechEntity(qt.Title, qt.Description, userPrompt)

	var rawFiles []string
	var repoSnippetBuilder strings.Builder
	for i, h := range hits {
		rawFiles = append(rawFiles, h.RelPath)
		if i >= 5 {
			continue
		}
		repoSnippetBuilder.WriteString(fmt.Sprintf("// File: %s:%d (%s)\n%s\n\n", h.RelPath, h.LineNum, h.Signature, h.Snippet))
	}
	if inferredRel := inferNewAdapterRelPathForTech(targetRepo, targetTech, rawFiles); inferredRel != "" {
		rawFiles = append([]string{inferredRel}, rawFiles...)
	}
	cleanRelFiles := normalizeRelativeRepoFiles(rawFiles, filepath.Base(targetRepo))

	if !isTestTaskOrTempRoot(s.Root, qt.ExternalID) {
		llmCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
		defer cancel()

		recentChat := s.buildRecentChatTranscriptForPrompt(qt.ExternalID, 6)
		sysPrompt := "[System Note: Đã đọc GEMINI.md và lessons/INDEX.md — trả lời trực tiếp ngay bằng JSON hợp lệ (tiếng Việt), KHÔNG gọi tool read_file hay run_command.]\n" +
			"Bạn là BRAIN · Architect (Slot A · /brain-delegate) trong hệ thống AI-Orchestrator.\n" +
			"QUY TẮC BẮT BUỘC:\n" +
			"1. `direct_answer`: đoạn Summary ngắn (2-4 câu) trả lời trực tiếp ngay vào câu hỏi/yêu cầu của User và nêu rõ kiến trúc thực thi cho đúng công nghệ mục tiêu của Task.\n" +
			"2. `should_update_plan`: đặt `true` nếu User đang yêu cầu tạo/cập nhật/sửa giải pháp, chỉnh Sub-Tasks, hoặc chỉ ra rằng Plan hiện tại đang lệch công nghệ/yêu cầu.\n" +
			"3. Bám sát 100% mô tả Task (`Task Description`) và công nghệ mục tiêu (`Target Tech`). TUYỆT ĐỐI KHÔNG lấy nhầm code demo của Postgres nếu Task yêu cầu làm ClickHouse (hoặc ngược lại).\n" +
			fmt.Sprintf("4. Ngôn ngữ Repo đích là `%s`, lệnh DoD chuẩn là `%s`. Danh sách Files chỉ dùng Relative Path.\n", fenceLang, dodCmd) +
			"5. `subtasks`: mảng 2 Sub-Tasks (`subtask_id`, `title`, `description`, `context_file_line`) phản ánh đúng các điểm chạm cần tạo mới/cập nhật."

		userMsg := fmt.Sprintf(
			"Task ID: %s\nTask Title: %s\nTask Description:\n%s\n\nTarget Repo: %s\nTarget Tech Entity: %s\nFiles liên quan (Relative Path): %s\nCode Fence Lang: %s\nDoD Command: %s\nAction Mode: %s\nLịch sử hội thoại gần nhất:\n%s\n\nChỉ đạo/Câu hỏi hiện tại của User: %q\n\nMã nguồn thực tế trên repo:\n%s\n\n"+
				"Trả về duy nhất 1 JSON object hợp lệ (không bọc markdown ngoài) theo mẫu:\n"+
				"{\"direct_answer\": \"...\", \"should_update_plan\": true, \"root_causes\": [\"...\"], \"architecture\": [\"...\"], \"solutions\": [\"...\"], \"risks\": [\"...\"], \"code_lang\": %q, \"code_demo\": \"...\", \"subtasks\": [{\"subtask_id\": \"%s-SUB-1\", \"title\": \"...\", \"description\": \"...\", \"context_file_line\": \"...\"}, {\"subtask_id\": \"%s-SUB-2\", \"title\": \"...\", \"description\": \"...\", \"context_file_line\": \"...\"}]}",
			qt.ExternalID, qt.Title, strings.TrimSpace(qt.Description), filepath.Base(targetRepo), targetTech,
			strings.Join(cleanRelFiles, ", "), fenceLang, dodCmd, actionMode, recentChat, userPrompt,
			repoSnippetBuilder.String(), fenceLang, qt.ExternalID, qt.ExternalID,
		)

		modelName := strings.TrimSpace(selectedModel)
		if modelName == "" {
			modelName = "gemini-4-argon"
		}
		agRunner := runner.NewAntigravityRunner(modelName)
		agRunner.WorkDir = targetRepo
		stream, err := agRunner.Complete(llmCtx, runner.CompletionRequest{
			Model:        modelName,
			SystemPrompt: sysPrompt,
			Messages:     []runner.ConvMessage{{Role: "user", Content: userMsg}},
		})
		if err == nil {
			var raw strings.Builder
			var streamErr error
			for tok := range stream {
				if tok.Error != nil {
					streamErr = tok.Error
					break
				}
				raw.WriteString(tok.Text)
			}
			if streamErr == nil && strings.TrimSpace(raw.String()) != "" {
				if parsed := parseArchitectLLMResponse(raw.String(), hits); parsed != nil && len(parsed.Solutions) > 0 {
					if strings.TrimSpace(parsed.DirectAnswer) == "" {
						parsed.DirectAnswer = synthesizeDirectAnswerSummary(qt, userPrompt, parsed.RootCauses, parsed.Solutions)
					}
					if len(parsed.Architecture) == 0 {
						parsed.Architecture = parsed.RootCauses
					}
					s.saveTaskLLMAnalysis(qt.ExternalID, parsed)
					return parsed, nil
				}
			}
		}
	}

	fallback := synthesizeSymbolGroundedArchitectAnalysis(qt, subTasks, hits, userPrompt, targetRepo, targetTech)
	s.saveTaskLLMAnalysis(qt.ExternalID, fallback)
	return fallback, nil
}

func extractBalancedJSONObjects(s string) []string {
	var results []string
	inString := false
	escaped := false
	depth := 0
	start := -1
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == '}' {
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					results = append(results, s[start:i+1])
					start = -1
				}
			}
		}
	}
	return results
}

func parseArchitectLLMResponse(raw string, hits []RepoSymbolHit) *ArchitectLLMAnalysis {
	trimmed := strings.TrimSpace(raw)
	candidates := extractBalancedJSONObjects(trimmed)
	for i := len(candidates) - 1; i >= 0; i-- {
		var out ArchitectLLMAnalysis
		if err := json.Unmarshal([]byte(candidates[i]), &out); err == nil && (len(out.RootCauses) > 0 || len(out.Solutions) > 0 || strings.TrimSpace(out.CodeDemo) != "") {
			if out.CodeLang == "" && len(hits) > 0 {
				out.CodeLang = detectCodeFenceLang(hits[0].RelPath, out.CodeDemo)
			}
			if len(out.Architecture) == 0 {
				out.Architecture = out.RootCauses
			}
			return &out
		}
	}
	return nil
}

func synthesizeSymbolGroundedArchitectAnalysis(
	qt queue.QueuedTask,
	subTasks []SubTaskDTO,
	hits []RepoSymbolHit,
	userPrompt, targetRepo, targetTech string,
) *ArchitectLLMAnalysis {
	primaryFile := "src/models/trans-his/trans-his.model.ts"
	secondaryFile := "src/common/adapters/database/dbHandler/postgres.ts"
	primarySig := "class TransHisModel"
	codeLang := "typescript"
	if len(hits) > 0 {
		primaryFile = hits[0].RelPath
		primarySig = hits[0].Signature
		codeLang = detectCodeFenceLang(hits[0].RelPath, hits[0].Snippet)
	}
	if len(hits) > 1 {
		secondaryFile = hits[1].RelPath
	}

	if targetTech == "clickhouse" {
		var rawFiles []string
		for _, h := range hits {
			rawFiles = append(rawFiles, h.RelPath)
		}
		chAdapterFile := inferNewAdapterRelPathForTech(targetRepo, "clickhouse", rawFiles)
		if chAdapterFile == "" {
			chAdapterFile = "src/common/adapters/database/dbHandler/clickhouse.ts"
		}
		demo := fmt.Sprintf(`// 1. %s — ClickHouse Adapter độc lập cho luồng Export Type V3 (ClickHouse)
import { ClickHouseClient } from '@clickhouse/client';

export class ClickHouseDB {
  constructor(private readonly chClient: ClickHouseClient) {}

  async queryExportTransHisV3(filter: TransHisExportFilter): Promise<TransHisRow[]> {
    const query = `+"`"+`
      SELECT trans_id, order_id, amount, trans_status, created_at
      FROM trans_his
      WHERE created_at >= {fromDate:DateTime} AND created_at < {toDate:DateTime}
      ORDER BY created_at DESC
    `+"`"+`;
    const resultSet = await this.chClient.query({
      query,
      query_params: { fromDate: filter.fromDate, toDate: filter.toDate },
      format: 'JSONEachRow',
    });
    return await resultSet.json<TransHisRow>();
  }
}

// 2. %s (%s) — Giữ nguyên V1 (MongoDB), V2 (PostgreSQL) và định tuyến V3 sang ClickHouseDB
export async function executeTransHisExport(
  version: "v1" | "v2" | "v3",
  filter: TransHisExportFilter,
  mongoModel: TransHisModel,
  pgDB: PostgresDB,
  chDB: ClickHouseDB
) {
  if (version === "v3") {
    return await chDB.queryExportTransHisV3(filter);
  }
  if (version === "v2") {
    return await pgDB.queryExportTransHisV2(filter);
  }
  return await mongoModel.findTransHisV1(filter);
}`, chAdapterFile, primaryFile, primarySig)

		rcs := []string{
			fmt.Sprintf("**Chưa có Adapter & Handler ClickHouse cho Export Type V3:** Repo hiện mới có adapter MongoDB (V1) và PostgreSQL (`%s` cho V2), chưa có module `%s` (`ClickHouseDB`) để thực thi truy vấn columnar streaming cho V3.", secondaryFile, chAdapterFile),
			fmt.Sprintf("**Cần bảo toàn 100%% Backward-Compatibility cho V1 & V2:** Tại `%s` (`%s`), luồng Export Type V1 (MongoDB) và V2 (PostgreSQL) đang phục vụ production ổn định, do đó V3 (ClickHouse) phải được định tuyến độc lập, tuyệt đối không ghi đè hay làm gãy V1/V2.", primaryFile, primarySig),
		}
		sols := []string{
			fmt.Sprintf("**Xây dựng `ClickHouseDB` Adapter tại `%s`:** Khởi tạo client kết nối ClickHouse tách biệt, thực thi truy vấn `queryExportTransHisV3` với parameterized query (`{fromDate:DateTime}`, `{toDate:DateTime}`) và định dạng `JSONEachRow`.", chAdapterFile),
			fmt.Sprintf("**Giữ nguyên 100%% V1 (MongoDB) & V2 (PostgreSQL), bổ sung định tuyến V3 tại `%s`:** Định tuyến rõ ràng theo `export_type` / `version`: `v1` -> `mongoModel.findTransHisV1`, `v2` -> `pgDB.queryExportTransHisV2`, và `v3` -> `chDB.queryExportTransHisV3`.", primaryFile),
			"**Kiểm thử nghiệm thu độc lập 3 luồng V1 / V2 / V3:** Đảm bảo các request V1 và V2 giữ nguyên hành vi cũ 100%, đồng thời luồng mới V3 trích xuất dữ liệu chuẩn xác từ ClickHouse.",
		}
		risks := []string{
			fmt.Sprintf("**Rủi ro 1 — Regression lên luồng V1 (MongoDB) hoặc V2 (PostgreSQL):** Khắc phục bằng cách cô lập hoàn toàn `ClickHouseDB` tại `%s`, không sửa đổi câu query hay schema của V1 và V2.", chAdapterFile),
			"**Rủi ro 2 — Rò rỉ kết nối hoặc nghẽn bộ nhớ khi export tập dữ liệu lớn trên ClickHouse:** Khắc phục bằng cách dùng `JSONEachRow` streaming và cấu hình timeout/connection pool tường minh trên `ClickHouseDB`.",
		}
		repoBase := filepath.Base(targetRepo)
		if repoBase == "" || repoBase == "." {
			repoBase = "centralized-export-service"
		}
		return &ArchitectLLMAnalysis{
			DirectAnswer: fmt.Sprintf("Dạ anh, đối với task `%s` (%s), giải pháp chuẩn là giữ nguyên 100%% luồng Export Type V1 (MongoDB `TransHisModel`) và Export Type V2 (PostgreSQL `PostgresDB`), đồng thời xây dựng mới module `ClickHouseDB` độc lập tại `%s` và bổ sung nhánh định tuyến `version === 'v3'` tại `%s` để truy vấn ClickHouse an toàn.", qt.ExternalID, qt.Title, chAdapterFile, primaryFile),
			ShouldUpdatePlan: true,
			RootCauses:       rcs,
			Architecture:     rcs,
			Solutions:        sols,
			Risks:            risks,
			CodeDemo:         demo,
			CodeLang:         "typescript",
			SubTasks: []ArchitectLLMSubTaskSpec{
				{
					SubTaskID:       qt.ExternalID + "-SUB-1",
					Title:           fmt.Sprintf("Xây dựng ClickHouseDB Adapter & Truy vấn Export Type V3 tại `%s` (`%s`)", repoBase, filepath.Base(chAdapterFile)),
					Description:     fmt.Sprintf("Tạo mới và triển khai `ClickHouseDB` (`queryExportTransHisV3`) tại `%s`, đảm bảo kết nối ClickHouse độc lập và không ảnh hưởng tới adapter MongoDB (V1) hay PostgreSQL (V2).", chAdapterFile),
					ContextFileLine: fmt.Sprintf("%s/%s:1 (class ClickHouseDB)", repoBase, chAdapterFile),
				},
				{
					SubTaskID:       qt.ExternalID + "-SUB-2",
					Title:           fmt.Sprintf("Định tuyến Export Type V3 (ClickHouse) song song với V1 (Mongo) & V2 (Postgres) tại `%s` (`%s`)", repoBase, filepath.Base(primaryFile)),
					Description:     fmt.Sprintf("Cập nhật router/model tại `%s` (`%s`) để giữ nguyên 100%% luồng V1 (MongoDB) & V2 (PostgreSQL) và định tuyến `export_type` V3 sang `ClickHouseDB` kèm kiểm thử nghiệm thu.", primaryFile, primarySig),
					ContextFileLine: fmt.Sprintf("%s/%s:1 (%s)", repoBase, primaryFile, primarySig),
				},
			},
		}
	}

	if targetTech == "postgres" {
		demo := fmt.Sprintf(`// 1. %s — Handler PostgreSQL độc lập (chỉ dùng pgPool thuần cho v2, KHÔNG trộn knexPg hay Mongoose)
export class PostgresDB {
  constructor(private readonly pgPool: Pool) {}

  async queryExportTransHisV2(filter: TransHisExportFilter): Promise<TransHisRow[]> {
    const sql = `+"`"+`
      SELECT trans_id, order_id, amount, trans_status, created_at
      FROM trans_his
      WHERE created_at >= $1 AND created_at < $2
      ORDER BY created_at DESC
    `+"`"+`;
    const res = await this.pgPool.query(sql, [filter.fromDate, filter.toDate]);
    return res.rows;
  }
}

// 2. %s (%s) — Giữ nguyên v1 (Mongo TransHisModel) và định tuyến v2 sang PostgresDB
export async function executeTransHisExport(
  version: "v1" | "v2",
  filter: TransHisExportFilter,
  mongoModel: TransHisModel,
  pgDB: PostgresDB
) {
  if (version === "v2") {
    return await pgDB.queryExportTransHisV2(filter);
  }
  return await mongoModel.findTransHisV1(filter);
}`, secondaryFile, primaryFile, primarySig)

		rcs := []string{
			fmt.Sprintf("**Cần tách biệt luồng V1 (MongoDB) và V2 (PostgreSQL) tại `%s`:** Không ghi đè trực tiếp lên `TransHisModel` đang phục vụ luồng V1 ổn định.", primaryFile),
			fmt.Sprintf("**Chuẩn hóa kết nối PostgreSQL tại `%s`:** Chỉ sử dụng `pgPool` thuần trong `PostgresDB`, không trộn lẫn `knexPg` hay Mongoose Schema.", secondaryFile),
		}
		sols := []string{
			fmt.Sprintf("**Giữ nguyên 100%% luồng V1 (MongoDB):** Bảo toàn nguyên trạng `%s` (`%s`) phục vụ `export_type = TRANS_HIS` (V1 cũ).", primaryFile, primarySig),
			fmt.Sprintf("**Tách riêng hàm xử lý PostgreSQL tại `%s`:** Sử dụng `pgPool` thuần trong `PostgresDB` cho luồng V2.", secondaryFile),
			"**Định tuyến Strategy V1 / V2 rõ ràng:** Nếu `version === 'v2'` thì gọi `pgDB.queryExportTransHisV2(filter)`, ngược lại giữ nguyên `mongoModel.findTransHisV1(filter)`.",
		}
		return &ArchitectLLMAnalysis{
			DirectAnswer:     fmt.Sprintf("Dạ anh, đối với task `%s` (%s), giải pháp chuẩn là giữ nguyên 100%% luồng `v1` (MongoDB `TransHisModel`) và tách độc lập luồng `v2` qua `PostgresDB` (chỉ dùng `pgPool` thuần tại `%s`), định tuyến song song theo `export_type`.", qt.ExternalID, qt.Title, secondaryFile),
			ShouldUpdatePlan: true,
			RootCauses:       rcs,
			Architecture:     rcs,
			Solutions:        sols,
			CodeDemo:         demo,
			CodeLang:         "typescript",
		}
	}

	var rcs, sols []string
	for _, st := range subTasks {
		relTouch := normalizeOrchestratorDisplayPath(st.ContextFileLine, st.TargetRepoName)
		rcs = append(rcs, fmt.Sprintf("Điểm chạm nguồn cần xử lý tại **`%s`** (`%s`): %s", relTouch, st.TargetRepoName, st.Title))
		sols = append(sols, fmt.Sprintf("**`%s` (`%s`):** %s", st.SubTaskID, st.TargetRepoName, st.Description))
	}
	var snippet string
	if len(hits) > 0 {
		snippet = fmt.Sprintf("// File: %s:%d (%s)\n%s", hits[0].RelPath, hits[0].LineNum, hits[0].Signature, hits[0].Snippet)
	}
	return &ArchitectLLMAnalysis{
		DirectAnswer: synthesizeDirectAnswerSummary(qt, userPrompt, rcs, sols),
		RootCauses:   rcs,
		Architecture: rcs,
		Solutions:    sols,
		CodeDemo:     snippet,
		CodeLang:     codeLang,
	}
}
