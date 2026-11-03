# 13 Analysis: TASK-8-M03 (`agent`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py:1 (import os)` | `VERIFIED` | `/Users/trainguyen/Documents/work/agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py` (missing on disk and not in diff.patch: agent:memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-8-M01, TASK-8-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-8/task-8-m03` (`.orchestrator/worktrees/TASK-8/agent`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/agent/memory/workspaces/cds-project-inventory-2026-06-18/refactor-plan/analyze_dependencies.py (lines 1-25)
   1 | import os
   2 | import re
   3 | 
   4 | service_dir = "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service"
   5 | files = [f for f in os.listdir(service_dir) if f.endswith(".go") and not f.endswith("_test.go")]
   6 | 
   7 | # Map symbol -> defining file
   8 | symbol_to_file = {}
   9 | file_defs = {}
  10 | 
  11 | struct_re = re.compile(r"type\s+([A-Z][a-zA-Z0-9_]*)\s+(struct|interface)")
  12 | func_re = re.compile(r"func\s+([A-Z][a-zA-Z0-9_]*)\(")
  13 | 
  14 | # Target sub-packages mapping
  15 | subpackages = {
  16 |     # source
  17 |     "metadata_registry_service.go": "source",
  18 |     "registry_service.go": "source",
  19 |     "connection_manager.go": "source",
  20 |     "connection_overrides.go": "source",
  21 |     "connector_resolver.go": "source",
  22 |     "source_router.go": "source",
  23 |     "mongo_introspection.go": "source",
  24 |     "scan_service.go": "source",
  25 |     # shadow
```
