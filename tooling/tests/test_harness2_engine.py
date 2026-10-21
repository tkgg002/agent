#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Unit & integration tests for tooling/orchestrator/harness2_engine.py (F1–F6)."""
from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import sys
import tempfile
import unittest
from typing import Dict

AGENT_ROOT = Path(__file__).resolve().parents[2]
if str(AGENT_ROOT) not in sys.path:
    sys.path.insert(0, str(AGENT_ROOT))

from tooling.orchestrator import harness2_engine as h2  # noqa: E402


class Harness2EngineTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp_obj = tempfile.TemporaryDirectory()
        self.tmp = Path(self.tmp_obj.name)
        self.base_tree: Dict[str, str] = {
            "GEMINI.md": (
                "# GEMINI CORE\n"
                "Use skill `brain-delegate` and run `python3 tooling/verify_governance.py`.\n"
            ),
            "OPERATOR_MAP.md": (
                "# Operator Map\n"
                "| `brain-delegate` | Brain | S0-S3 | Core planner |\n"
            ),
            "skills/brain-delegate/SKILL.md": (
                "---\n"
                "name: brain-delegate\n"
                "description: Brain delegate skill\n"
                "---\n"
                "# Brain Delegate\n"
            ),
            "tooling/verify_governance.py": "def ok() -> bool:\n    return True\n",
        }

    def tearDown(self) -> None:
        self.tmp_obj.cleanup()

    # === Test F3 (Task 3): AgentSurfaceValidator & Auto-Trigger Compiler ===
    def test_01_f3_validate_real_agent_surface_and_detect_surface_violations(self) -> None:
        real_tree = h2.load_agent_surface_tree(AGENT_ROOT)
        spec = h2.AgentSurfaceSpec()
        self.assertEqual(h2.validate_agent_surface(spec, real_tree), [])

        # 1. Always-on budget exceeded
        tight_spec = h2.AgentSurfaceSpec(always_on_max=20)
        errs_budget = h2.validate_agent_surface(tight_spec, self.base_tree)
        self.assertTrue(any("always-on vượt trần" in e for e in errs_budget))

        # 2. Missing trigger in always-on + autotrigger fixes it
        untriggered = dict(self.base_tree)
        untriggered["skills/domain-cdc/SKILL.md"] = (
            "---\nname: domain-cdc\ndescription: CDC specialist\n---\n# CDC\n"
        )
        errs_trig = h2.validate_agent_surface(spec, untriggered)
        self.assertTrue(any("chưa được nhắc tên" in e for e in errs_trig))

        auto_fixed = h2.autotrigger_agent_skills(spec, untriggered)
        self.assertEqual(h2.validate_agent_surface(spec, auto_fixed), [])
        self.assertIn("domain-cdc", auto_fixed["OPERATOR_MAP.md"])

        # 3. Missing script reference & Python SyntaxError
        bad_script_tree = dict(self.base_tree)
        bad_script_tree["GEMINI.md"] += "\nRun `python3 tooling/non_existent_script.py`\n"
        bad_script_tree["tooling/broken.py"] = "def broken(:\n    pass\n"
        errs_script = h2.validate_agent_surface(spec, bad_script_tree)
        self.assertTrue(any("non_existent_script.py" in e for e in errs_script))
        self.assertTrue(any("lỗi cú pháp Python" in e for e in errs_script))

    # === Test F2 (Task 2): Orthogonal Proposer Axes (INV vs DOM) & KEEP_BASE ===
    def test_02_f2_orthogonal_axis_isolation_and_keep_base(self) -> None:
        ok_keep, msg_keep = h2.validate_orthogonal_axis_edit("INV", self.base_tree, self.base_tree)
        self.assertTrue(ok_keep)
        self.assertEqual(msg_keep, "KEEP_BASE")

        # INV cannot edit project memory or domain-* skills
        bad_inv = dict(self.base_tree)
        bad_inv["memory/global/project/data_hub/arch.md"] = "domain note"
        ok_inv, msg_inv = h2.validate_orthogonal_axis_edit("INV", self.base_tree, bad_inv)
        self.assertFalse(ok_inv)
        self.assertIn("INV Proposer không được sửa", msg_inv)

        # DOM cannot edit tooling/ or hooks/agy/ or GEMINI.md
        bad_dom = dict(self.base_tree)
        bad_dom["tooling/verify_governance.py"] = "def ok() -> bool:\n    return False\n"
        ok_dom, msg_dom = h2.validate_orthogonal_axis_edit("DOM", self.base_tree, bad_dom)
        self.assertFalse(ok_dom)
        self.assertIn("DOM Proposer không được sửa", msg_dom)

        # Valid INV edit (tooling script) and valid DOM edit (skill)
        good_inv = dict(self.base_tree)
        good_inv["tooling/extra_check.py"] = "X = 1\n"
        self.assertTrue(h2.validate_orthogonal_axis_edit("INV", self.base_tree, good_inv)[0])

        good_dom = dict(self.base_tree)
        good_dom["skills/domain-sql/SKILL.md"] = (
            "---\nname: domain-sql\ndescription: SQL skill\n---\n"
        )
        self.assertTrue(h2.validate_orthogonal_axis_edit("DOM", self.base_tree, good_dom)[0])

    # === Test F4 (Task 4): Pre-Flight Clean-Room Leak & Verdict Tripwire ===
    def test_03_f4_context_tree_and_pre_flight_clean_room_tripwire(self) -> None:
        ctx_dir = self.tmp / "context_clean"
        info = h2.build_evidence_context_tree(
            ctx_dir,
            task_id="t100",
            task_prompt="Rebuild pipeline router",
            harness_tree=self.base_tree,
            rollout_logs={"base1": "Step 1 executed cleanly"},
            forbidden_globs=["legacy_router.go", "internal/secret_impl/*"],
        )
        self.assertEqual(info["rollout_count"], 1)
        self.assertTrue((ctx_dir / "EVIDENCE_BOUNDARY.md").is_file())

        # 1. Forbidden source path leaked into context
        leaked_file = ctx_dir / "deliverables" / "legacy_router.go"
        leaked_file.parent.mkdir(parents=True, exist_ok=True)
        leaked_file.write_text("package main\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Pre-flight leak path detected"):
            h2.assert_pre_flight_clean_room(ctx_dir, ["legacy_router.go"])
        leaked_file.unlink()

        # 2. Grader verdict marker leaked into context
        verdict_file = ctx_dir / "rollouts" / "eval_leak.json"
        verdict_file.write_text('{"pass_rate": 1.0}', encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Grader verdict marker"):
            h2.assert_pre_flight_clean_room(ctx_dir, [])
        verdict_file.unlink()

        # 3. Transcript post-check contamination
        contaminated_steps = [
            {
                "tool_calls": [
                    {
                        "name": "view_file",
                        "args": {"AbsolutePath": "/repo/internal/legacy_router.go"},
                    }
                ]
            }
        ]
        with self.assertRaisesRegex(ValueError, "Clean-Room transcript contaminated"):
            h2.assert_pre_flight_clean_room(ctx_dir, ["legacy_router.go"], contaminated_steps)

    # === Test F5 (Task 5): Blind Anonymized Selection & Line-Jaccard Diversity ===
    def test_04_f5_anonymous_ids_and_pairwise_line_jaccard_diversity(self) -> None:
        labels = ["base", "agg1", "agg2"]
        m1 = h2.assign_anonymous_candidate_ids(labels, task_id="task-42", k=2, mode="agg")
        m2 = h2.assign_anonymous_candidate_ids(labels, task_id="task-42", k=2, mode="agg")
        self.assertEqual(m1, m2)
        self.assertEqual(sorted(m1.keys()), ["A", "B", "C"])
        self.assertEqual(sorted(m1.values()), sorted(labels))

        identical = {"c1": self.base_tree, "c2": dict(self.base_tree)}
        self.assertEqual(h2.compute_tree_diversity(identical)["mean_line_jaccard"], 1.0)

        diverged = {
            "c1": self.base_tree,
            "c2": {**self.base_tree, "tooling/new_helper.py": "def calc():\n    return 42\n"},
        }
        div_score = h2.compute_tree_diversity(diverged)["mean_line_jaccard"]
        self.assertLess(div_score, 1.0)
        self.assertGreater(div_score, 0.0)

    # === Test F6 (Task 6): Ledger Identity (R=4k, P=3k), SHA-256 Cache & flock ===
    def test_05_f6_ledger_identity_and_concurrent_flock_summary(self) -> None:
        ledger = h2.DeterministicRecursionLedger(mode="agg", task_id="t1", k=1)
        for r in ("base1", "inv1", "dom1", "agg1"):
            ledger.record_rollout(r)
        for s in ("inv1", "dom1", "agg1"):
            ledger.record_session(s)
        ledger.assert_identity()

        # Duplicate label or wrong count must fail immediately
        ledger.record_rollout("agg1")
        with self.assertRaisesRegex(AssertionError, "BudgetError"):
            ledger.assert_identity()

        # Concurrent multi-worker flock merge test (8 parallel workers)
        summary_file = self.tmp / "concurrent_summary.json"

        def _worker(idx: int) -> None:
            h2.atomic_flock_merge_summary(
                summary_file,
                task_id=f"task_{idx:02d}",
                entry={"Chosen": f"agg{idx}", "ok": True},
            )

        with ThreadPoolExecutor(max_workers=8) as pool:
            list(pool.map(_worker, range(8)))

        merged = json.loads(summary_file.read_text(encoding="utf-8"))
        self.assertEqual(len(merged["tasks"]), 8)

    # === Test F1 (Task 1): Two-Level Recursion (agg & seq) + Domain Library ===
    def test_06_f1_two_level_recursion_agg_and_seq_with_domain_library_and_cache(self) -> None:
        lib_dir = self.tmp / "harness_library"
        cache_dir = self.tmp / "session_cache"
        lib = h2.DomainHarnessLibrary(lib_dir)
        engine = h2.RecursiveHarnessEngine(library=lib, cache_dir=cache_dir)

        def proposer_fn(label: str, dim: str, tree: Dict[str, str]) -> Dict[str, str]:
            out = dict(tree)
            if dim == "INV":
                out[f"tooling/check_{label}.py"] = f" STEP = '{label}'\n".lstrip()
            elif dim == "DOM":
                sname = f"domain-{label}"
                out[f"skills/{sname}/SKILL.md"] = (
                    f"---\nname: {sname}\ndescription: Skill for {label}\n---\n# {sname}\n"
                )
            elif dim == "COMPOSE":
                out[f"tooling/composed_{label}.py"] = f"COMPOSED = '{label}'\n"
            return out

        def rollout_fn(label: str, tree: Dict[str, str]) -> Dict[str, float]:
            score = 0.95 if label.startswith(("agg", "c")) else 0.60
            return {"score": score, "files": float(len(tree))}

        # 1. Run parallel mode="agg" with k=2 -> R = 8, P = 6
        res_agg = engine.execute_task_recursion(
            task_id="cdc_batch_tuning",
            domain="data_pipeline",
            mode="agg",
            k=2,
            base_tree=self.base_tree,
            proposer_fn=proposer_fn,
            rollout_fn=rollout_fn,
            dg3_conf=0.92,
            mutants_killed=3,
        )
        self.assertEqual(res_agg["rollouts_count"], 8)
        self.assertEqual(res_agg["sessions_count"], 6)
        self.assertEqual(res_agg["cache_hits"], 0)
        self.assertTrue(res_agg["chosen_label"].startswith("agg"))
        self.assertIsNotNone(res_agg["library_slot"])

        index_md = (lib_dir / "data_pipeline" / "INDEX.md").read_text(encoding="utf-8")
        self.assertIn("cdc_batch_tuning", index_md)
        exp_md = (
            lib_dir / "data_pipeline" / "harnesses" / "cdc_batch_tuning" / "EXPERIENCE.md"
        ).read_text(encoding="utf-8")
        self.assertIn("Winning Candidate", exp_md)

        # Re-running the same agg task hits SHA-256 session cache for all 6 proposer sessions
        res_agg_cached = engine.execute_task_recursion(
            task_id="cdc_batch_tuning",
            domain="data_pipeline",
            mode="agg",
            k=2,
            base_tree=self.base_tree,
            proposer_fn=proposer_fn,
            rollout_fn=rollout_fn,
        )
        self.assertEqual(res_agg_cached["cache_hits"], 6)

        # 2. Run sequential mode="seq" with k=2 -> R = 8, P = 6
        engine_seq = h2.RecursiveHarnessEngine(library=lib)
        res_seq = engine_seq.execute_task_recursion(
            task_id="seq_schema_drift",
            domain="data_pipeline",
            mode="seq",
            k=2,
            base_tree=self.base_tree,
            proposer_fn=proposer_fn,
            rollout_fn=rollout_fn,
        )
        self.assertEqual(res_seq["rollouts_count"], 8)
        self.assertEqual(res_seq["sessions_count"], 6)
        self.assertTrue(res_seq["chosen_label"].startswith("c"))

        # Gate check helper rejects low confidence or append-only target files
        self.assertFalse(
            h2.can_commit_adopted_harness_to_domain_library(
                dg3_conf=0.2,
                mutants_killed=3,
                s3_docs_present=["01_req.md", "05_prog.md", "08_tsk.md", "09_sol.md"],
                target_file="memory/global/harness_library/INDEX.md",
            )
        )
        self.assertFalse(
            h2.can_commit_adopted_harness_to_domain_library(
                dg3_conf=0.9,
                mutants_killed=3,
                s3_docs_present=["01_req.md", "05_prog.md", "08_tsk.md", "09_sol.md"],
                target_file="memory/global/lessons.md",
            )
        )

    # === Test Task 7 (AF-01..AF-07): Adversarial Audit Remediation & Subprocess Runtime ===
    def test_07_af01_to_af07_adversarial_remediation_and_real_subprocess_trace(self) -> None:
        from tooling.orchestrator import proposer_axes

        # 1. AF-01: three_way_merge_trees preserves INV edits on existing files and deletions
        base_with_obsolete = {
            **self.base_tree,
            "tooling/obsolete.py": "OLD = True\n",
        }
        inv_branch = dict(base_with_obsolete)
        inv_branch["tooling/verify_governance.py"] = (
            "def ok() -> bool:\n    return True  # INV_MODIFIED\n"
        )
        del inv_branch["tooling/obsolete.py"]

        dom_branch = dict(base_with_obsolete)
        dom_branch["skills/dom-cdc/SKILL.md"] = (
            "---\nname: dom-cdc\ndescription: CDC skill\n---\n# CDC\n"
        )

        merged_3way = proposer_axes.three_way_merge_trees(
            base_with_obsolete, inv_branch, dom_branch
        )
        self.assertIn("# INV_MODIFIED", merged_3way["tooling/verify_governance.py"])
        self.assertNotIn("tooling/obsolete.py", merged_3way)
        self.assertIn("skills/dom-cdc/SKILL.md", merged_3way)

        # 2. AF-04 & AF-06: UTF-8 multi-byte cap, strict YAML frontmatter, word-boundary skill token, and INV skill fence
        viet_tree = dict(self.base_tree)
        # 8000 'ă' chars = 8000 chars (< 10000), but 16000 UTF-8 bytes (> 10000)
        viet_tree["GEMINI.md"] = self.base_tree["GEMINI.md"] + ("ă" * 8000)
        utf8_spec = h2.AgentSurfaceSpec(always_on_max=10_000)
        errs_utf8 = h2.validate_agent_surface(utf8_spec, viet_tree)
        self.assertTrue(any("always-on vượt trần" in e for e in errs_utf8))

        # Body text mimicking frontmatter without --- delimiters must fail
        no_delim_tree = dict(self.base_tree)
        no_delim_tree["skills/fake-fm/SKILL.md"] = (
            "name: fake-fm\ndescription: Not inside YAML delimiters\n# Body\n"
        )
        no_delim_tree["OPERATOR_MAP.md"] += "\n| `fake-fm` |\n"
        errs_fm = h2.validate_agent_surface(h2.AgentSurfaceSpec(), no_delim_tree)
        self.assertTrue(any("thiếu description:" in e for e in errs_fm))

        # Substring collision: skill 'brain' must NOT match 'brain-delegate' in GEMINI.md
        sub_skill_tree = dict(self.base_tree)
        sub_skill_tree["skills/brain/SKILL.md"] = (
            "---\nname: brain\ndescription: Standalone brain skill\n---\n# Brain\n"
        )
        errs_sub = h2.validate_agent_surface(h2.AgentSurfaceSpec(), sub_skill_tree)
        self.assertTrue(any("skill 'brain' chưa được nhắc tên" in e for e in errs_sub))

        # AF-06: INV proposer prohibited from editing ANY skill in skills/* (even without 'domain-' prefix)
        inv_touch_skill = dict(self.base_tree)
        inv_touch_skill["skills/brain-delegate/SKILL.md"] += "\n# Extra\n"
        ok_axis, msg_axis = h2.validate_orthogonal_axis_edit(
            "INV", self.base_tree, inv_touch_skill
        )
        self.assertFalse(ok_axis)
        self.assertIn("INV Proposer không được sửa", msg_axis)

        # 3. AF-02, AF-03, AF-05: Cross-task cache isolation, base rollout winning, grader-free selector_fn & real s3_docs
        cache_dir = self.tmp / "shared_cache"
        lib_dir = self.tmp / "lib_af"
        empty_ws = self.tmp / "empty_ws"
        empty_ws.mkdir(parents=True, exist_ok=True)
        engine = h2.RecursiveHarnessEngine(
            library=h2.DomainHarnessLibrary(lib_dir),
            cache_dir=cache_dir,
        )

        call_counter = {"count": 0}

        def mutating_proposer(label: str, dim: str, tree: Dict[str, str]) -> Dict[str, str]:
            call_counter["count"] += 1
            out = dict(tree)
            if dim == "INV":
                out["tooling/verify_governance.py"] = (
                    f"def ok() -> bool:\n    return True  # {label}_{call_counter['count']}\n"
                )
            elif dim == "DOM":
                sname = f"dom-{label}"
                out[f"skills/{sname}/SKILL.md"] = (
                    f"---\nname: {sname}\ndescription: Skill {label}\n---\n"
                )
            return out

        # Base rollout scores 0.99 while candidate rollouts score 0.10 -> 'base' MUST win (AF-03)
        def base_wins_rollout(label: str, tree: Dict[str, str]) -> Dict[str, float]:
            return {"score": 0.99 if label.startswith(("base", "b")) else 0.10}

        res_task_a = engine.execute_task_recursion(
            task_id="task_alpha",
            domain="gov",
            mode="agg",
            k=1,
            base_tree=self.base_tree,
            proposer_fn=mutating_proposer,
            rollout_fn=base_wins_rollout,
            workspace_dir=empty_ws,
        )
        self.assertEqual(res_task_a["chosen_label"], "base")
        self.assertEqual(res_task_a["cache_hits"], 0)
        # Empty workspace lacks S3 docs -> library_slot must be None (AF-05)
        self.assertIsNone(res_task_a["library_slot"])

        # Different task_id ('task_beta') sharing cache_dir MUST NOT collide with 'task_alpha' (AF-02)
        res_task_b = engine.execute_task_recursion(
            task_id="task_beta",
            domain="gov",
            mode="agg",
            k=1,
            base_tree=self.base_tree,
            proposer_fn=mutating_proposer,
            rollout_fn=base_wins_rollout,
            workspace_dir=empty_ws,
        )
        self.assertEqual(res_task_b["cache_hits"], 0)

        # Grader-free selector_fn operating on anonymized IDs without 'score' field (AF-03)
        valid_ws = self.tmp / "valid_ws"
        valid_ws.mkdir(parents=True, exist_ok=True)
        for fname in (
            "01_requirements_t.md",
            "05_progress_t.md",
            "08_tasks_t.md",
            "09_tasks_solution_t.md",
        ):
            (valid_ws / fname).write_text("# Doc\n", encoding="utf-8")

        lessons_src = self.tmp / "lessons.md"
        lessons_src.write_text(
            "# Global Lessons\n\n## 1. Governance\n\n"
            "### [2026-10-20] Test Lesson\n"
            "- **Global Pattern:** [A] -> [B]. **Đúng:** flow\n"
            "- **Bối cảnh (Trigger):** ctx\n"
            "- **Root Cause:** rc\n"
            "- **Fix/Correct Flow:** fix\n"
            "- **Tags:** #governance\n",
            encoding="utf-8",
        )
        lessons_out = self.tmp / "lessons_out"

        def scoreless_rollout(label: str, tree: Dict[str, str]) -> Dict[str, object]:
            return {"trace": f"completed:{label}", "files": len(tree)}

        def custom_selector(
            id_map: Dict[str, str],
            trees: Dict[str, Dict[str, str]],
            rollouts: Dict[str, Dict[str, object]],
        ) -> str:
            # Pick the anonymous candidate whose tree contains both INV edit and DOM skill
            for anon_id, lbl in sorted(id_map.items()):
                if any(k.startswith("skills/dom-") for k in trees[lbl]):
                    return anon_id
            return sorted(id_map.keys())[0]

        res_sel = engine.execute_task_recursion(
            task_id="task_gamma",
            domain="gov",
            mode="agg",
            k=1,
            base_tree=self.base_tree,
            proposer_fn=mutating_proposer,
            rollout_fn=scoreless_rollout,
            selector_fn=custom_selector,
            workspace_dir=valid_ws,
            sync_lessons_index=True,
            lessons_source=lessons_src,
            lessons_out_dir=lessons_out,
        )
        self.assertEqual(res_sel["chosen_label"], "agg1")
        self.assertIn("# inv1_", res_sel["winning_tree"]["tooling/verify_governance.py"])
        self.assertIn("skills/dom-dom1/SKILL.md", res_sel["winning_tree"])
        self.assertIsNotNone(res_sel["library_slot"])
        self.assertEqual(res_sel["lessons_index_stats"]["index_entries"], 1)
        self.assertTrue((lessons_out / "INDEX.md").is_file())

        # 4. G-05: run-recursion KHÔNG còn proposer/scorer giả viết cứng -> thiếu lệnh thật phải exit 2
        rc_missing = h2.main(["run-recursion", "--k", "1", "--workspace-dir", str(valid_ws)])
        self.assertEqual(rc_missing, 2)

        # 5. AF-07/G-05: chạy subprocess thật với lệnh do caller truyền vào (grader-free, không dùng score giả)
        agent_root = self.tmp / "agent_root_cli"
        for rel, content in {
            "GEMINI.md": "# Core Rules\nRun `python3 tooling/existing_check.py`\n- `core-skill`\n",
            "OPERATOR_MAP.md": "# Operator Map\n| `core-skill` | Trigger | `S1` | Core |\n",
            "skills/core-skill/SKILL.md": "---\nname: core-skill\ndescription: Core workflow skill\n---\n# Body\n",
            "tooling/existing_check.py": "def verify_inv() -> bool:\n    return True\n",
        }.items():
            (agent_root / rel).parent.mkdir(parents=True, exist_ok=True)
            (agent_root / rel).write_text(content, encoding="utf-8")
        prop_py = (
            "import pathlib, sys; h = pathlib.Path(sys.argv[1]); lbl = sys.argv[2]; dim = sys.argv[3]; "
            "p = h / 'tooling/existing_check.py'; "
            "(p.write_text(p.read_text() + '# inv:' + lbl + '\\n', encoding='utf-8') if dim == 'INV' else None)"
        )
        roll_py = (
            "import json, pathlib, sys; w = pathlib.Path(sys.argv[1]); "
            "src = (w / 'tooling/existing_check.py').read_text(); "
            "compile(src, 'existing_check.py', 'exec'); "
            "print(json.dumps({'status': 'VALIDATED', 'has_inv': '# inv:' in src}))"
        )
        cli_lib = self.tmp / "cli_lib"
        rc = h2.main(
            [
                "run-recursion",
                "--task-id",
                "cli_subproc_test",
                "--domain",
                "agent_gov",
                "--mode",
                "agg",
                "--k",
                "1",
                "--workspace-dir",
                str(valid_ws),
                "--library-root",
                str(cli_lib),
                "--agent-root",
                str(agent_root),
                "--proposer-cmd",
                json.dumps([sys.executable, "-c", prop_py, "{harness_dir}", "{label}", "{dim}"]),
                "--rollout-cmd",
                json.dumps([sys.executable, "-c", roll_py, "{worktree}"]),
            ]
        )
        self.assertEqual(rc, 0)
        self.assertTrue((cli_lib / "agent_gov" / "INDEX.md").is_file())
        self.assertTrue(
            (cli_lib / "agent_gov" / "harnesses" / "cli_subproc_test" / "harness.json").is_file()
        )

    # === Test Task 8 (RF-01..RF-05): Real Model Proposer, Grader-Free Blind Selector, Persistent Library & S0-S6 Wiring ===
    def test_08_rf01_to_rf05_live_model_blind_selector_and_s0_s6_wiring(self) -> None:
        from http.server import BaseHTTPRequestHandler, HTTPServer
        import threading
        from hooks.agy import agy_guard

        # Khởi tạo local HTTP model endpoint giả lập định dạng /v1/predict của Laya để test deterministic
        class _PredictHandler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:  # noqa: N802
                length = int(self.headers.get("Content-Length", "0"))
                body = json.loads(self.rfile.read(length).decode("utf-8"))
                ttype = body.get("task_type", "decision_gate")
                resp = {
                    "prediction": "APPROVE",
                    "confidence": 0.94,
                    "reasoning": f"Live model verified {ttype} orthogonal synthesis.",
                }
                raw = json.dumps(resp).encode("utf-8")
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def log_message(self, format: str, *args: object) -> None:  # noqa: A002
                return

        srv = HTTPServer(("127.0.0.1", 0), _PredictHandler)
        port = srv.server_address[1]
        th = threading.Thread(target=srv.serve_forever, daemon=True)
        th.start()
        live_url = f"http://127.0.0.1:{port}/v1/predict"

        try:
            # 1. RF-03: SubprocessHarnessRunner.run_proposer KHÔNG truyền proposer_cmd_template phải gọi _run_live_model_proposer
            runner = h2.SubprocessHarnessRunner(laya_url=live_url)
            inv_tree = runner.run_proposer("inv1", "INV", self.base_tree)
            self.assertNotEqual(inv_tree, self.base_tree)
            self.assertIn("[Harness2-INV:inv1]", inv_tree["tooling/verify_governance.py"])

            dom_tree = runner.run_proposer("dom1", "DOM", self.base_tree)
            self.assertNotEqual(dom_tree, self.base_tree)
            self.assertTrue(any(k.startswith("skills/dom-dom1/") for k in dom_tree))

            # Nếu model endpoint offline -> bắt buộc raise RuntimeError, tuyệt đối không trả về base_tree no-op
            dead_runner = h2.SubprocessHarnessRunner(laya_url="http://127.0.0.1:1/v1/predict", timeout_sec=1.0)
            with self.assertRaises(RuntimeError):
                dead_runner.run_proposer("inv_dead", "INV", self.base_tree)

            # 2. RF-02: Grader-Free Blind Selector (laya_blind_selector) chấm mù không cần trường 'score'
            r_base = runner.run_rollout("base", self.base_tree)
            self.assertNotIn("score", r_base)
            self.assertEqual(r_base["returncode"], 0)

            comp_tree = h2.three_way_merge_trees(
                self.base_tree,
                inv_tree,
                 h2.autotrigger_agent_skills(h2.AgentSurfaceSpec(), dom_tree),
            )
            id_map = {"A": "base", "B": "agg1"}
            picked = h2.laya_blind_selector(
                id_map,
                {"base": self.base_tree, "agg1": comp_tree},
                {
                    "base": {"returncode": 0, "stdout": "base ok"},
                    "agg1": {"returncode": 0, "stdout": "agg1 ok"},
                },
                base_tree=self.base_tree,
                laya_url=live_url,
            )
            self.assertEqual(picked, "B")

            # Nếu agg1 có returncode != 0 -> laya_blind_selector bắt buộc chọn base ('A')
            picked_fail = h2.laya_blind_selector(
                id_map,
                {"base": self.base_tree, "agg1": comp_tree},
                {
                    "base": {"returncode": 0, "stdout": "base ok"},
                    "agg1": {"returncode": 2, "stderr": "runtime failure"},
                },
                base_tree=self.base_tree,
                laya_url=live_url,
            )
            self.assertEqual(picked_fail, "A")

            # 3. RF-04 & RF-05: DEFAULT_LIBRARY_ROOT trỏ tới memory/global/harness_library & H1/H6 wiring
            self.assertEqual(
                h2.DEFAULT_LIBRARY_ROOT,
                h2.AGENT_ROOT / "memory" / "global" / "harness_library",
            )
            boot_out = agy_guard.handle_pre_invocation(
                {"conversationId": "test-rf05-boot"},
                state_dir=self.tmp / "state",
                mem_root=self.tmp / "mem",
            )
            boot_msg = boot_out["injectSteps"][0]["ephemeralMessage"]
            self.assertIn("[Harness² S0] Surface: VALIDATED", boot_msg)
            self.assertIn("run-recursion --use-live-model", agy_guard.ESCALATION_MENTOR_MSG)
        finally:
            srv.shutdown()
            srv.server_close()


if __name__ == "__main__":
    unittest.main()

