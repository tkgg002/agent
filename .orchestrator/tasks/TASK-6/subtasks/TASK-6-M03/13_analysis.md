# 13 Analysis: TASK-6-M03 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/hooks/agy/tests/test_agy_guard.py:15 (class TestAgyGuard(unittest.TestCase):)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py` (agent:hooks/agy/tests/test_agy_guard.py (verified file on disk)) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: `TASK-6-M01, TASK-6-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-6/task-6-m03` (`.orchestrator/worktrees/TASK-6/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/hooks/agy/tests/test_agy_guard.py (lines 15-39)
  15 | class TestAgyGuard(unittest.TestCase):
  16 |     def setUp(self) -> None:
  17 |         self._tmp = tempfile.TemporaryDirectory()
  18 |         self.tmp = Path(self._tmp.name)
  19 |         self.mem_root = self.tmp / "memory"
  20 |         self.ws_root = self.mem_root / "workspaces"
  21 |         self.state_dir = self.tmp / "state"
  22 |         self.ws_root.mkdir(parents=True, exist_ok=True)
  23 |         self.state_dir.mkdir(parents=True, exist_ok=True)
  24 | 
  25 |     def tearDown(self) -> None:
  26 |         self._tmp.cleanup()
  27 | 
  28 |     def _write_transcript(self, lines: list[dict]) -> str:
  29 |         tr = self.tmp / "transcript.jsonl"
  30 |         tr.write_text("\n".join(json.dumps(x) for x in lines) + "\n", encoding="utf-8")
  31 |         return str(tr)
  32 | 
  33 |     # 1. H2: Overwrite existing memory file -> deny
  34 |     def test_01_h2_overwrite_existing_memory_denied(self) -> None:
  35 |         lessons = self.mem_root / "global" / "lessons.md"
  36 |         lessons.parent.mkdir(parents=True, exist_ok=True)
  37 |         lessons.write_text("# Lessons\n", encoding="utf-8")
  38 |         res = agy_guard.handle_pre_write(
  39 |             {
```
