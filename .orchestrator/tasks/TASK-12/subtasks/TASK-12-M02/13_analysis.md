# 13 Analysis: TASK-12-M02 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py:83 (class LessonEntry:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py` (missing on disk and not in diff.patch: agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py) |
| Hợp đồng Đầu vào (`Wave 0`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: ``) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-12/task-12-m02` (`.orchestrator/worktrees/TASK-12/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/build_lessons_index.py (lines 83-107)
  83 | class LessonEntry:
  84 |     idx: int
  85 |     date: str
  86 |     title: str
  87 |     line_no: int
  88 |     raw_block: str
  89 |     topic_code: str
  90 |     topic_file: str
  91 |     short_label: str
  92 | 
  93 | 
  94 | # G-08: lesson tripwire fields (máy đọc được) — ID ổn định, Trigger regex, Check tiền điều kiện, Require bằng chứng tuỳ chọn.
  95 | TRIP_FIELD_RE = re.compile(r"^- \*\*(ID|Trigger|Check|Require):\*\*\s*(.+?)\s*$", re.M)
  96 | TRIP_REQUIRED = ("id", "trigger", "check")
  97 | 
  98 | 
  99 | def parse_tripwire(entry: "LessonEntry") -> Optional[Dict[str, str]]:
 100 |     fields = {k.lower(): v for k, v in TRIP_FIELD_RE.findall(entry.raw_block)}
 101 |     if not fields:
 102 |         return None
 103 |     missing = [k for k in TRIP_REQUIRED if not fields.get(k)]
 104 |     if missing:
 105 |         raise RuntimeError(f"Tripwire thiếu {missing} tại lessons.md:L{entry.line_no}")
 106 |     for key in ("trigger", "require"):
 107 |         if fields.get(key):
```
