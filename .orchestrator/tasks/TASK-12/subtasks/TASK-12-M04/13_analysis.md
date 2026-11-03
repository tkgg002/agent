# 13 Analysis: TASK-12-M04 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py:89 (def norm_ws(text: str) -> str:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py` (missing on disk and not in diff.patch: agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-12-M01, TASK-12-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-12/task-12-m04` (`.orchestrator/worktrees/TASK-12/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/laya_gate.py (lines 89-113)
  89 | def norm_ws(text: str) -> str:
  90 |     return re.sub(r"\s+", " ", text).strip()
  91 | 
  92 | 
  93 | def now_iso() -> str:
  94 |     return dt.datetime.now(dt.timezone(dt.timedelta(hours=7))).isoformat(timespec="seconds")
  95 | 
  96 | 
  97 | def resolve_ref_path(raw_path: str, base_dirs: List[Path]) -> Optional[Path]:
  98 |     p = Path(raw_path)
  99 |     if p.is_absolute():
 100 |         return p if p.is_file() else None
 101 |     for base in base_dirs:
 102 |         cand = (base / p).resolve()
 103 |         if cand.is_file():
 104 |             return cand
 105 |     return None
 106 | 
 107 | 
 108 | def verify_evidence(claim_obj: Dict[str, Any], base_dirs: Optional[List[Path]] = None) -> Tuple[bool, str]:
 109 |     """Verify that every REF: path:start-end exists and its quoted lines match the actual file."""
 110 |     bases = base_dirs or [Path.cwd(), AGENT_ROOT, AGENT_ROOT.parent]
 111 |     evidence = claim_obj.get("evidence", "") or ""
 112 |     blocks = re.split(r"(?m)^(?=REF:|DEMO:)", evidence)
 113 |     refs = [b for b in blocks if b.startswith("REF:")]
```
