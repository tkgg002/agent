# 13 Analysis: TASK-4-M04 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py:32 (def sha256_bytes(data: bytes) -> str:)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py` (agent:memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: `TASK-4-M01, TASK-4-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-4/task-4-m04` (`.orchestrator/worktrees/TASK-4/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/global/harness_library/agent_governance/harnesses/harness2_s0_s6_live_model_run/harness/tooling/mutation_gate.py (lines 32-56)
  32 | def sha256_bytes(data: bytes) -> str:
  33 |     return hashlib.sha256(data).hexdigest()
  34 | 
  35 | 
  36 | def _invalidate_pyc(fpath: Path) -> None:
  37 |     if fpath.suffix != ".py":
  38 |         return
  39 |     pycache = fpath.parent / "__pycache__"
  40 |     if pycache.is_dir():
  41 |         for pyc in pycache.glob(f"{fpath.stem}.*.pyc"):
  42 |             try:
  43 |                 pyc.unlink()
  44 |             except OSError:
  45 |                 pass
  46 | 
  47 | 
  48 | def resolve_target_file(raw_path: str, mutations_file: Path) -> Optional[Path]:
  49 |     p = Path(raw_path.strip())
  50 |     if p.is_absolute() and p.is_file():
  51 |         return p.resolve()
  52 |     for base in (Path.cwd(), AGENT_ROOT, mutations_file.resolve().parent):
  53 |         cand = (base / p).resolve()
  54 |         if cand.is_file():
  55 |             return cand
  56 |     return None
```
