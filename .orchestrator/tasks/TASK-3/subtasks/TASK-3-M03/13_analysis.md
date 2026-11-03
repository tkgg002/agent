# 13 Analysis: TASK-3-M03 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py:163 (def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py` (agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: `TASK-3-M01, TASK-3-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-3/task-3-m03` (`.orchestrator/worktrees/TASK-3/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/hooks/agy/agy_guard.py (lines 163-187)
 163 | def steps(payload: Dict[str, Any]) -> List[Dict[str, Any]]:
 164 |     tr = payload.get("transcriptPath", "")
 165 |     if not tr:
 166 |         return []
 167 |     try:
 168 |         with open(tr, encoding="utf-8") as f:
 169 |             return [json.loads(line) for line in f if line.strip()]
 170 |     except OSError:
 171 |         return []
 172 | 
 173 | 
 174 | def calls(st: List[Dict[str, Any]]) -> Iterator[Tuple[int, Optional[str], Dict[str, Any]]]:
 175 |     for i, s in enumerate(st):
 176 |         for tc in s.get("tool_calls") or []:
 177 |             raw_args = tc.get("args") or {}
 178 |             clean_args = {k: _unquote(v) for k, v in raw_args.items()}
 179 |             yield i, tc.get("name"), clean_args
 180 | 
 181 | 
 182 | def last_user_idx(st: List[Dict[str, Any]]) -> int:
 183 |     for i in range(len(st) - 1, -1, -1):
 184 |         if st[i].get("type") == "USER_INPUT":
 185 |             return i
 186 |     return -1
 187 | 
```
