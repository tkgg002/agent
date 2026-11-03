# 13 Analysis: TASK-7-M04 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py:36 (def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py` (agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-7-M01, TASK-7-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-7/task-7-m04` (`.orchestrator/worktrees/TASK-7/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/proposer_axes.py (lines 36-60)
  36 | def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:
  37 |     """Đóng AF-01: Hợp nhất 3 chiều bảo toàn thay đổi và phép xóa của cả 2 trục trực giao INV & DOM."""
  38 |     merged: Tree = dict(base_tree)
  39 |     for branch in (inv_tree, dom_tree):
  40 |         deleted = set(base_tree) - set(branch)
  41 |         for k in deleted:
  42 |             merged.pop(k, None)
  43 |         for k, v in branch.items():
  44 |             if k not in base_tree or branch[k] != base_tree[k]:
  45 |                 merged[k] = v
  46 |     return merged
  47 | 
  48 | 
  49 | def validate_orthogonal_axis_edit(
  50 |     dim: str,
  51 |     before: Tree,
  52 |     after: Tree,
  53 |     surface: Optional[Any] = None,
  54 | ) -> Tuple[bool, str]:
  55 |     """Đóng F2, AF-04, AF-06: Kiểm tra rào chắn trực giao INV vs DOM, KEEP_BASE và trần UTF-8 bytes."""
  56 |     always_on_files: Tuple[str, ...] = getattr(
  57 |         surface, "always_on", ("GEMINI.md", "OPERATOR_MAP.md")
  58 |     )
  59 |     always_on_max: int = int(getattr(surface, "always_on_max", 24_000))
  60 | 
```
