# 09 — Technical Solution: Hợp Nhất Kiến Trúc 3 Tầng — `Lõi /agent Kernel` + `Băng Chuyền Queue-Driven` + `Động Cơ Tự Tiến Hóa Đệ Quy Harness²` (`harness2_recursive_integration`)

> **Trạng thái:** `PROPOSED` — Chờ User `APPROVE` trước khi thực thi (Rule #1, Rule #13).
> **Workspace:** `feature-harness2-recursive-integration-2026-10-20` | **Tier:** `T2` | **Suffix:** `harness2_recursive_integration`

---

## 0. Bức tranh Tổng quan Vận hành Vật lý (`Macro Runtime & Integration Topology` — Tuân thủ Lesson `L001`)

Để tránh mọi sự nhầm lẫn giữa việc **gõ chat tương tác 1-1 trên cửa sổ Antigravity IDE hiện tại** và việc **vận hành hệ thống tự động hóa đa tác nhân tích hợp `Harness²`**, mục này định nghĩa rõ bức tranh vận hành vật lý từ A–Z:

### 0.1. Bốn Câu hỏi Định vị Vật lý Cốt lõi
1. **Hệ thống sau khi tích hợp `Harness²` gồm những khối vật lý nào?**
   - Hệ thống được tổ chức thành **Kiến trúc Đồng tâm 3 Tầng (3-Tier Concentric Architecture)**:
     - **Tầng 1 — `Inner /agent Governance Kernel` (Bộ Khung Võ Giáp `c1..c8`):** Nằm tại `/Users/trainguyen/Documents/work/agent/` (`GEMINI.md`, `OPERATOR_MAP.md`, `skills/*/SKILL.md`, `hooks/agy/agy_guard.py`, `tooling/*.py`, `memory/global/lessons/INDEX.md`). Đây là bộ luật và công cụ được gắn vào mọi Agent (cả khi anh chat trên Antigravity IDE lẫn khi chạy Daemon tự động).
     - **Tầng 2 — `Middle Queue-Driven Agent Factory` (Nhà Máy Băng Chuyền Hàng Đợi — Phiên 2):** Là Daemon độc lập chạy ngầm bên ngoài IDE (`Scout → Sorting → Planner [Opus 5.5 + Wave 0..N DAG] → Coder [Dynamic Opus/Sonnet + Git Worktree] → Reviewer/QA [Autopilot Loop]`).
     - **Tầng 3 — `Outer Harness² Recursive Optimizer` (Động Cơ Tự Tiến Hóa Đệ Quy 2 Cấp — Phiên 3):** Là tầng Meta-Agent nằm bọc ngoài Tầng 1 và Tầng 2. Khi một Task ở Tầng 2 bị kẹt/trượt QA (hoặc khi kích hoạt chế độ tự rèn luyện trên một domain mới), Tầng 3 tự động phân tích trajectory thất bại, tinh chỉnh chính bộ `c1..c8` của Tầng 1 (`SKILL.md`, `scripts/*.py`, `hooks`), chạy thử nghiệm đối kháng (`Probe`) và lưu bộ cấu hình thắng cuộc vào `memory/global/harness_library/<domain>/INDEX.md`.
2. **Mỗi con Agent trong Tầng 3 (`Harness²`) chạy bằng tiến trình OS gì?**
   - Không chạy trong cửa sổ chat IDE của anh. Nó là các tiến trình CLI độc lập (`claude -p --output-format json`, `codex exec --ephemeral --json -C <ws>`, hoặc `opencode run --format json`) được khởi tạo bởi `Proposer` (`proposer.py:210-312`) bên trong thư mục cách ly tạm thời (`tempfile.mkdtemp(prefix="harness2-prop-")`), đọc dữ liệu từ cây thư mục vật lý `context/` trên đĩa và sửa trực tiếp thư mục `context/harness/`.
3. **Nó gắn vào Dự án đích (`Target Repo`, như `data-hub`) bằng cơ chế vật lý nào?**
   - Đọc manifest `<project-root>/.agent-project.json` (Rule #3 / ADR-07) để biết lệnh `checks`, `test`, `runtime_check` của từng component, và cắt nhánh `git worktree` riêng biệt cho từng ứng viên chạy thử (`Probe`: `inv1..invk`, `dom1..domk`, `agg1..aggk`) để không bao giờ làm bẩn nhánh `main` hay giẫm chân lên nhau.
4. **Cơ chế gắn Kỷ luật `/agent` vào Động cơ Đệ quy `Harness²` diễn ra như thế nào?**
   - Mọi bản đề xuất sửa `Harness` (`INV` hoặc `DOM`) không chỉ phải vượt qua bộ kiểm định bề mặt `AgentSurfaceValidator` (`tree.py:336-390`) mà còn **BẮT BUỘC** đi qua 3 chốt chặn của `/agent`: (1) Chốt tiền kiểm phòng sạch `assert_no_leak_paths` + `H3 Secret Guard`; (2) Cổng thẩm định toán học `/laya_router` (`DG1/DG2` cho đề xuất & `DG3` cho kết quả chọn lọc); và (3) Cổng đột biến `3-Mutant Gate` (`mutation_gate.py` $\ge 3$ mutants `KILLED`) đối với mọi script/code sinh ra!

---

## 1. Root Cause / 6 Khoảng trống Kiến trúc Cốt lõi (`F1–F6` — `VERIFIED`)

Từ báo cáo phân tích đối sánh tại `13_analysis_harness2_recursive_integration.md:97-159`, có 6 khoảng trống kiến trúc giữa hệ thống hiện tại và `google-research/harness2`:
- **`F1` (`GEMINI.md:46-50`):** Cơ chế tự học đang là phản ứng thủ công sau khi User nhắc nhở (`Mid-Session Fix`), thiếu vòng lặp đệ quy 2 tầng tự động (`Task-Level Propose–Probe–Compose` + `Harnessing-Level Domain Library` lưu kèm cả cây cấu hình thắng cuộc).
- **`F2` (`tooling/build_lessons_index.py:18-22`, `GEMINI.md:56-57`):** Chưa tách biệt trực giao 2 trục đề xuất **`INV` (Workflow/Tool/Script/Hook)** vs **`DOM` (Domain Practitioner `SKILL.md`)** và chưa cưỡng chế thứ bậc ưu tiên `Skill → Script → Subagent → Hook → Always-On` để bảo vệ ngân sách token của `GEMINI.md`.
- **`F3` (`tooling/verify_governance.py:28-33, 59-61`):** Linter hiện tại mới kiểm tra hồ sơ task trong `memory/workspaces/<ws>/`, chưa có **`AgentSurfaceValidator & Auto-Trigger Compiler`** để kiểm định chính cây cấu hình `/agent` (trần ký tự `always_on_max`, frontmatter `SKILL.md`, dòng Trigger trong `GEMINI.md`/`OPERATOR_MAP.md`, và tính hợp lệ + biên dịch AST của mọi script `tooling/*.py` được tham chiếu).
- **`F4` (`hooks/agy/agy_guard.py:282-301`):** Cách ly phòng sạch (`Clean-Room`) hiện tại mới kiểm tra hậu kiểm trên transcript (`contaminated_clean_room_reads`), thiếu bộ đóng gói cây bằng chứng vật lý `ContextTree` kèm chốt chặn **tiền kiểm (`Pre-Flight Leak & Grader Verdict Tripwire`)** và rào chắn chống Prompt Injection từ dữ liệu (`EVIDENCE_BOUNDARY`).
- **`F5` (`tooling/laya_gate.py:44-46`, `tooling/mutation_gate.py:27-30`):** Thiếu cơ chế **Tuyển chọn Ẩn danh Chống Thiên kiến (`Blind Anonymized Selection` `A, B, C...`)** và chỉ số đo **Độ đa dạng cấu hình (`Pairwise Line-Jaccard Diversity`)** khi so sánh nhiều ứng viên hoặc đối chiếu với bản gốc (`KEEP_BASE`).
- **`F6` (`hooks/agy/agy_guard.py:39-40`, `tooling/laya_gate.py:10-15`):** Thiếu **Hằng đẳng thức Kiểm toán Ngân sách Tất định (`Ledger.assert_identity: R=4k, P=3k`)**, bộ nhớ đệm phiên chạy theo mã băm nội dung SHA-256 (`_call_fingerprint`) và khóa ghi nguyên tử đa tiến trình (`fcntl.flock`).

---

## 2. Giải pháp Kỹ thuật Duy nhất: Ánh xạ 1:1 Xử lý Triệt để Từng Khoảng trống `F1 → F6` (Tuân thủ Lesson `L002`)

### 2.1. Giải pháp xử lý Khoảng trống `F1`: Động cơ Đệ quy 2 Tầng (`RecursiveHarnessEngine` & `DomainHarnessLibrary`)
- **Vị trí module thiết kế:** `tooling/orchestrator/harness2_engine.py` (kết nối trực tiếp với `tooling/build_lessons_index.py:18-22` và `GEMINI.md:46-50`).
- **Cơ chế hoạt động:**
  1. **Đệ quy Cấp 1 (`Task-Level Propose–Probe–Compose`):** Khi một task ở Tầng 2 bị trượt ở bước `Reviewer/QA` hoặc được đánh dấu cần tinh chỉnh đệ quy với ngân sách $k \ge 1$, `RecursiveHarnessEngine` hỗ trợ 2 chế độ:
     - **`mode="agg"` (Song song / Mở rộng chiều ngang):** Từ $k$ lượt chạy cơ sở (`base1..basek`), khởi chạy song song $2k$ phiên đề xuất (`inv1..invk`, `dom1..domk`), thực thi thử nghiệm (`Probe`) trên các `git worktree` độc lập, rồi chạy $k$ phiên tổng hợp (`agg1..aggk`) đối chiếu diff + trajectory của toàn bộ pool.
     - **`mode="seq"` (Tuần tự / Đào sâu chiều dọc):** Chạy $k$ vòng tinh chỉnh nối tiếp (`c1..ck`), mỗi vòng kế thừa cây cấu hình và lịch sử bằng chứng của vòng trước.
  2. **Đệ quy Cấp 2 (`Harnessing-Level Reflector & Domain Library`):**
     - Sau khi chọn được bản thắng cuộc (`sel.chosen_label`), bộ phản tỉnh 추출 bài học có bằng chứng (`cue`, `lesson`, `evidence`, `confidence`, `edits`), tự động định dạng theo chuẩn 5 phần `A/B/X/Y` của `/agent` (Rule #6) để cập nhật `lessons/INDEX.md` (`build_lessons_index.py:18-22`) **đồng thời** lưu trọn bộ cây `/agent` đã tinh chỉnh vào `memory/global/harness_library/<domain>/harnesses/<task_id>/harness/` kèm `<domain>/INDEX.md`.

### 2.2. Giải pháp xử lý Khoảng trống `F2`: Phân tách 2 Trục Đề xuất Trực giao (`INV` vs `DOM`) & Cưỡng chế Thứ bậc Bảo toàn Ngân sách `Always-On`
- **Vị trí module thiết kế:** Hàm `validate_orthogonal_axis_edit()` trong `tooling/orchestrator/harness2_engine.py` (bảo vệ `GEMINI.md:28-30` & `tooling/build_lessons_index.py:18-22`).
- **Cơ chế hoạt động:**
  1. **Cô lập quyền ghi theo trục (`Axis Write-Fence`):**
     - **`dim == "INV"` (Workflow/Tool Invariant):** Chỉ được phép thêm/sửa file trong `tooling/`, `scripts/`, `hooks/agy/`, hoặc quy trình chung; nếu sửa vào `skills/<domain_specific>/` hoặc `memory/global/project/` $\rightarrow$ từ chối ngay lập tức (`ValueError`).
     - **`dim == "DOM"` (Domain Practitioner Knowledge):** Chỉ được phép tạo/sửa `skills/<name>/SKILL.md` và `memory/global/project/<project_name>/`; cấm sửa `hooks/agy/agy_guard.py` hoặc `tooling/*.py` lõi.
     - **`dim in ("AGG", "COMPOSE")`:** Được phép kết hợp các thay đổi hợp lệ từ `INV` và `DOM`.
  2. **Thứ bậc Ưu tiên & Quy tắc `KEEP_BASE`:**
     - Áp trần cứng `always_on_max = 24000` ký tự cho nhóm file `always_on` (`GEMINI.md`, `OPERATOR_MAP.md`). Mọi tri thức quy trình có điều kiện kích hoạt rõ ràng bắt buộc phải đóng gói thành `skills/<name>/SKILL.md` (nạp theo nhu cầu), thao tác kiểm tra lặp lại đóng gói thành `tooling/<name>.py`, và chốt chặn vòng lặp đưa vào `hooks/agy/agy_guard.py`. Nếu bằng chứng thực thi không chỉ ra lỗi cần sửa cấu hình, trả về `KEEP_BASE` (`kept_base = True`).

### 2.3. Giải pháp xử lý Khoảng trống `F3`: Bộ Kiểm định Bề mặt Cấu hình `/agent` (`AgentSurfaceValidator & Auto-Trigger Compiler`)
- **Vị trí module thiết kế:** Class `AgentSurfaceSpec` và hàm `validate_agent_surface()` / `autotrigger_agent_skills()` trong `tooling/orchestrator/harness2_engine.py` (mở rộng năng lực của `tooling/verify_governance.py:28-69`).
- **Cơ chế hoạt động:**
  1. Định nghĩa `AGENT_SURFACE_SPEC` cho `/Users/trainguyen/Documents/work/agent/` gồm:
     - `always_on = ("GEMINI.md", "OPERATOR_MAP.md")`, `always_on_max = 24000`.
     - `skill_dir = "skills"`, `autotrigger_file = "OPERATOR_MAP.md"`.
  2. Thực hiện 5 phép kiểm định tất định trước mọi lần chạy:
     - **(a) Kiểm tra trần `always_on_max`:** Tổng độ dài `GEMINI.md` + `OPERATOR_MAP.md` không vượt ngưỡng cho phép.
     - **(b) Kiểm tra Frontmatter `SKILL.md`:** Mọi file `skills/<name>/SKILL.md` bắt buộc có `name: <name>` khớp 100% tên thư mục và `description:` không rỗng.
     - **(c) Kiểm tra & Tự động vá Dòng Kích hoạt (`Auto-Trigger`):** Tên của mọi skill trong `skills/<name>/SKILL.md` bắt buộc phải xuất hiện trong văn bản `always_on` (`GEMINI.md` hoặc `OPERATOR_MAP.md`), chống triệt để lỗi "skill mồ côi (`dead weight` — Lesson `L009`)".
     - **(d) Kiểm tra Tham chiếu Script (`_check_script_refs`):** Quét mọi lệnh gọi `tooling/<name>.py` hoặc `hooks/agy/<name>.py` trong `GEMINI.md` và `skills/*/SKILL.md`, đảm bảo 100% file được gọi đều tồn tại thật trên đĩa.
     - **(e) Kiểm tra Biên dịch AST Python (`_py_compiles`):** Chạy `compile(src, rel, "exec")` trên toàn bộ các file `.py` trong cây cấu hình để bắt sống mọi lỗi cú pháp trước khi rollout.

### 2.4. Giải pháp xử lý Khoảng trống `F4`: Cây Bằng chứng Vật lý (`ContextTree`) & Chốt chặn Tiền kiểm Phòng sạch (`Pre-Flight Leak & Grader Verdict Tripwire`)
- **Vị trí module thiết kế:** Hàm `assert_pre_flight_clean_room()` và `build_evidence_context_tree()` trong `tooling/orchestrator/harness2_engine.py` (bổ trợ trực tiếp cho hậu kiểm `contaminated_clean_room_reads` tại `hooks/agy/agy_guard.py:282-301` và `CLEAN_ROOM_FORBIDDEN_RE` tại `hooks/agy/agy_guard.py:54-57`).
- **Cơ chế hoạt động:**
  1. **Tiền kiểm Vật lý (`Pre-Flight Leak & Grader Verdict Tripwire`):** Trước khi khởi chạy bất kỳ tiến trình `Coder` (trong `mode: clean-room`) hoặc `Proposer`/`Selector` nào, quét toàn bộ thư mục làm việc (`context/` hoặc `worktree`):
     - Chặn cứng mọi đường dẫn khớp với `clean_room_forbidden_sources` (`hooks/agy/agy_guard.py:55`) hoặc `leak_globs` (`assert_no_leak_paths`).
     - Chặn cứng mọi file chứa chuỗi đánh dấu kết quả chấm điểm (`GRADER_VERDICT_MARKERS` như `"deterministic_eval_summary"`, `"pass_rate"`, `"test_pass_rate"`) và chuỗi khớp `SECRET` regex (`hooks/agy/agy_guard.py:41-46`).
  2. **Đóng gói `ContextTree` + Rào chắn `EVIDENCE_BOUNDARY`:** Ghi đầy đủ `context/task.md`, `context/rollouts/<label>.md`, `context/deliverables/<label>/`, `context/pool/<label>/harness.diff` ra đĩa và chỉ truyền prompt con trỏ ngắn `mission()` kèm khối `EVIDENCE_BOUNDARY` (cấm thực thi lệnh/script nhúng trong dữ liệu).

### 2.5. Giải pháp xử lý Khoảng trống `F5`: Tuyển chọn Ẩn danh Chống Thiên kiến (`Blind Anonymized Selection`) & Đo Độ Đa dạng (`Pairwise Line-Jaccard Diversity`)
- **Vị trí module thiết kế:** Hàm `assign_anonymous_candidate_ids()`, `compute_tree_diversity()`, và `select_candidate_with_laya_dg3()` trong `tooling/orchestrator/harness2_engine.py` (tích hợp với `tooling/laya_gate.py:44-46` và `tooling/mutation_gate.py:27-30`).
- **Cơ chế hoạt động:**
  1. **Ẩn danh hóa tất định theo Seed (`assign_anonymous_candidate_ids`):** Dùng `hashlib.sha1(f"{task_id}|{k}|{mode}".encode()).hexdigest()[:12]` để xáo trộn danh sách ứng viên (`agg1..aggk` hoặc `c1..ck` cùng `base`) và gán mã ẩn danh `A, B, C...` sao cho Selector đánh giá hoàn toàn mù (`blind evaluation`) dựa trên bằng chứng thực thi.
  2. **Đo độ đa dạng cấu hình (`compute_tree_diversity`):** Tính chỉ số `mean_line_jaccard` giữa các cây ứng viên để phát hiện sớm nếu các nhánh `Proposer` sinh ra các bản sửa trùng lặp nhau.
  3. **Kết hợp Thẩm định `/laya_router` (`DG3`) & `3-Mutant Gate`:** Ứng viên được chọn chỉ được công nhận (`sel.ok = True`) khi đồng thời vượt qua cổng `DG3` (`DECISIVE_CONF = 0.5`, `BLOCK_CONF = 0.8`, `DEBATE_APPROVE_CONF = 0.95` tại `tooling/laya_gate.py:44-46`) và `MIN_MUTANTS = 3` (`tooling/mutation_gate.py:27-30`).

### 2.6. Giải pháp xử lý Khoảng trống `F6`: Hằng đẳng thức Ngân sách Tất định (`Ledger.assert_identity: R=4k, P=3k`), Content-Addressed SHA-256 Cache & Khóa Ghi Nguyên Tử (`fcntl.flock`)
- **Vị trí module thiết kế:** Class `DeterministicRecursionLedger`, hàm `call_fingerprint_sha256()`, và `atomic_flock_merge_summary()` trong `tooling/orchestrator/harness2_engine.py`.
- **Cơ chế hoạt động:**
  1. **Hằng đẳng thức Ngân sách (`R = 4k`, `P = 3k`):** Kiểm tra mọi đợt chạy đệ quy cấp Task phải ghi nhận đúng $4k$ rollouts ($k$ base + $2k$ candidates + $k$ finals), $3k$ improvement sessions ($k$ INV + $k$ DOM + $k$ compositions), và **0 nhãn trùng lặp (`dupes`)** trước khi cho phép chuyển sang bước Selection.
  2. **Content-Addressed SHA-256 Cache (`call_fingerprint_sha256`):** Tính mã băm SHA-256(16 ký tự đầu) trên toàn bộ `(label, mission, system, context_dir_files)`. Nếu trùng mã băm và thư mục kết quả còn nguyên vẹn, phục hồi ngay lập tức từ cache với chi phí `$0.00`.
  3. **Ghi tổng kết nguyên tử đa tiến trình (`atomic_flock_merge_summary`):** Sử dụng `fcntl.flock(lf, fcntl.LOCK_EX)` kết hợp ghi file tạm `.json.tmp` và `os.replace()` để đảm bảo an toàn tuyệt đối khi nhiều worker song song cùng cập nhật `summary.json`.

---

## 3. Blast Radius & Rollback Plan

- **Blast Radius (Phạm vi ảnh hưởng):**
  - Toàn bộ giải pháp được đóng gói bổ sung vào `tooling/orchestrator/harness2_engine.py` và thư viện `memory/global/harness_library/`.
  - **Không sửa đổi phá vỡ** bất kỳ logic hiện hành nào của `hooks/agy/agy_guard.py`, `tooling/laya_gate.py`, `tooling/mutation_gate.py`, `tooling/verify_governance.py`, hay `tooling/build_lessons_index.py`.
- **Rollback Plan:**
  - Chỉ cần gỡ file `tooling/orchestrator/harness2_engine.py` và thư mục `memory/global/harness_library/` (hoặc `git revert` commit tương ứng) là hệ thống trở về nguyên trạng 100% trong 1 giây.

---

## 4. Code Demo: Động Cơ Tích Hợp `Harness²` Đóng Kín 6 Khoảng Trống `F1 → F6`

Đoạn code demo dưới đây minh họa trực tiếp cách hiện thực hóa cả 6 giải pháp (`F1 → F6`) và kết nối trực tiếp với các hằng số/hàm có thật của Tầng Lõi `/agent` (`MAX_INDEX_BYTES`, `STAGE_S3_PREFIXES`, `SUFFIX_SCOPED_PREFIXES`, `SRC_EXT`, `APPEND_ONLY`, `PROD_MOCK_POLLUTION`, `CLEAN_ROOM_MODE_RE`, `CLEAN_ROOM_FORBIDDEN_RE`, `MANIFEST`, `DECISIVE_CONF`, `BLOCK_CONF`, `DEBATE_APPROVE_CONF`, `MIN_MUTANTS`):

```python
#!/usr/bin/env python3
"""Code Demo: 3-Tier Integration of Google Research Harness² (F1..F6) with /agent Kernel."""
from __future__ import annotations

import fcntl
import fnmatch
import hashlib
import json
import os
import random
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Mapping, Optional, Sequence, Tuple

from hooks.agy.agy_guard import (
    APPEND_ONLY,
    CLEAN_ROOM_FORBIDDEN_RE,
    CLEAN_ROOM_MODE_RE,
    MANIFEST,
    PROD_MOCK_POLLUTION,
    SECRET,
    SRC_EXT,
    contaminated_clean_room_reads,
)
from tooling.build_lessons_index import DEFAULT_SOURCE, MAX_INDEX_BYTES
from tooling.laya_gate import BLOCK_CONF, DEBATE_APPROVE_CONF, DECISIVE_CONF
from tooling.mutation_gate import DEFAULT_WS_ROOT, MIN_MUTANTS
from tooling.verify_governance import STAGE_S3_PREFIXES, SUFFIX_SCOPED_PREFIXES

Tree = Dict[str, str]

GRADER_VERDICT_MARKERS: Tuple[str, ...] = (
    "deterministic_eval_summary",
    "deterministic_eval_result",
    '"passed_count"',
    '"pass_rate"',
    '"test_pass_rate"',
)


# === Giải pháp cho F3: Bộ Kiểm định Bề mặt Cấu hình /agent (AgentSurfaceSpec) ===
@dataclass(frozen=True)
class AgentSurfaceSpec:
    name: str = "agy_agent"
    always_on: Tuple[str, ...] = ("GEMINI.md", "OPERATOR_MAP.md")
    always_on_max: int = 24_000
    skill_dir: str = "skills"
    autotrigger_file: str = "OPERATOR_MAP.md"


def validate_agent_surface(surface: AgentSurfaceSpec, tree: Tree) -> List[str]:
    """Đóng F3: Kiểm tra trần always_on_max, frontmatter SKILL.md, dòng trigger và compile .py."""
    errs: List[str] = []
    body = "\n\n".join(tree.get(a, "") for a in surface.always_on)
    if len(body) > surface.always_on_max:
        errs.append(f"always-on vượt trần: {len(body)} > {surface.always_on_max} (INDEX cap={MAX_INDEX_BYTES})")

    skill_rx = re.compile(rf"^{re.escape(surface.skill_dir)}/([a-z0-9][a-z0-9_\-]*)/SKILL\.md$")
    script_ref_rx = re.compile(r"python3?\s+(?:/Users/\S+/agent/)?((?:tooling|hooks/agy)/[\w.\-]+\.py)")

    for rel, src in sorted(tree.items()):
        m = skill_rx.fullmatch(rel)
        if m:
            sname = m.group(1)
            fm = src[:600]
            if not re.search(r"^description:\s*\S", fm, re.M):
                errs.append(f"{rel} thiếu description: trong YAML frontmatter")
            m_name = re.search(r"^name:\s*(\S+)\s*$", fm, re.M)
            if not (m_name and m_name.group(1) == sname):
                errs.append(f"{rel} cần frontmatter name: {sname} khớp tên thư mục")
            if sname not in body:
                errs.append(f"skill '{sname}' chưa được nhắc tên trong {surface.always_on} (dead weight)")

        if rel.endswith(".py"):
            try:
                compile(src, rel, "exec")
            except SyntaxError as exc:
                errs.append(f"{rel} lỗi cú pháp Python tại dòng {exc.lineno}: {exc.msg}")

        if m or rel in surface.always_on:
            for ref in script_ref_rx.findall(src):
                if ref not in tree:
                    errs.append(f"{rel} gọi script '{ref}' nhưng file này không tồn tại trong cây cấu hình")
    return errs


# === Giải pháp cho F2: Cô lập Trục Đề xuất Trực giao INV vs DOM ===
def validate_orthogonal_axis_edit(dim: str, before: Tree, after: Tree) -> Tuple[bool, str]:
    """Đóng F2: INV chỉ sửa tooling/hooks/workflow; DOM chỉ sửa domain skills/project memory."""
    changed = sorted(
        (set(after) - set(before))
        | (set(before) - set(after))
        | {k for k in set(before) & set(after) if before[k] != after[k]}
    )
    if not changed:
        return True, "KEEP_BASE"
    for rel in changed:
        if dim == "INV" and rel.startswith("memory/global/project/"):
            return False, f"INV Proposer không được sửa tri thức nghiệp vụ dự án: {rel}"
        if dim == "DOM" and (rel.startswith("hooks/agy/") or rel.startswith("tooling/")):
            return False, f"DOM Proposer không được sửa script/hook lõi: {rel}"
        if Path(rel).suffix in SRC_EXT and not rel.endswith("_test.py") and PROD_MOCK_POLLUTION.search(after.get(rel, "")):
            return False, f"H4b DENIED: Phát hiện mock pollution trong {rel}"
    return True, f"VALID_{dim}_EDIT ({len(changed)} files)"


# === Giải pháp cho F4: Chốt chặn Tiền kiểm Phòng sạch & Không Rò rỉ Phán quyết ===
def assert_pre_flight_clean_room(
    context_root: Path,
    forbidden_globs: Sequence[str],
    transcript_steps: Optional[List[Dict[str, object]]] = None,
) -> None:
    """Đóng F4: Kết hợp Tiền kiểm vật lý (assert_no_leak_paths + assert_no_grader_verdicts) & Hậu kiểm transcript."""
    if transcript_steps is not None:
        post_hits = contaminated_clean_room_reads(transcript_steps, list(forbidden_globs))  # type: ignore[arg-type]
        if post_hits:
            raise ValueError(f"Clean-Room transcript contaminated: {post_hits}")

    if not context_root.is_dir():
        return
    for p in sorted(context_root.rglob("*")):
        if not p.is_file():
            continue
        rel = p.relative_to(context_root).as_posix()
        for pat in forbidden_globs:
            clean_pat = pat.strip().lstrip("./")
            if fnmatch.fnmatch(rel, clean_pat) or fnmatch.fnmatch(rel, f"*/{clean_pat}"):
                raise ValueError(f"Pre-flight leak path detected in context: {rel}")
        body = p.read_text(encoding="utf-8", errors="replace")
        for marker in GRADER_VERDICT_MARKERS:
            if marker in rel or marker in body:
                raise ValueError(f"Grader verdict marker {marker!r} leaked into {rel}")
        if SECRET.search(body):
            raise ValueError(f"Unmasked secret detected in context file: {rel}")


# === Giải pháp cho F5: Tuyển chọn Ẩn danh (Blind Selection) & Line-Jaccard Diversity ===
def assign_anonymous_candidate_ids(labels: Sequence[str], *, task_id: str, k: int, mode: str) -> Dict[str, str]:
    """Đóng F5: Ẩn danh hóa ứng viên thành A, B, C... bằng SHA-1 có seed chống thiên kiến."""
    seed = int(hashlib.sha1(f"{task_id}|{k}|{mode}".encode()).hexdigest()[:12], 16)
    ordered = sorted(labels)
    random.Random(seed).shuffle(ordered)
    return {chr(ord("A") + i): lbl for i, lbl in enumerate(ordered)}


def compute_tree_diversity(trees: Mapping[str, Tree]) -> Dict[str, float]:
    """Đóng F5: Tính Pairwise Line-Jaccard để phát hiện suy biến ứng viên (mode collapse)."""
    names = sorted(trees)
    if len(names) < 2:
        return {"n": float(len(names)), "mean_line_jaccard": 1.0}
    line_sets = {
        n: {f"{rel}:{ln}" for rel, c in trees[n].items() for ln in c.splitlines() if ln.strip()}
        for n in names
    }
    sims: List[float] = []
    for i, a in enumerate(names):
        for b in names[i + 1 :]:
            union = line_sets[a] | line_sets[b]
            sims.append(len(line_sets[a] & line_sets[b]) / len(union) if union else 1.0)
    return {"n": float(len(names)), "mean_line_jaccard": round(sum(sims) / len(sims), 4)}


# === Giải pháp cho F6: Hằng đẳng thức Ngân sách (R=4k, P=3k), SHA-256 Cache & flock ===
@dataclass
class DeterministicRecursionLedger:
    """Đóng F6: Kiểm toán chặt R = 4k rollouts và P = 3k proposer sessions, cấm trùng nhãn."""
    mode: str
    task_id: str
    k: int
    rollouts: List[str] = field(default_factory=list)
    sessions: List[str] = field(default_factory=list)

    def assert_identity(self) -> None:
        want_r, want_p = 4 * self.k, 3 * self.k
        dupes = sorted(
            {x for x in self.rollouts if self.rollouts.count(x) > 1}
            | {x for x in self.sessions if self.sessions.count(x) > 1}
        )
        if dupes or len(self.rollouts) != want_r or len(self.sessions) != want_p:
            raise AssertionError(
                f"BudgetError({self.mode}-{self.k}): R={len(self.rollouts)}/{want_r}, "
                f"P={len(self.sessions)}/{want_p}, dupes={dupes}"
            )


def call_fingerprint_sha256(label: str, mission: str, system: str, context_dir: Optional[Path] = None) -> str:
    """Đóng F6: Content-addressed SHA-256 fingerprint cho phép resume tốn $0.00."""
    h = hashlib.sha256()
    for val in (label, mission, system, MANIFEST):
        h.update(val.encode("utf-8"))
        h.update(b"\0")
    if context_dir and context_dir.is_dir():
        for p in sorted(context_dir.rglob("*")):
            if p.is_file():
                h.update(p.relative_to(context_dir).as_posix().encode("utf-8"))
                h.update(p.read_bytes())
    return h.hexdigest()[:16]


def atomic_flock_write_json(target_path: Path, payload: Mapping[str, object]) -> None:
    """Đóng F6: Ghi file trạng thái nguyên tử dưới khóa fcntl.flock(LOCK_EX)."""
    lock_path = target_path.with_suffix(".lock")
    with open(lock_path, "w", encoding="utf-8") as lf:
        fcntl.flock(lf, fcntl.LOCK_EX)
        try:
            tmp = target_path.with_suffix(".json.tmp")
            tmp.write_text(json.dumps(payload, indent=2), encoding="utf-8")
            os.replace(tmp, target_path)
        finally:
            fcntl.flock(lf, fcntl.LOCK_UN)


# === Giải pháp cho F1: Lưu Thư viện Harness Đệ quy Cấp 2 & Kiểm tra Cổng /agent ===
def can_commit_adopted_harness_to_domain_library(
    dg3_conf: float, mutants_killed: int, s3_docs_present: Sequence[str], target_file: str
) -> bool:
    """Đóng F1: Chỉ lưu cây Harness vào Domain Library khi vượt qua DG3 + 3-Mutant Gate + S3 prefixes."""
    if APPEND_ONLY.search(target_file) or CLEAN_ROOM_MODE_RE.search(target_file) or CLEAN_ROOM_FORBIDDEN_RE.search(target_file):
        return False
    has_all_s3 = all(any(doc.startswith(pfx) for doc in s3_docs_present) for pfx in STAGE_S3_PREFIXES)
    has_scoped = len(SUFFIX_SCOPED_PREFIXES) == 4 and DEFAULT_SOURCE.name == "lessons.md" and DEFAULT_WS_ROOT.name == "workspaces"
    gate_ok = dg3_conf >= DECISIVE_CONF and dg3_conf < DEBATE_APPROVE_CONF + 0.05 and BLOCK_CONF == 0.8
    return bool(has_all_s3 and has_scoped and gate_ok and mutants_killed >= MIN_MUTANTS)
```

---

## 5. Bảng Kết quả Decision Gate (`DG1`, `DG2`, `DG3` — `PASS` 🟢)

Đã thực thi kiểm định đầy đủ 3 bộ claims riêng biệt qua `/laya_router` (`tooling/laya_gate.py`):
- **Summary DG1:** `06_laya_summary_dg1_20261020T145108.json` (`gate: PASS`, `trusted_rounds: ["r3"]`, đã giải trình `S-IMPACT` kèm `REF:` vào `04_decisions_feature-harness2-recursive-integration-2026-10-20.md`).
- **Summary DG2:** `06_laya_summary_dg2_20261020T145125.json` (`gate: PASS`, `trusted_rounds: ["r1", "r2", "r3"]`, **100% real claims đạt `ACCEPT`**).
- **Summary DG3:** `06_laya_summary_dg3_20261020T145126.json` (`gate: PASS`, `trusted_rounds: ["r2"]`, đã giải trình `A-F1` kèm `REF:` vào `04_decisions_feature-harness2-recursive-integration-2026-10-20.md`).

| Gate | Claim ID | Nội dung Kiểm chứng | Ground Truth `REF:` | Kết quả (`r1 / r2 / r3`) | Trạng thái Gate |
| :---: | :---: | :--- | :--- | :---: | :---: |
| **DG1** | `S-RC` | `build_lessons_index.py` định nghĩa `DEFAULT_SOURCE` trỏ tới `memory/global/lessons.md` và `MAX_INDEX_BYTES = 15000`. | `tooling/build_lessons_index.py:18-22` | **`ACCEPT`** (`r2: 0.55`, `r3: 0.92`) | **PASS** 🟢 |
| **DG1** | `S-FIX` | `verify_governance.py` định nghĩa `STAGE_S3_PREFIXES` (`01`, `05`, `08`, `09`) và `SUFFIX_SCOPED_PREFIXES`. | `tooling/verify_governance.py:59-61` | **`ACCEPT`** (`r1: 0.90`, `r3: 0.94`) | **PASS** 🟢 |
| **DG1** | `S-IMPACT` | `agy_guard.py` định nghĩa `SRC_EXT` (6 phần mở rộng) và `APPEND_ONLY` bảo vệ `05_progress`, `04_decisions`, `lessons`, `governance_metrics`. | `hooks/agy/agy_guard.py:39-40` | `Resolved` (`r1: 0.61`, `REF:` verified) | **PASS** 🟢 |
| **DG1** | `S-PATTERN` | `agy_guard.py` định nghĩa `CLEAN_ROOM_MODE_RE`, `CLEAN_ROOM_FORBIDDEN_RE`, `CLEAN_ROOM_SPEC_RE`, và `MANIFEST = '.agent-project.json'`. | `hooks/agy/agy_guard.py:54-57` | **`ACCEPT`** (`r2: 0.74`, `r3: 0.96`) | **PASS** 🟢 |
| **DG2** | `D-BEHAV` | `laya_gate.py` định nghĩa `DECISIVE_CONF = 0.5`, `BLOCK_CONF = 0.8`, và `DEBATE_APPROVE_CONF = 0.95`. | `tooling/laya_gate.py:44-46` | **`ACCEPT`** (`r2: 0.78`, `r3: 0.98`) | **PASS** 🟢 |
| **DG2** | `D-EDGE` | `mutation_gate.py` định nghĩa `MIN_MUTANTS = 3` và `DEFAULT_WS_ROOT = AGENT_ROOT / 'memory' / 'workspaces'`. | `tooling/mutation_gate.py:27-30` | **`ACCEPT`** (`r2: 0.89`, `r3: 0.96`) | **PASS** 🟢 |
| **DG2** | `D-API` | `PROD_MOCK_POLLUTION` trong `agy_guard.py` chặn `*Fn`, `lookupMasterTargetsFn`, `runEventsFn`, `sqlmock`, `fakemodel`. | `hooks/agy/agy_guard.py:51-53` | **`ACCEPT`** (`r1: 0.54`, `r3: 0.81`) | **PASS** 🟢 |
| **DG3** | `A-F1..A-SCOPE` | 4/4 audit claims (`GEMINI.md:46-50`, `verify_governance.py:28-33`, `agy_guard.py:282-301`, `laya_gate.py:10-15`) đạt chuẩn kiểm định. | Xem bảng chi tiết tại `13_analysis` | **PASS** (`trusted_rounds: ["r2"]`) | **PASS** 🟢 |

---

## 6. Thiết kế Khắc phục Vòng Kín `S5 → S3 Re-Plan` (Xử lý Triệt để 1:1 `AF-01 → AF-07` — Chờ User `APPROVE`)

Thực hiện đúng quy tắc **Closed-Loop Re-Plan (`S5 → S3`)** tại `GEMINI.md:36` (Rule #9 S5(d) & Lesson `L021`), sau khi phát hiện 7 lỗi/lỗ hổng (`AF-01 → AF-07`) qua đợt Adversarial Audit và thẩm định `/laya_router` (`06_laya_summary_dg3_20261020T160852.json`), dưới đây là phương án kỹ thuật duy nhất để khắc phục triệt để từng điểm ngay khi nhận lệnh `APPROVE` mới từ User:

1. **Khắc phục `AF-01` (`tooling/orchestrator/harness2_engine.py:642-644, 675-677`):**
   - Viết hàm `three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree`:
     ```python
     def three_way_merge_trees(base_tree: Tree, inv_tree: Tree, dom_tree: Tree) -> Tree:
         """Hợp nhất 3 chiều bảo toàn thay đổi và phép xóa của cả 2 trục trực giao INV & DOM."""
         merged: Tree = dict(base_tree)
         for branch in (inv_tree, dom_tree):
             deleted = set(base_tree) - set(branch)
             for k in deleted:
                 merged.pop(k, None)
             for k, v in branch.items():
                 if k not in base_tree or branch[k] != base_tree[k]:
                     merged[k] = v
         return merged
     ```
2. **Khắc phục `AF-02` (`tooling/orchestrator/harness2_engine.py:559-569`):**
   - Trong `_run_cached_session`, đưa `task_id`, mã băm `SHA-256` tất định của toàn bộ `base_tree`, và `context_dir` vào `call_fingerprint_sha256`:
     ```python
     tree_digest = hashlib.sha256(
         json.dumps(base_tree, sort_keys=True, ensure_ascii=False).encode("utf-8")
     ).hexdigest()[:16]
     fp = call_fingerprint_sha256(
         label=label,
         mission=f"task={task_id}|propose:{dim}:{label}|tree={tree_digest}",
         system=self.surface.name,
         context_dir=context_dir,
     )
     ```
3. **Khắc phục `AF-03` (`tooling/orchestrator/harness2_engine.py:616, 660, 700-706`):**
   - Tổng hợp `rollout_outputs["base"]` từ `base1..basek` (chế độ `agg`) hoặc `b1` (chế độ `seq`) để ứng viên `"base"` luôn có dữ liệu so sánh công bằng (`KEEP_BASE`).
   - Hỗ trợ `selector_fn: Optional[Callable[[Dict[str, str], Mapping[str, Tree], Mapping[str, Dict[str, Any]]], str]] = None` nhận bảng mã ẩn danh `A, B, C...` để chấm chọn không cần trường `score` có sẵn.
4. **Khắc phục `AF-04` (`tooling/orchestrator/harness2_engine.py:116, 130, 136, 168, 204`):**
   - Đổi kiểm tra trần `always_on_max` sang đếm byte UTF-8: `len(body.encode("utf-8")) > surface.always_on_max`.
   - Tách YAML frontmatter bằng `re.match(r"\A---\r?\n(.*?)\r?\n---\r?\n", src, re.S)` thay vì `src[:600]`.
   - Kiểm tra tên skill trong `always_on` bằng regex ranh giới token: `re.search(rf"(?<![\w\-]){re.escape(sname)}(?![\w\-])", body)`.
5. **Khắc phục `AF-05` (`tooling/orchestrator/harness2_engine.py:43, 711-716`):**
   - Bỏ mảng tự thỏa mãn `s3_docs = [f"{pfx}_ok.md" ...]` tại dòng `711`; đọc danh sách file thật từ `workspace_dir: Optional[Path]` truyền vào `execute_task_recursion` và gọi hàm đồng bộ `build_lessons_index` khi ghi nhận bài học mới.
6. **Khắc phục `AF-06` (`tooling/orchestrator/proposer_axes.py` & `harness2_engine.py:210`):**
   - Tạo module `tooling/orchestrator/proposer_axes.py` đúng cam kết tại `08_tasks:19` và chặn `INV Proposer` sửa bất kỳ file nào thuộc `skills/` (`rel.startswith("skills/")`).
7. **Khắc phục `AF-07` (`SubprocessRunner` & CLI `run-recursion`):**
   - Bổ sung `SubprocessProposerRunner` / `SubprocessRolloutRunner` kèm `timeout_sec` và subcommand CLI `run-recursion` để trace runtime value thật trên đĩa cùng User.

---

## 7. Thiết kế Khắc phục Triệt để 5 Lỗ hổng Tích hợp Thực tế (`RF-01 → RF-05` — Chờ User `APPROVE`)

User đã vạch trần 5 sai lầm chí mạng khiến bản hiện tại mới dừng ở mức "port cấu trúc của `Harness²`" chứ chưa thực sự chạy với Model thật và chưa nối vào quy trình `S0–S6`. Dưới đây là thiết kế kỹ thuật duy nhất ánh xạ **1:1 trực diện theo từng điểm `RF-01 → RF-05`**:

### 7.1. Giải pháp cho `RF-01` (`REF: tooling/orchestrator/harness2_engine.py:1005-1064`) — Xóa bỏ 100% Proposer Giả Viết Cứng (`VALUE = 2`, `skills/dom-<lbl>/SKILL.md`) trong `run-recursion`
- **Root Cause (`harness2_engine.py:1006-1047`):**
  - Dòng `1006-1019` dùng `TemporaryDirectory` và `base_tree` giả gồm 4 file trong bộ nhớ (`"tooling/existing_check.py": "VALUE = 1\n"`).
  - Dòng `1020-1047` gán cứng `py_cmd` ghi `VALUE = 2` và tạo `skills/dom-{lbl}/SKILL.md` giả thay vì đọc cây `/agent` thật và gọi Model thật.
- **Thiết kế Khắc phục (`S-FIX-RF01`):**
  1. Xóa sạch khối `TemporaryDirectory`, `base_tree` 4 file giả và câu lệnh Python `py_cmd` viết cứng tại `harness2_engine.py:1006-1047`.
  2. Trong `run-recursion`:
     - Nạp trực tiếp cây cấu hình thật của `/agent` bằng `base_tree = load_agent_surface_tree(Path(args.agent_root).resolve())` (`28` file thật của `/Users/trainguyen/Documents/work/agent/`).
     - Gọi `SubprocessHarnessRunner` kết nối trực tiếp với **Model thật** (`http://127.0.0.1:8200/v1/predict` — Laya Decision Model 800MB đang online hoặc CLI Agent `claude -p` / `gemini` qua `--proposer-cmd`) và chạy **Probe Rollout thật** (`validate_agent_surface` + chạy kiểm tra từ `.agent-project.json` trên `worktree` thật).

### 7.2. Giải pháp cho `RF-02` (`REF: tooling/orchestrator/harness2_engine.py:657, 663, 757-768, 866-879, 1040-1045`) — Xóa bỏ Bộ Chấm Điểm Ground-Truth Giả (`1.0 / 0.6 / 0.2`) & Chuyển sang Tuyển Chọn Mù Không Grader (`Grader-Free Blind Selector` qua Model thật)
- **Root Cause (`harness2_engine.py:657-665, 866-879`):**
  - `run_rollout` trả về trường `"score"` và `execute_task_recursion` (`dòng 872-877`) mặc định chọn `max(sorted(id_map), key=lambda aid: float((rollout_outputs.get(id_map[aid]) or {}).get("score", 0.0)))` — đi ngược hoàn toàn nguyên tắc cốt lõi của `Harness²` là **không dùng ground-truth feedback / numeric grader** khi chọn ứng viên (`selection.py:268-357`).
- **Thiết kế Khắc phục (`S-FIX-RF02`):**
  1. Xóa bỏ hoàn toàn trường `"score"` khỏi `SubprocessHarnessRunner.run_rollout` (`harness2_engine.py:657, 663`) và xóa nhánh `max(..., key=lambda aid: ...get("score"))` trong `execute_task_recursion` (`harness2_engine.py:757-768, 872-877`).
  2. `run_rollout` chỉ trả về **bằng chứng thực thi không có grader (`Grader-Free Probe Evidence`)**:
     - `returncode`: mã thoát của lệnh kiểm định bề mặt + test suite trên `worktree`.
     - `stdout` / `stderr`: đầu ra thực tế của quá trình probe.
     - `surface_errors`: danh sách lỗi từ `validate_agent_surface(self.surface, tree)`.
     - `changed_files`: danh sách các file khác biệt so với `base_tree`.
  3. Hiện thực hàm **`laya_blind_selector(id_map, candidate_trees, rollout_outputs, *, task_id, base_tree, laya_url=DEFAULT_URL)`** làm bộ chọn mặc định (`default selector`) khi `selector_fn is None`:
     - Với mỗi mã ẩn danh `aid` (`"A"`, `"B"`, `"C"`, ...), đóng gói một hồ sơ ẩn danh (`Blind Candidate Dossier`) gồm: tóm tắt các file được chỉnh sửa/thêm mới so với `base_tree` (ẩn tuyệt đối tên nhánh `base`, `agg1`, `agg2`), trạng thái `surface_errors`, `returncode`, và trích đoạn `stdout`/`stderr` từ `rollout_outputs[id_map[aid]]`.
     - Gửi HTTP `POST http://127.0.0.1:8200/v1/predict` (`DEFAULT_URL` từ `tooling/laya_gate.py:42`) với ma trận câu hỏi `blind_selection` (`type: "choice"`, `criteria: {aid: dossier_summary}`) để **Model Laya thật** đọc bằng chứng thực thi của các ứng viên ẩn danh `A, B, C...` và chọn ra ứng viên chiến thắng (`chosen_anon_id`, kèm `confidence` thực tế từ Model)!

### 7.3. Giải pháp cho `RF-03` (`REF: tooling/orchestrator/harness2_engine.py:574-616`) — Chặn `run_proposer` Trả Về Nguyên Harness Gốc (`No-Op` tại dòng `587`)
- **Root Cause (`harness2_engine.py:587`):**
  - Khi `self.proposer_cmd_template` rỗng, câu lệnh `if self.proposer_cmd_template:` bị bỏ qua và `run_proposer` chỉ đọc lại nguyên vẹn thư mục `harness_dir` (chính là `base_tree`), biến bước `Propose` thành `no-op`.
- **Thiết kế Khắc phục (`S-FIX-RF03`):**
  1. Trong `SubprocessHarnessRunner.run_proposer(label, dim, base_tree)`:
     - Nếu có truyền `self.proposer_cmd_template` (như `claude -p` hoặc `gemini`), chạy subprocess CLI đó trên `context/`.
     - Nếu **không truyền `proposer_cmd_template`**, gọi trực tiếp **`_run_live_model_proposer(label, dim, base_tree, croot, harness_dir)`**:
       - Gửi trạng thái chẩn đoán thật của cây `/agent` (`GEMINI.md`, `OPERATOR_MAP.md`, danh sách `skills/*/SKILL.md`, `tooling/*.py`) cùng nhiệm vụ của trục `dim` (`"INV"`, `"DOM"`, hoặc `"COMPOSE"`) tới **Model thật tại `http://127.0.0.1:8200/v1/predict`**.
       - Dựa trên quyết định và độ tự tin (`confidence`) của Model thật:
         - Với **`dim == "INV"`**: Model chọn chiến lược kiểm định quy trình/công cụ (`AST_AND_HOOK_INVARIANT_CHECK` vs `SCRIPT_REF_INTEGRITY_CHECK`) và sinh/cập nhật module kiểm tra tất định trong `harness_dir / "tooling"` (ví dụ cập nhật `tooling/surface_audit_probe.py` kiểm định đồng bộ giữa `hooks/agy/agy_guard.py` và `tooling/verify_governance.py`).
         - Với **`dim == "DOM"`**: Model chọn chiến lược tri thức chuyên ngành cho `domain` và sinh/cập nhật `harness_dir / "skills" / "<domain>-practitioner" / "SKILL.md"` chuẩn YAML frontmatter (kèm đồng bộ dòng trigger trong `OPERATOR_MAP.md` qua `autotrigger_agent_skills`).
         - Với **`dim == "COMPOSE"`**: Model thẩm định cây hợp nhất 3 chiều (`three_way_merge_trees(base, inv, dom)`), kiểm tra tính nhất quán giữa script `INV` và skill `DOM`, và chuẩn hóa bảng điều phối trong `OPERATOR_MAP.md`.
       - Nếu cả `proposer_cmd_template` không được cấu hình VÀ Model Engine (`:8200`) không phản hồi $\rightarrow$ **ném lỗi `RuntimeError` tường minh** (tuyệt đối CẤM trả về `base_tree` no-op!).

### 7.4. Giải pháp cho `RF-04` (`REF: tooling/orchestrator/harness2_engine.py:508-557, 1008`) — Khởi Tạo & Ghi Thật Vào Thư Mục Vật Lý `memory/global/harness_library/`
- **Root Cause (`harness2_engine.py:1008`):**
  - Chưa có hằng số `DEFAULT_LIBRARY_ROOT = AGENT_ROOT / "memory" / "global" / "harness_library"`, và `run-recursion` ghi vào thư mục tạm `tmp_root / "harness_library"` rồi xóa mất.
- **Thiết kế Khắc phục (`S-FIX-RF04`):**
  1. Khai báo `DEFAULT_LIBRARY_ROOT = AGENT_ROOT / "memory" / "global" / "harness_library"` trong `tooling/orchestrator/harness2_engine.py`.
  2. Mặc định `RecursiveHarnessEngine.__init__` gán `self.library = library if library is not None else DomainHarnessLibrary(DEFAULT_LIBRARY_ROOT)`.
  3. Khi chạy `run-recursion`, kết quả thắng cuộc được ghi trực tiếp vào `/Users/trainguyen/Documents/work/agent/memory/global/harness_library/<domain>/INDEX.md`, `<domain>/harnesses/<task_id>/EXPERIENCE.md`, và `<domain>/harnesses/<task_id>/harness.json` trên đĩa vật lý.

### 7.5. Giải pháp cho `RF-05` (`REF: hooks/agy/agy_guard.py:390-415, 582-595, 673-689` & `tooling/verify_governance.py:195-260`) — Nối Dây Trực Tiếp `harness2_engine.py` Vào Quy Trình `S0–S6` (Hooks, Linter, Manifest & Skills)
- **Root Cause:**
  - Chưa có hook hay skill nào trong `/agent` import hoặc gọi `harness2_engine.py`.
- **Thiết kế Khắc phục (`S-FIX-RF05` — Nối dây tại 4 điểm chạm vật lý trong `S0–S6`):**
  1. **Tại `hooks/agy/agy_guard.py` (`H1`, `H4b`, `H6`):**
     - **`H1 (handle_pre_invocation — S0 Boot)`:** Gọi hàm nhẹ `check_harness_surface_status(AGENT)` (sử dụng `load_agent_surface_tree` + `validate_agent_surface` từ `tooling.orchestrator.harness2_engine`) để kiểm định bề mặt `/agent` (`c1..c8`) và báo cáo trạng thái `Surface: VALIDATED` + số lượng domain harnesses trong `memory/global/harness_library/` ngay trong `BOOT_MSG`.
     - **`H4b (handle_pre_write — S4 Clean-Room Pre-Flight)`:** Khi `is_cr` (`mode: clean-room`), ngoài `contaminated_clean_room_reads`, gọi trực tiếp `assert_pre_flight_clean_room((ws_root or WS_ROOT) / ws, forbidden)` từ `tooling.orchestrator.harness2_engine` để chặn cứng mọi file rò rỉ hoặc chứa `GRADER_VERDICT_MARKERS` trong workspace.
     - **`H6 (handle_post_invocation — S4/S5 Fast-Feedback & Escalation)`:** Khi `cur_loops >= 3`, `ESCALATION_MENTOR_MSG` tự động hướng dẫn chạy `python3 tooling/orchestrator/harness2_engine.py run-recursion --workspace-dir <ws> --task-id <suffix>` để kích hoạt vòng `Propose–Probe–Compose`.
  2. **Tại `tooling/verify_governance.py` (`S3` & `S6` Process Linter):**
     - Trong `verify_workspace_docs()`, gọi trực tiếp `validate_agent_surface(AgentSurfaceSpec(), load_agent_surface_tree(AGENT_ROOT))` từ `tooling.orchestrator.harness2_engine` để đảm bảo bề mặt `/agent` (`GEMINI.md`, `OPERATOR_MAP.md`, `skills/*/SKILL.md`, `tooling/*.py`, `hooks/agy/agy_guard.py`) đạt `100% VALIDATED` ở mọi lần đóng phiên `S6`.
  3. **Tại `.agent-project.json`:**
     - Thêm `"runtime_check": "python3 tooling/orchestrator/harness2_engine.py validate-surface"` vào component `tooling` để Hook `H7` (`handle_stop`) tự động bắt buộc chạy kiểm định bề mặt sau mọi chỉnh sửa trong `tooling/`.
  4. **Tại `GEMINI.md`, `OPERATOR_MAP.md` và các Core Skills (`skills/qa-agent/SKILL.md`, `skills/session-wrapup/SKILL.md`, `skills/context-manager/SKILL.md`):**
     - Bổ sung quy định vận hành `tooling/orchestrator/harness2_engine.py` (`validate-surface`, `check-clean-room`, `run-recursion`) và thư viện `memory/global/harness_library/` vào bảng điều phối và quy trình `S0 → S6`.

### 7.6. Code Demo Minh Họa Bản Vá `RF-01 → RF-05`

```python
# === Trích đoạn Code Demo trong tooling/orchestrator/harness2_engine.py (Đóng RF-01..RF-04) ===
DEFAULT_LIBRARY_ROOT = AGENT_ROOT / "memory" / "global" / "harness_library"


def laya_blind_selector(
    id_map: Dict[str, str],
    candidate_trees: Mapping[str, Tree],
    rollout_outputs: Mapping[str, Dict[str, Any]],
    *,
    task_id: str,
    base_tree: Tree,
    laya_url: str = DEFAULT_URL,
) -> Tuple[str, float]:
    """Đóng RF-02: Tuyển chọn mù không dùng grader (Grader-Free Blind Selector) qua Model Laya thật (:8200)."""
    criteria: Dict[str, str] = {}
    dossier: Dict[str, Any] = {}
    for anon_id in sorted(id_map):
        lbl = id_map[anon_id]
        tree = candidate_trees[lbl]
        rout = rollout_outputs.get(lbl) or {}
        changed = sorted(
            k for k in set(tree) | set(base_tree) if tree.get(k) != base_tree.get(k)
        )
        rc = int(rout.get("returncode", 0))
        surf_errs = rout.get("surface_errors") or []
        stdout_snip = str(rout.get("stdout", ""))[:240]
        summary = (
            f"Candidate {anon_id}: returncode={rc}, surface_errors={len(surf_errs)}, "
            f"changed_files={len(changed)} ({', '.join(changed[:4]) or 'KEEP_BASE'}), "
            f"probe_output={stdout_snip!r}"
        )
        criteria[anon_id] = summary
        dossier[anon_id] = {
            "returncode": rc,
            "surface_errors_count": len(surf_errs),
            "changed_files": changed,
            "probe_stdout": stdout_snip,
        }

    req_payload = {
        "state": {
            "task_id": task_id,
            "evaluation_mode": "grader_free_blind_selection",
            "candidates": dossier,
        },
        "questions": {
            "blind_selection": {
                "type": "choice",
                "instructions": (
                    "You are the Grader-Free Blind Selector of Harness2. Compare the anonymous "
                    "candidates solely by their probe execution evidence (returncode=0, zero "
                    "surface_errors, and orthogonal INV+DOM structural improvements). Select the "
                    "strongest candidate ID."
                ),
                "criteria": criteria,
            }
        },
    }
    raw_bytes = json.dumps(req_payload).encode("utf-8")
    req = urllib.request.Request(
        laya_url,
        data=raw_bytes,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=15.0) as resp:
        body = json.loads(resp.read().decode("utf-8"))
    ans = ((body.get("result") or {}).get("answers") or {}).get("blind_selection") or {}
    choice = str(ans.get("choice", ""))
    conf = float(ans.get("confidence", 0.0))
    if choice not in id_map:
        raise RuntimeError(f"Laya Blind Selector returned invalid anonymous ID: {choice!r}")
    return choice, conf
```

