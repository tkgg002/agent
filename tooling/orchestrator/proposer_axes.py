#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""proposer_axes.py — Orthogonal Proposer Axes (INV vs DOM), Context Hierarchy & 3-Way Tree Merge (F2, AF-01, AF-06).

Implements Task 2 & Task 7 from 08_tasks_harness2_recursive_integration.md:
  - INV Proposer (Task-Invariant Workflow/Tool Lever): allowed to edit tooling/, scripts/, hooks/agy/, workflow rules;
    strictly prohibited from touching skills/* or memory/global/project/*.
  - DOM Proposer (Domain Practitioner Lever): allowed to edit skills/<name>/SKILL.md and memory/global/project/*;
    strictly prohibited from editing hooks/agy/*, tooling/*, scripts/*, or always-on core files directly.
  - three_way_merge_trees(): 3-way delta merge preserving both INV and DOM edits on pre-existing files as well as
    intentional file deletions relative to base_tree (resolving AF-01).
"""
from __future__ import annotations

from pathlib import Path
import sys
from typing import Any, Dict, Optional, Tuple

AGENT_ROOT = Path(__file__).resolve().parents[2]
if str(AGENT_ROOT) not in sys.path:
    sys.path.insert(0, str(AGENT_ROOT))

from hooks.agy.agy_guard import PROD_MOCK_POLLUTION, SRC_EXT  # noqa: E402

Tree = Dict[str, str]

CONTEXT_HIERARCHY_ORDER: Tuple[str, ...] = (
    "SKILL",
    "SCRIPT",
    "SUBAGENT",
    "HOOK",
    "ALWAYS_ON",
)


def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:
    """Đóng AF-01: Hợp nhất 3 chiều bảo toàn thay đổi và phép xóa của cả 2 trục trực giao INV & DOM."""
    merged: Tree = dict(base_tree)
    for branch in (inv_tree, dom_tree):
        deleted = set(base_tree) - set(branch)
        for k in deleted:
            merged.pop(k, None)
        for k, v in branch.items():
            if k not in base_tree or branch[k] != base_tree[k]:
                merged[k] = v
    return merged


def validate_orthogonal_axis_edit(
    dim: str,
    before: Tree,
    after: Tree,
    surface: Optional[Any] = None,
) -> Tuple[bool, str]:
    """Đóng F2, AF-04, AF-06: Kiểm tra rào chắn trực giao INV vs DOM, KEEP_BASE và trần UTF-8 bytes."""
    always_on_files: Tuple[str, ...] = getattr(
        surface, "always_on", ("GEMINI.md", "OPERATOR_MAP.md")
    )
    always_on_max: int = int(getattr(surface, "always_on_max", 24_000))

    changed = sorted(
        (set(after) - set(before))
        | (set(before) - set(after))
        | {k for k in set(before) & set(after) if before[k] != after[k]}
    )
    if not changed:
        return True, "KEEP_BASE"

    always_on_body = "\n\n".join(after.get(a, "") for a in always_on_files)
    always_on_bytes = len(always_on_body.encode("utf-8"))
    if always_on_bytes > always_on_max:
        return False, f"ALWAYS_ON_OVER_BUDGET: {always_on_bytes} bytes > {always_on_max}"

    for rel in changed:
        if dim == "INV" and (
            rel.startswith("memory/global/project/") or rel.startswith("skills/")
        ):
            return False, f"INV Proposer không được sửa skill hoặc tri thức nghiệp vụ dự án: {rel}"
        if dim == "DOM" and (
            rel.startswith("hooks/agy/")
            or rel.startswith("tooling/")
            or rel.startswith("scripts/")
            or rel in always_on_files
        ):
            return False, f"DOM Proposer không được sửa script/hook/always-on lõi: {rel}"
        is_test = (
            rel.endswith("_test.py")
            or "/tests/" in f"/{rel}"
            or Path(rel).name.startswith("test_")
        )
        if (
            Path(rel).suffix in SRC_EXT
            and not is_test
            and PROD_MOCK_POLLUTION.search(after.get(rel, ""))
        ):
            return False, f"H4b DENIED: Phát hiện mock pollution trong {rel}"
    return True, f"VALID_{dim}_EDIT ({len(changed)} files)"
