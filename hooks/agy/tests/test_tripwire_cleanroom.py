#!/usr/bin/env python3
"""Tests for lesson tripwires (G-08) and clean-room hardening (G-01..G-04) in agy_guard.py."""
from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import agy_guard  # noqa: E402

README_TRIP = {
    "id": "LS-20261020-readme-only",
    "trigger": r"github\.com/[\w.-]+/[\w.-]+",
    "check": "Tra memory/workspaces/*<repo>* và mở source thật trước khi kết luận.",
    "require": r"(git\s+clone|/scratch/)",
}


class TestTripwireAndCleanRoom(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.tmp = Path(self._tmp.name)
        self.mem_root = self.tmp / "memory"
        self.ws_root = self.mem_root / "workspaces"
        self.state_dir = self.tmp / "state"
        self.ws_root.mkdir(parents=True, exist_ok=True)
        self.state_dir.mkdir(parents=True, exist_ok=True)
        lessons_dir = self.mem_root / "global" / "lessons"
        lessons_dir.mkdir(parents=True, exist_ok=True)
        (lessons_dir / "tripwires.json").write_text(json.dumps([README_TRIP]), encoding="utf-8")

    def tearDown(self) -> None:
        self._tmp.cleanup()

    def _tr(self, lines: list) -> str:
        p = self.tmp / "transcript.jsonl"
        p.write_text("\n".join(json.dumps(x) for x in lines) + "\n", encoding="utf-8")
        return str(p)

    # ---------- G-08: tripwires ----------
    def test_g08_match_tripwires_only_on_trigger(self) -> None:
        trips = agy_guard.load_tripwires(self.mem_root)
        self.assertEqual(len(trips), 1)
        hit = agy_guard.match_tripwires("đọc https://github.com/google-research/harness2 giúp", trips)
        self.assertEqual([t["id"] for t in hit], ["LS-20261020-readme-only"])
        self.assertEqual(agy_guard.match_tripwires("giải thích rule 0", trips), [])

    def test_g08_invalid_regex_tripwire_is_skipped(self) -> None:
        bad = [{"id": "X", "trigger": "([", "check": "c"}, README_TRIP]
        hit = agy_guard.match_tripwires("github.com/a/b", bad)
        self.assertEqual([t["id"] for t in hit], ["LS-20261020-readme-only"])

    def test_g08_match_limit(self) -> None:
        many = [dict(README_TRIP, id=f"T{i}") for i in range(9)]
        self.assertEqual(len(agy_guard.match_tripwires("github.com/a/b", many)), 5)

    def test_g08_pre_invocation_injects_check_text(self) -> None:
        tr = self._tr([{"type": "USER_INPUT", "content": "check github.com/morluto/rea đi"}])
        res = agy_guard.handle_pre_invocation(
            {"conversationId": "c1", "transcriptPath": tr}, state_dir=self.state_dir, mem_root=self.mem_root
        )
        msg = res["injectSteps"][0]["ephemeralMessage"]
        self.assertIn("LS-20261020-readme-only", msg)
        self.assertIn(README_TRIP["check"], msg)

    def test_g08_pre_invocation_no_tripwire_section_when_unmatched(self) -> None:
        tr = self._tr([{"type": "USER_INPUT", "content": "giải thích rule 0"}])
        res = agy_guard.handle_pre_invocation(
            {"conversationId": "c2", "transcriptPath": tr}, state_dir=self.state_dir, mem_root=self.mem_root
        )
        self.assertNotIn("LS-20261020-readme-only", res["injectSteps"][0]["ephemeralMessage"])

    def _analysis_write(self, tr: str) -> dict:
        ws = self.ws_root / "aud"
        ws.mkdir(parents=True, exist_ok=True)
        return agy_guard.handle_pre_write(
            {
                "transcriptPath": tr,
                "toolCall": {
                    "name": "write_to_file",
                    "args": {"TargetFile": str(ws / "13_analysis_x.md"), "CodeContent": "# a\n"},
                },
            },
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )

    def test_g08_require_blocks_analysis_without_evidence(self) -> None:
        tr = self._tr(
            [
                {"type": "USER_INPUT", "content": "đã đảm bảo github.com/google-research/harness2 chưa"},
                {"tool_calls": [{"name": "read_url_content", "args": {"Url": "https://github.com/google-research/harness2"}}]},
            ]
        )
        res = self._analysis_write(tr)
        self.assertEqual(res.get("decision"), "deny")
        self.assertIn("LS-20261020-readme-only", res.get("reason", ""))

    def test_g08_require_allows_analysis_after_evidence(self) -> None:
        tr = self._tr(
            [
                {"type": "USER_INPUT", "content": "đã đảm bảo github.com/google-research/harness2 chưa"},
                {"tool_calls": [{"name": "run_command", "args": {"CommandLine": "git clone --depth 1 https://github.com/google-research/harness2 /tmp/h2"}}]},
            ]
        )
        res = self._analysis_write(tr)
        self.assertNotEqual(res.get("decision"), "deny")

    def test_g08_evidence_before_trigger_prompt_does_not_count(self) -> None:
        tr = self._tr(
            [
                {"tool_calls": [{"name": "run_command", "args": {"CommandLine": "git clone x"}}]},
                {"type": "USER_INPUT", "content": "xem github.com/a/b"},
            ]
        )
        self.assertEqual(self._analysis_write(tr).get("decision"), "deny")

    # ---------- G-01: approval gate ----------
    def test_g01_approve_not_matched_mid_sentence(self) -> None:
        self.assertFalse(agy_guard.APPROVE.search("chỉ nhận SPEC.md đã được Jev duyệt có bằng chứng"))
        self.assertTrue(agy_guard.APPROVE.search("ok duyệt làm đi"))
        self.assertTrue(agy_guard.APPROVE.search("APPROVE"))
        self.assertTrue(agy_guard.APPROVE.search("dòng 1\nduyệt"))

    def test_g01_clean_room_requires_exact_start_line(self) -> None:
        st = [{"type": "USER_INPUT", "content": "ok duyệt làm đi"}]
        self.assertTrue(agy_guard.user_approved(st, 0, clean_room=False))
        self.assertFalse(agy_guard.user_approved(st, 0, clean_room=True))
        for txt in ("BẮT ĐẦU", "  bắt đầu.  ", "APPROVE"):
            self.assertTrue(agy_guard.user_approved([{"type": "USER_INPUT", "content": txt}], 0, clean_room=True), txt)
        self.assertFalse(
            agy_guard.user_approved([{"type": "USER_INPUT", "content": "chưa BẮT ĐẦU nhé"}], 0, clean_room=True)
        )

    # ---------- G-02: contaminated reads ----------
    def test_g02_run_command_and_url_reads_detected(self) -> None:
        fb = ["legacy/payment.go"]
        for name, args in (
            ("run_command", {"CommandLine": "cat /repo/legacy/payment.go | head"}),
            ("read_url_content", {"Url": "https://raw.githubusercontent.com/o/r/main/legacy/payment.go"}),
            ("view_file", {"AbsolutePath": "/repo/legacy/payment.go"}),
        ):
            st = [{"tool_calls": [{"name": name, "args": args}]}]
            self.assertEqual(agy_guard.contaminated_clean_room_reads(st, fb), fb, name)
        st_ok = [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls /repo/new"}}]}]
        self.assertEqual(agy_guard.contaminated_clean_room_reads(st_ok, fb), [])

    # ---------- G-03: spec rules ----------
    def test_g03_spec_rules_require_ref(self) -> None:
        ok = "## Clean-Room Approved Spec\n- R1: Trả 400 khi amount<=0 — REF: src/pay.go:10-20\n## Khác\n"
        self.assertEqual(agy_guard.spec_rule_issues(ok), [])
        missing = "## Clean-Room Approved Spec\n- R1: a — REF: x.go:1-2\n- R2: không có bằng chứng\n"
        self.assertEqual(len(agy_guard.spec_rule_issues(missing)), 1)
        self.assertIn("R2", agy_guard.spec_rule_issues(missing)[0])
        empty = "## Clean-Room Approved Spec\nchưa có rule\n## Footer\n- R9: ngoài spec\n"
        self.assertTrue(agy_guard.spec_rule_issues(empty))

    def test_g03_pre_write_denies_solution_with_unreferenced_rule(self) -> None:
        ws = self.ws_root / "cr"
        ws.mkdir(parents=True, exist_ok=True)
        (ws / "01_requirements_cr1.md").write_text("tier: T2\nmode: clean-room\n", encoding="utf-8")
        (ws / "08_tasks_cr1.md").write_text("- [ ] t\n", encoding="utf-8")
        tr = self._tr(
            [
                {"type": "USER_INPUT", "content": "Rebuild feature"},
                {"tool_calls": [{"name": "replace_file_content", "args": {"TargetFile": str(ws / "05_progress_cr1.md")}}]},
            ]
        )
        body = "# S\n## Clean-Room Approved Spec\n- R1: hành vi không bằng chứng\n"
        res = agy_guard.handle_pre_write(
            {"transcriptPath": tr, "toolCall": {"name": "write_to_file", "args": {"TargetFile": str(ws / "09_tasks_solution_cr1.md"), "CodeContent": body}}},
            mem_root=self.mem_root,
            ws_root=self.ws_root,
        )
        self.assertEqual(res.get("decision"), "deny")
        self.assertIn("R1", res.get("reason", ""))

    # ---------- G-04: subagent prompt leak ----------
    def test_g04_leaked_subagent_prompt_by_path_and_content(self) -> None:
        legacy = self.tmp / "legacy_pay.py"
        legacy.write_text("def charge(amount):\n    return gateway.capture(amount, currency='VND', idempotent=True)\n", encoding="utf-8")
        fb = [str(legacy)]
        by_path = [{"tool_calls": [{"name": "invoke_subagent", "args": {"Subagents": [{"Prompt": f"đọc {legacy} rồi viết lại"}]}}]}]
        self.assertEqual(agy_guard.leaked_subagent_prompts(by_path, fb), fb)
        by_body = [{"tool_calls": [{"name": "invoke_subagent", "args": {"Subagents": [{"Prompt": "code: return gateway.capture(amount, currency='VND', idempotent=True)"}]}}]}]
        self.assertEqual(agy_guard.leaked_subagent_prompts(by_body, fb), fb)
        clean = [{"tool_calls": [{"name": "invoke_subagent", "args": {"Subagents": [{"Prompt": "Chỉ dùng Clean-Room Approved Spec R1..R3"}]}}]}]
        self.assertEqual(agy_guard.leaked_subagent_prompts(clean, fb), [])


    def _real_doc(self, ws: str, name: str) -> str:
        p = self.mem_root / "workspaces" / ws / name
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text("# doc\n", encoding="utf-8")
        return str(p)

    # ---------- AF-1: denied write must not self-satisfy Require ----------
    def test_af1_write_tools_are_not_evidence(self) -> None:
        trip = dict(README_TRIP, require=r"/memory/workspaces/[^\"\s]+/13_analysis_[^\"\s]*\.md")
        own = self._real_doc("w", "13_analysis_s.md")
        st = [
            {"type": "USER_INPUT", "content": "xem github.com/a/b"},
            {"tool_calls": [{"name": "write_to_file", "args": {"TargetFile": own}}]},
        ]
        self.assertEqual([t["id"] for t in agy_guard.tripwire_violations(st, [trip])], [trip["id"]])
        old = self._real_doc("old", "13_analysis_o.md")
        st.append({"tool_calls": [{"name": "view_file", "args": {"AbsolutePath": old}}]})
        self.assertEqual(agy_guard.tripwire_violations(st, [trip]), [])

    # ---------- AF-4: evidence pointing at target workspace or unexpanded $VAR does not count ----------
    def test_af4_target_workspace_and_shell_var_not_evidence(self) -> None:
        trip = dict(README_TRIP, require=r"/memory/workspaces/[^\"\s]+/09_tasks_solution_[^\"\s]*\.md")
        base = [{"type": "USER_INPUT", "content": "xem github.com/a/b"}]
        cur = self._real_doc("cur-ws", "09_tasks_solution_s.md")
        own = base + [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": f"cat {cur}"}}]}]
        self.assertEqual(len(agy_guard.tripwire_violations(own, [trip], exclude_ws="cur-ws")), 1)
        var = base + [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "cat /x/memory/workspaces/$WS/09_tasks_solution_s.md"}}]}]
        self.assertEqual(len(agy_guard.tripwire_violations(var, [trip], exclude_ws="cur-ws")), 1)
        old = self._real_doc("old-ws", "09_tasks_solution_o.md")
        prior = own + [{"tool_calls": [{"name": "view_file", "args": {"AbsolutePath": old}}]}]
        self.assertEqual(agy_guard.tripwire_violations(prior, [trip], exclude_ws="cur-ws"), [])

    # ---------- AF-5: absolute path token that does not exist on disk is not evidence ----------
    def test_af5_nonexistent_path_string_not_evidence(self) -> None:
        trip = dict(README_TRIP, require=r"/memory/workspaces/[^\"\s]+/13_analysis_[^\"\s]*\.md")
        base = [{"type": "USER_INPUT", "content": "xem github.com/a/b"}]
        fake = base + [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "python3 -c \"x='/nope/memory/workspaces/w/13_analysis_s.md'\""}}]}]
        self.assertEqual(len(agy_guard.tripwire_violations(fake, [trip])), 1)
        real = self._real_doc("w2", "13_analysis_r.md")
        ok = base + [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": f"head -50 {real}; echo done"}}]}]
        self.assertEqual(agy_guard.tripwire_violations(ok, [trip]), [])
        # Non-path evidence (git clone) vẫn được chấp nhận
        clone = base + [{"tool_calls": [{"name": "run_command", "args": {"CommandLine": "git clone --depth 1 https://github.com/a/b /tmp/b"}}]}]
        self.assertEqual(agy_guard.tripwire_violations(clone, [README_TRIP]), [])

    # ---------- AF-2: relative forbidden path resolved via run_command Cwd ----------
    def test_af2_relative_forbidden_content_leak_detected(self) -> None:
        proj = self.tmp / "proj"
        (proj / "legacy").mkdir(parents=True)
        (proj / "legacy" / "pay.py").write_text(
            "def charge(amount):\n    return gateway.capture(amount, currency='VND', idempotent=True)\n", encoding="utf-8"
        )
        st = [
            {"tool_calls": [{"name": "run_command", "args": {"CommandLine": "ls", "Cwd": str(proj)}}]},
            {"tool_calls": [{"name": "invoke_subagent", "args": {"Subagents": [{"Prompt": "return gateway.capture(amount, currency='VND', idempotent=True)"}]}}]},
        ]
        self.assertEqual(agy_guard.leaked_subagent_prompts(st, ["legacy/pay.py"]), ["legacy/pay.py"])


if __name__ == "__main__":
    unittest.main()
