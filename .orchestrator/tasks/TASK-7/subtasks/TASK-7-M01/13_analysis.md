# 13 Analysis: TASK-7-M01 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py:84 (def _extract_yaml_frontmatter(src: str) -> str:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py` (agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 0`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: ``) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-7/task-7-m01` (`.orchestrator/worktrees/TASK-7/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/orchestrator/harness2_engine.py (lines 84-108)
  84 | def _extract_yaml_frontmatter(src: str) -> str:
  85 |     """Đóng AF-04: Tách chính xác khối YAML frontmatter nằm giữa cặp dấu --- ở đầu file."""
  86 |     m = _FRONTMATTER_RE.match(src)
  87 |     return m.group(1) if m else ""
  88 | 
  89 | 
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
```
