# 09 Tasks Solution: Chuẩn Hóa Toàn Diện 14 Điểm Đứt Gãy Giữa `ai-orchestrator` (`Start -> Isolated Branch -> S0..S6`) Và Hệ Thống Lõi `/agent`

- **Workspace:** `audit-master-mongo-orchestrator-2026-10-22`
- **Suffix:** `flow_gap_vs_agent_core`
- **Tier:** `T2`
- **Trạng thái:** `S3 SOLUTION — PROPOSE-ONLY (Chờ User APPROVE)`

---

## 1. Tóm Tắt 14 Bước / Cơ Chế Còn Thiếu (`GAP-FLOW-01 .. GAP-FLOW-14`) Sau Khi Đọc 100% `/agent`

Sau khi đọc trọn vẹn `hooks/agy/agy_guard.py` (`1–1243`), `tooling/orchestrator/harness2_engine.py` (`1–1406`), `tooling/verify_governance.py` (`1–407`), `tooling/evidence_run.py` (`1–119`), `tooling/mutation_gate.py` (`1–400`), `tooling/laya_gate.py` (`1–687`), `memory/global/lessons/tripwires.json` (`1–15`), `rules/GEMINI_FULL_ARCHIVE_2026-10.md` (`1–255`), `OPERATOR_MAP.md` (`1–34`) và đối chiếu với cả 2 luồng của `ai-orchestrator` (`internal/orchestrator/task_runner.go:1-1941` và `internal/webui/server.go:1-2482` + `task_inspector.go:1-2501` + `internal/agentgate/gate.go:1-374`), dưới đây là **14 điểm lệch/thiếu chính xác tuyệt đối**:

1. **`GAP-FLOW-01` (Phân mảnh 2 luồng `TaskRunner` vs `WebUI Server`):**
   - `task_runner.go:281-503, 1011-1343` có bộ máy chạy `Beads Wave Parallel Execution -> TDD QA -> Coder Worktree -> QA/UX/Sec Review -> PM Arbitrate` nhưng không gắn với hồ sơ `/agent` và UI `:8090`; ngược lại `server.go:1043-1493` phục vụ UI `:8090` nhưng đang chạy hàm Go tĩnh mà chưa kích hoạt `TaskRunner`/CLI Runner theo từng Wave `0..N`.
2. **`GAP-FLOW-02` (`S0 Boot / Hook H1` thiếu 5 bước khởi động lõi):**
   - Chưa ping `http://127.0.0.1:8200/health` (`laya_up` — `agy_guard.py:678-695, 728-729`).
   - Chưa chạy `Harness² AgentSurfaceValidator` (`validate_agent_surface` kiểm tra `c1..c8`, `GEMINI.md + OPERATOR_MAP.md <= 24KB` — `agy_guard.py:731-739`, `harness2_engine.py:133-175`).
   - Chưa nạp `Domain Harness Library` (`memory/global/harness_library/*/INDEX.md` — `agy_guard.py:740-743`).
   - Chưa quét **`Lesson Tripwires (G-08)`** từ `memory/global/lessons/tripwires.json` (`match_tripwires` — `agy_guard.py:443-453, 746-750`).
   - Chưa đọc Bộ não dự án tại `memory/global/project/<project_name>/{project_context.md,active_plans.md,tech_stack.md}` (`GEMINI_FULL_ARCHIVE_2026-10.md:103`).
3. **`GAP-FLOW-03` (`Start -> Isolated Branch / Per-Repo Git Worktree` — Rule #19):**
   - `ensureTaskIsolation` (`server.go:494-518`) chỉ chạy `git worktree add -B task/<key>` khi `repoDir == s.Root`. Với task đa repo (`AffectedRepos`, `SubTaskDTO.WorktreePath = .orchestrator/worktrees/<task>/<repoName>`), hệ thống chỉ tạo thư mục rỗng `.task-worktree` mà chưa chạy `git rev-parse --show-toplevel` và khởi tạo worktree/sandbox cô lập cho từng repo con.
4. **`GAP-FLOW-04` (`S1 Intake / Hooks H2 & H4` thiếu `05_progress_<suffix>.md` bất biến, `04_decisions_<suffix>.md`, Tier động `T0..T3` & `mode: clean-room`):**
   - `task_inspector.go:1728` hardcode `"tier: T2"`; không hỗ trợ khai báo `mode: clean-room` + `clean_room_forbidden_sources`; thiếu hoàn toàn sổ cái **Append-Only `05_progress_<suffix>.md`** theo chuẩn `- [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] START...` (`verify_governance.py:29, 310-338`) và `04_decisions_<suffix>.md`.
5. **`GAP-FLOW-05` (`S2 Investigate / Hook H4t & Gate G2`):**
   - Chưa kiểm tra `H4t unmet_tripwires()` (`agy_guard.py:456-509, 793-805`) trước khi cho phép ghi `13_analysis` / `09_tasks_solution`.
   - `13_analysis.md` chưa phân tách rõ bằng chứng `VERIFIED` (có `REF: path:start-end`) vs `HYPOTHESIS`, và chưa có bước **Gate G2 Reproduce (`Red -> Green`)** tái hiện lỗi trước khi sửa.
6. **`GAP-FLOW-06` (`S3 Solution / Hook H4a & G-03 Clean-Room Approved Spec`):**
   - Khi task có `mode: clean-room`, `09_tasks_solution.md` chưa tách mục `## Clean-Room Approved Spec` và chưa chạy bộ lọc `spec_rule_issues` (`G-03` tại `agy_guard.py:323-355`, `verify_governance.py:82-105` bắt buộc mọi rule `R<n>`/`CR-<n>` và mọi dòng bullet đều có `REF: path:line` và không có `REJECTED/NO_EVIDENCE`).
7. **`GAP-FLOW-07` (`DG1 + DG2 Gate` đang ghi tĩnh trong WebUI `plan`):**
   - `task_inspector.go:1807` đang ghi chuỗi tĩnh `"Decision Gate DG1 / DG2: PASS 🟢"` mà chưa tự sinh file `06_laya_claims_dg1_<suffix>.json` (`S-RC`, `S-FIX`, `S-IMPACT`, `S-PATTERN`, `D-BEHAV`, `D-EDGE`, `D-API` + `>=2 control_false` `K1,K2` + `>=1 control_true` `K3` kèm `REF:` khớp từng dòng file thật) và chưa gọi thật `laya_gate.py run --stage dg1` (kèm `laya_gate.py resolve` lưu vào `04_decisions_<suffix>.md` nếu có `FLAG`).
8. **`GAP-FLOW-08` (`S4 Pre-Implement`: Thiếu `Git Restore-Point` local & `Clean-Room Subagent Leak Guard G-01..G-04`):**
   - Trước khi sửa code ở `S4`, chưa chụp **Git Restore-Point (`restore_points.json` / local commit SHA)** theo Rule #19 (`GEMINI.md:73, 120-121`).
   - Chưa có cổng Cách ly Phòng Sạch (`G-01/G-02/G-04`: `contaminated_clean_room_reads`, `assert_pre_flight_clean_room`, và `handle_pre_subagent` tại `agy_guard.py:397-428, 991-1014` chặn truyền đường dẫn cấm hoặc dòng code `>= 40` ký tự từ file cấm sang Sub-Agent).
9. **`GAP-FLOW-09` (`S4 Post-Implement`: Thiếu đọc `<repo>/.agent-project.json`, thiếu chạy `tooling/evidence_run.py` lấy biên lai `EV-...json` có `src_fingerprint`, và thiếu `H4c Claim <-> Evidence Linter`):**
   - `server.go:1338-1346` chỉ chạy `go vet ./...` bằng `sh -c`, chưa đọc `checks` / `test` / `runtime_check` từ `<repo>/.agent-project.json` (`find_manifest` — `agy_guard.py:650-675`), chưa bọc qua `evidence_run.py --kind unit|integration|runtime` để sinh `06_evidence/EV-<ts>-<hex4>.json` có `src_fingerprint` (SHA-256 toàn bộ source tree), và chưa chạy `claim_evidence_issues` (`H4c` — `agy_guard.py:540-594`) để chặn biên lai giả/cũ (`STALE`).
10. **`GAP-FLOW-10` (`S4 / Gate G3`: WebUI chưa nối `tooling/mutation_gate.py --base <ref>`):**
    - `server.go` chưa gọi `RunMutationGate`, còn `gate.go:313-326` chưa truyền tham số `--base <restore_ref>` để kích hoạt kiểm tra `diff_anchor_issues` (`mutation_gate.py:133-168`: mọi mutant phải chạm dòng `+` của `git diff <base>` và mọi file source đã sửa phải có `>= 1` mutant bị `KILLED` bởi `KILL_SIGNAL_RE`, đồng thời `all_restored_sha256 == true`).
11. **`GAP-FLOW-11` (`S5 Autonomous Self-Healing Loop` lệch Rule #9.7d & thiếu `H6 Escalation Mentor >= 3 loops -> Harness² Recursion`):**
    - Theo Rule #9.7d (`GEMINI.md:78`, `agy_guard.py:899-917`), khi ở `S5` phát hiện lỗi, Agent **CẤM dừng lại bắt User `APPROVE` lần 2** mà phải tự động chạy vòng lặp khép kín `S5 -> S3 (cập nhật 09_tasks_solution + 08_tasks + chạy lại DG1 PASS) -> S4 (tự sửa dứt điểm + chạy lại EV-...) -> S5 (Audit + DG3)`. Chỉ khi lặp thất bại **`>= 3` lần liên tiếp** (`Rule #15` & `H6` tại `agy_guard.py:136-143, 1061-1066`), hệ thống mới kích hoạt **`harness2_engine.py run-recursion --use-live-model`** (trục `INV` vs `DOM`) và dừng ở `NEEDS_YOU` báo User. Hiện tại `server.go:1395-1399, 1478-1480` đang dừng ngay ở lần đầu tiên!
12. **`GAP-FLOW-12` (`S5 Hội đồng /laya_router DG3`: Thiếu `Mode A + Mode B Review` và kiểm tra `mtime(dg3) >= last_src_edit`):**
    - `server.go:1380-1390` và `gate.go:262-263` mới chỉ đọc `dg1/dg2`, chưa gọi `laya_gate.py run --stage dg3` (Mode A) + `laya_gate.py review` (Mode B Agentic Debate kèm `corrupted-diff` control), và chưa cưỡng chế `mtime(06_laya_summary_dg3_*.json) >= last_src_edit` (`agy_guard.py:1132-1135`).
13. **`GAP-FLOW-13` (`S5 Audit Report & H4c Receipt Binding`):**
    - `audit_report_<taskKey>.md` do `server.go:1401-1456` xuất ra chưa nhúng mã biên lai `EV-...` thật từ `06_evidence/EV-*.json` cho từng câu khẳng định `PASS / OK / runtime / a/b` và chưa đồng bộ sang `/Users/trainguyen/Documents/work/agent/memory/workspaces/<ws>/audit_report_<suffix>.md`.
14. **`GAP-FLOW-14` (`S6 Close / Rule #18 & #22 / Hook H7`):**
    - Trước khi chuyển `Stage = DONE` (`server.go:1461-1488`), hệ thống chưa đồng bộ bộ tài liệu (`01, 05, 08, 09, audit_report`) sang `/agent/memory/workspaces/<ws>/`, chưa cập nhật `memory/global/project/<project_name>/`, chưa append dòng `DONE` vào `05_progress_<suffix>.md`, và chưa gọi `verify_governance.py --workspace <ws> --suffix <suffix> --stage S6` để xác nhận `GOVERNANCE AUDIT PASSED 🟢`!

---

## 2. Phương Án Kỹ Thuật Duy Nhất & Tối Ưu Nhất (Single Best Solution)

Nâng cấp đồng bộ 3 module trong `ai-orchestrator` (`internal/agentgate/gate.go`, `internal/webui/server.go`, `internal/webui/task_inspector.go` + nối `internal/orchestrator/task_runner.go`) để thực thi trọn vẹn 14 mắt xích của `/agent`:

1. **Tầng `agentgate/gate.go` — Bổ sung 8 Hàm Cầu Nối Trực Tiếp Tới Bộ Tooling Vật Lý Của `/agent` (`GAP-FLOW-02, 05, 06, 08, 09, 10, 11, 12, 14`):**
   - `CheckLayaAndHarnessSurface(ctx)`: Ping `http://127.0.0.1:8200/health` (nếu down thì gọi `scripts/ensure_laya_up.sh`), chạy `harness2_engine.py validate-surface`, đọc `memory/global/harness_library/*/INDEX.md` và `memory/global/project/<project>/`.
   - `MatchAndVerifyTripwires(prompt, evidencedPaths)`: Đọc `memory/global/lessons/tripwires.json`, trả về các tripwire khớp ở `S0` (`G-08`) và kiểm tra điều kiện `require` trước khi ghi `13_analysis` / `09_tasks_solution` (`H4t`).
   - `ValidateCleanRoomSpecAndGuard(solText, forbiddenGlobs, subagentPrompt)`: Thực thi `spec_rule_issues` (`G-03`), `assert_pre_flight_clean_room` và `leaked_subagent_prompts` (`G-04`).
   - `LoadProjectManifest(repoDir)`: Đọc `<repo>/.agent-project.json` lấy `checks`, `test`, `runtime_check` (Rule #3 ADR-07).
   - `RunEvidenceCommand(ctx, workspace, kind, cwd, cmdLine)`: Gọi trực tiếp `python3 /Users/trainguyen/Documents/work/agent/tooling/evidence_run.py --workspace <ws> --kind <unit|integration|runtime> --cwd <cwd> -- <cmd>`, trả về mã biên lai `EV-...` kèm `src_fingerprint`.
   - `VerifyClaimEvidenceReceipts(workspace, docText)`: Thực thi logic `claim_evidence_issues` (`H4c`) kiểm tra mọi dòng `PASS/OK/KILLED/runtime/a/b` đều có `EV-...` thật và `src_fingerprint == source_fingerprint(cwd)`.
   - `RunMutationGateWithBase(ctx, workspace, suffix, mutationsPath, baseRef)`: Gọi `mutation_gate.py --base <baseRef>`.
   - `RunHarness2RecursionOnEscalation(ctx, taskID)`: Khi số vòng lặp thất bại liên tiếp `>= 3` (`H6`), gọi `harness2_engine.py run-recursion --task-id <taskID> --use-live-model`.

2. **Tầng `webui/server.go` & `task_inspector.go` — Khép Kín Luồng `Start -> Isolated Branch -> S0..S6 -> Done` (`GAP-FLOW-01, 03, 04, 07, 11, 13, 14`):**
   - **`Start -> Isolated Branch (Layer 1)`:** Trong `ensureTaskIsolation`, với **mỗi repo trong `AffectedRepos`**, chạy `git rev-parse --show-toplevel` và `git rev-parse HEAD` lưu vào `restore_points.json` (Rule #19), đồng thời khởi tạo thư mục worktree con `.orchestrator/worktrees/<taskKey>/<repoName>`.
   - **`S1 Intake`:** Tự động phân loại Tier (`T0 | Audit | T1 | T2 | T3`) và `mode: standard | clean-room`; khởi tạo `05_progress_<suffix>.md` (Append-Only chuẩn `- [YYYY-MM-DDTHH:MM:SS+07:00] [Agent:<Model>] START...`) và `04_decisions_<suffix>.md`.
   - **`S2 & S3 + Real DG1/DG2`:** Bổ sung bảng `VERIFIED` vs `HYPOTHESIS` + kịch bản `Gate G2 (Red -> Green)` vào `13_analysis_<suffix>.md`; bổ sung `## Clean-Room Approved Spec` vào `09_tasks_solution_<suffix>.md`; tự động sinh `06_laya_claims_dg1_<suffix>.json` (đầy đủ `S-*`, `D-*`, `K1, K2, K3` kèm `REF:` thật) và gọi `Gatekeeper.RunLayaGate(ctx, "dg1", ...)` trước khi dừng ở `NEEDS_YOU`.
   - **`S4 -> S5 Autonomous Self-Healing Loop (`Rule #9.7d`) & `H6 Escalation (>= 3 loops)`:**
     - Khi chạy `qa` (`S5`), nếu phát hiện lỗi/Sub-Task chưa đạt và `selfHealLoop < 3`: **KHÔNG dừng bắt User `APPROVE` lần 2** mà tự động quay về `S3` cập nhật `09_tasks_solution` + `08_tasks` + chạy lại `DG1` (`PASS`) $\rightarrow$ tự chạy lại `S4 Implement` + `evidence_run.py` (`EV-...`) $\rightarrow$ chạy lại `S5 Audit + DG3` khép kín ngay trong cùng lượt!
     - Nếu `selfHealLoop >= 3` vẫn còn `blocking > 0`: Kích hoạt `H6 Escalation` (`RunHarness2RecursionOnEscalation`) và dừng ở `NEEDS_YOU` báo cáo chi tiết cho anh.
   - **`S6 Close`:** Chạy `RunLayaGate(ctx, "dg3", ...)` (đảm bảo `mtime(dg3) >= last_src_edit`), nhúng biên lai `EV-...` thật vào `audit_report_<suffix>.md`, đồng bộ bộ hồ sơ sang `/Users/trainguyen/Documents/work/agent/memory/workspaces/<ws>/`, append `DONE` vào `05_progress_<suffix>.md`, và bắt buộc `Gatekeeper.RunGovernanceLinter(ctx, ws, "S6", suffix)` đạt `PASSED 🟢` trước khi chuyển `Stage = DONE`.

---

## 3. Code Demo Cụ Thể (Concrete Implementation Preview)

```go
// === 1. internal/agentgate/gate.go: Thực thi evidence_run.py (H4c EV-... + src_fingerprint) & verify_governance.py ===
type EvidenceReceipt struct {
	ID             string `json:"id"`
	Cmd            string `json:"cmd"`
	Cwd            string `json:"cwd"`
	Kind           string `json:"kind"`
	Exit           int    `json:"exit"`
	StartedAt      string `json:"started_at"`
	StdoutTail     string `json:"stdout_tail"`
	StdoutSHA256   string `json:"stdout_sha256"`
	SrcFingerprint string `json:"src_fingerprint"`
}

func (g *Gatekeeper) RunEvidenceCommand(ctx context.Context, workspace, kind, cwd, cmdLine string) (EvidenceReceipt, error) {
	var rec EvidenceReceipt
	if g == nil || g.AgentCoreDir == "" {
		return rec, fmt.Errorf("gatekeeper not configured")
	}
	script := filepath.Join(g.AgentCoreDir, "tooling", "evidence_run.py")
	out, err := runPythonTool(ctx, g.AgentCoreDir, script,
		"--workspace", workspace,
		"--kind", kind,
		"--cwd", cwd,
		"--", cmdLine,
	)
	// Parse "EVIDENCE EV-YYYYmmddTHHMMSS-xxxx exit=0" from stdout tail
	re := regexp.MustCompile(`EVIDENCE\s+(EV-\d{8}T\d{6}-[0-9a-f]{4})\s+exit=(-?\d+)`)
	if m := re.FindStringSubmatch(out); len(m) == 3 {
		evPath := filepath.Join(g.AgentCoreDir, "memory", "workspaces", workspace, "06_evidence", m[1]+".json")
		if raw, rErr := os.ReadFile(evPath); rErr == nil {
			_ = json.Unmarshal(raw, &rec)
		}
	}
	return rec, err
}

// === 2. internal/webui/server.go: Vòng lặp Tự Chữa Lành Khép Kín S5 -> S3 -> S4 -> S5 (Rule #9.7d) & Escalation >= 3 (Rule #15 / H6) ===
func (s *Server) runAutonomousS4S5WithSelfHealing(ctx context.Context, taskKey, activeRunner string, stepDelay time.Duration) error {
	const maxSelfHealLoops = 3
	for loop := 1; loop <= maxSelfHealLoops; loop++ {
		if err := s.runSingleStageWithDelay(ctx, taskKey, "implement", activeRunner, stepDelay); err != nil {
			return err
		}
		if err := s.runSingleStageWithDelay(ctx, taskKey, "qa", activeRunner, stepDelay); err != nil {
			return err
		}
		qt, err := s.findQueuedTask(ctx, taskKey)
		if err != nil || qt.Stage == queue.StageDone {
			return err
		}
		if loop < maxSelfHealLoops {
			// Rule #9.7d: Không dừng bắt User APPROVE lần 2 -> Tự quay về S3 cập nhật 09_tasks_solution + 08_tasks + chạy lại DG1 rồi tự chạy lại S4 -> S5!
			s.appendImmutableProgressLog(taskKey, qt.AssignedModel,
				fmt.Sprintf("SELF_HEAL_LOOP_%d S5->S3->S4->S5: Cập nhật 09_tasks_solution & 08_tasks, chạy lại DG1 và tự thực thi lại S4/S5 (Rule #9.7d)", loop))
			s.autoRefreshS3PlanAndDG1ForSelfHeal(ctx, *qt, activeRunner, loop)
			continue
		}
		// Loop >= 3 (Rule #15 & Hook H6): Escalate sang Harness² Recursion & dừng ở NEEDS_YOU báo User
		if s.Gatekeeper != nil {
			_, _, _ = s.Gatekeeper.RunHarness2Recursion(ctx, qt.ExternalID)
		}
		s.appendImmutableProgressLog(taskKey, qt.AssignedModel,
			"ESCALATE_H6: Đã thử tự chữa lành 3 vòng liên tiếp không đạt -> Kích hoạt Harness² Recursion (INV vs DOM) và dừng chờ User tái định hướng (Rule #15)")
	}
	return nil
}
```

