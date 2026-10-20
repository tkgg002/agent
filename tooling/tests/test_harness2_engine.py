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


if __name__ == "__main__":
    unittest.main()
