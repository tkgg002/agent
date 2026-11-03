# 13 Analysis: TASK-2-M03 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:90 (def _has_skill_trigger_token(body: str, sname: str) -> bool:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-2-M01, TASK-2-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-2/task-2-m03` (`.orchestrator/worktrees/TASK-2/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py (lines 90-114)
  90 | def _has_skill_trigger_token(body: str, sname: str) -> bool:
  91 |     """Đóng AF-04: Kiểm tra tên skill theo ranh giới từ/token, tránh dương tính giả do chuỗi con."""
  92 |     return bool(re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body))
  93 | 
  94 | 
  95 | # ============================================================================
  96 | # Giải pháp xử lý Khoảng trống F3 (Task 3 & AF-04):
  97 | # Bộ Kiểm định Bề mặt Cấu hình /agent (AgentSurfaceValidator & Auto-Trigger)
  98 | # ============================================================================
  99 | @dataclass(frozen=True)
 100 | class AgentSurfaceSpec:
 101 |     name: str = "agy_agent"
 102 |     always_on: Tuple[str, ...] = ("GEMINI.md", "OPERATOR_MAP.md")
 103 |     always_on_max: int = 24_000
 104 |     skill_dir: str = "skills"
 105 |     autotrigger_file: str = "OPERATOR_MAP.md"
 106 | 
 107 | 
 108 | def load_agent_surface_tree(agent_root: Path = AGENT_ROOT) -> Tree:
 109 |     """Đọc cây cấu hình /agent thực tế từ đĩa phục vụ kiểm định bề mặt F3."""
 110 |     tree: Tree = {}
 111 |     for rel in ("GEMINI.md", "OPERATOR_MAP.md"):
 112 |         p = agent_root / rel
 113 |         if p.is_file():
 114 |             tree[rel] = p.read_text(encoding="utf-8")
```
