# 09 — Tasks Solution: Khắc phục Triệt để Lỗ hổng Bỏ qua Workflow & Hook Enforcement

- **Workspace:** `agent-process-optimization`
- **Tier:** `T3`
- **Ngày lập:** `2026-10-16`
- **Trạng thái:** `PROPOSE-ONLY (Chờ User APPROVE)`
- **Phiên điều tra đối chiếu:** Conversation `9d55dea5-f02d-4a52-8c5e-5169f1587e0b` (`transcript.jsonl:1-385`)

---

## 1. Bối cảnh & Triệu chứng Thực tế

User phản ánh sau khi chạy một task thực tế trên dự án `data-hub` (conversation `9d55dea5-f02d-4a52-8c5e-5169f1587e0b`):
> *"tao vừa cho 1 task của tao chạy theo agent này, nhưng chạy ko theo workflow, ko có hook, chỉ đơn thuần là báo đã đọc cái agent thôi. là sao ?"*

### Dòng thời gian thực tế trong `9d55dea5/transcript.jsonl` (`VERIFIED`):
1. **`11:25:24` (Step 1–101):** Phiên `9d55dea5` khởi chạy lúc `11:25:24`, **trước khi** `hooks/agy/agy_guard.py` (`11:51:42`) và `~/.gemini/config/hooks.json` (`14:54`) được triển khai xong.
2. **`11:58:10` (Step 102):** Khi `hooks.json` có hiệu lực ở `/Users/trainguyen/Documents/work/agent/.agents/hooks.json`, Hook `H1 (PreInvocation)` đã bắn message `BOOT S0` vào step 102 (`transcript.jsonl:102`) và tạo cờ `/tmp/agy/9d55dea5-f02d-4a52-8c5e-5169f1587e0b.boot`.
3. **`14:37:01` (Step 107–215):** User giao task mới: *"check dùm việc cấu hình luồng đối soát hằng ngày ở trên đang có vấn đề..."*.
   - Agent ở Step 109–110 đọc `GEMINI.md` và `memory/global/lessons/INDEX.md`, in ra câu `"Đã đọc GEMINI.md và lessons.md"`.
   - Nhưng sau đó Agent **không tạo/cập nhật `01_requirements_*`, `08_tasks_*`, không append `05_progress_*`, không lập `09_tasks_solution_*`, không chạy `laya_gate.py`**, mà nhảy thẳng vào đọc code `data-hub` và trả lời trên khung chat (Step 215)!
   - Khi Agent kết thúc lượt trả lời ở Step 215, **Hook `H7 (Stop)` hoàn toàn im lặng (không chặn)**!
4. **`14:57:15` (Step 222):** Chỉ đến khi User nhắc: *"mày ko thực hiện theo đúng các bước trong agent đã hướng dẫn ah"*, Agent mới vội vàng đọc `skills/brain-delegate/SKILL.md` (Step 226), tạo `09_tasks_solution_config_Hierarchy_v2.md`, chạy `laya_gate.py`, và chạy `verify_governance.py` (Step 346).

---

## 2. Phân tích 5 Nguyên nhân Gốc rễ (`VERIFIED` 100% kèm `file:line`)

### Root Cause 1 (`VERIFIED`): 9/10 Core Skills của `/agent` bị loại khỏi `<skills>` System Prompt do tràn Context Budget bởi 300+ skill ngoại lai trong `~/.gemini/config/skills/`
- **Bằng chứng cấu trúc Antigravity:** Theo `/Users/trainguyen/.gemini/antigravity/builtin/skills/agy-customizations/SKILL.md:58-75`, thứ tự ưu tiên nạp Skills của Antigravity là:
  1. `<workspace>/.agents/skills/` (Priority 1)
  2. `<workspace>/.agents/skills.json` (Priority 2)
  3. **`~/.gemini/config/skills/` (Priority 3 — Global Discovery)**
  4. `~/.gemini/antigravity/skills/` (Priority 4)
  5. **`~/.gemini/config/skills.json` (Priority 5 — Global Declared)**
- **Cơ chế gây lỗi:** Thư mục `~/.gemini/config/skills/` (Priority 3) đang chứa **hơn 300 skill bên thứ 3** (`avalonia-*`, `claude-*`, `django-*`, `laravel-*`, `perl-*`, `springboot-*`, `swift-*`, `unity-*`, `unreal-*`...). Toàn bộ 300+ skill này được nạp ở Priority 3 và **ngốn sạch Context Budget của khối `<skills>`** trước khi Antigravity kịp nạp đến `~/.gemini/config/skills.json` (Priority 5)!
- **Bằng chứng trực tiếp trong System Prompt:** Cuối khối `<skills>` của chính phiên hiện tại ghi rõ:
  `The following items were excluded due to context budget limits: ... audit-propose-only ... brain-delegate ... context-manager ... debug-agent ... laya-router ... muscle-execute ... qa-agent ... security-agent ... session-wrapup`
- **Hệ quả:** Chỉ duy nhất `workspace-docs` lọt vào `<skills>`; **9/10 Core Skills bị loại bỏ**. Trong khi đó, `/Users/trainguyen/Documents/work/agent/GEMINI.md:4` đã rút gọn chỉ còn trỏ sang `"Chi tiết từng quy trình nằm tại agent/skills/<skill>/SKILL.md"`. Vì Agent không nhìn thấy 9 skill kia trong `<skills>` và `GEMINI.md` Rule #9 (`GEMINI.md:66-75`) không ép lệnh đọc trực tiếp file `/Users/trainguyen/Documents/work/agent/skills/brain-delegate/SKILL.md`, Agent chỉ làm đúng câu đầu tiên trong `GEMINI.md` Rule #0 (`"Đã đọc GEMINI.md và lessons.md"`) rồi tự trả lời tự do!

### Root Cause 2 (`VERIFIED`): Bug nghiêm trọng trong `agy_guard.py` khi đọc `TargetFile` từ `transcript.jsonl` (`hooks/agy/agy_guard.py:88-101`)
- **Bằng chứng trong `transcript.jsonl`:** Trong `9d55dea5/transcript.jsonl:94,96`, Antigravity serialize các giá trị chuỗi bên trong `tool_calls[].args` kèm cặp dấu ngoặc kép bao ngoài:
  ```json
  "TargetFile": "\"/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/delivery/http/v1/reconciliation_handler.go\""
  ```
- **Bằng chứng trong code (`hooks/agy/agy_guard.py:88-101`):**
  ```python
  def tgt(args: Dict[str, Any]) -> str:
      return args.get("TargetFile") or ""

  def is_src(path_str: str, mem_root: Optional[Path] = None) -> bool:
      if not path_str or is_mem_path(path_str, mem_root):
          return False
      return Path(path_str).suffix in SRC_EXT
  ```
- **Cơ chế gây lỗi:** Khi đọc từ `transcript.jsonl` (trong `H4`, `H6`, `H7`), `tgt(a)` trả về chuỗi kết thúc bằng dấu `"` (`'.../reconciliation_handler.go"'`). Do đó `Path(path_str).suffix` trả về `'.go"'` (có dấu `"` ở cuối), **không bao giờ khớp** với `SRC_EXT = {".go", ".ts", ".tsx", ".js", ".py", ".sql"}`!
- **Hệ quả:** Hàm `src_edits(st)` (`hooks/agy/agy_guard.py:110-115`) **luôn trả về `[]` (rỗng) trên mọi transcript thật của Antigravity**, làm tê liệt hoàn toàn:
  - Toàn bộ `H6 (PostInvocation)` (`agy_guard.py:304`)
  - Toàn bộ nhánh `if src:` trong `H7 (Stop)` (`agy_guard.py:354-370`)!

### Root Cause 3 (`VERIFIED`): `H1 (PreInvocation)` chỉ chạy 1 lần duy nhất cho cả cuộc hội thoại (`hooks/agy/agy_guard.py:185-188`)
- **Bằng chứng trong code (`hooks/agy/agy_guard.py:185-190`):**
  ```python
  conv_id = payload.get("conversationId", "default")
  flag = sdir / f"{conv_id}.boot"
  if flag.exists():
      return {}
  ```
- **Cơ chế gây lỗi:** Khi một conversation kéo dài qua nhiều task hoặc nhiều câu hỏi của User (như `9d55dea5` hỏi lúc `11:25`, hỏi task mới lúc `14:37`, nhắc lúc `14:57`, giao tiếp lúc `15:37`), `H1` chỉ tiêm `BOOT_MSG` ở lượt đầu tiên (`11:58:10`) rồi tắt hẳn vì file `.boot` đã tồn tại. Khi User giao task mới ở Step 107 (`14:37:01`), không có bất kỳ `ephemeralMessage` nào nhắc Agent phân loại Tier `T0/T1/T2/T3` và mở workspace `S1 → S3`.

### Root Cause 4 (`VERIFIED`): `H4 (PreToolUse)` và `H7 (Stop)` bỏ lọt các lượt Brain / Thiết kế giải pháp (`S1 → S3`) chưa sửa source code (`hooks/agy/agy_guard.py:249-278, 336-388`)
- **Bằng chứng 4a — `H4` chỉ kiểm tra khi sửa file code (`agy_guard.py:249`):**
  - Khi Agent ở vai trò Brain tạo trực tiếp `09_tasks_solution_<suffix>.md` (như ở Step 269 của `9d55dea5`) mà **bỏ qua S1** (không tạo `01_requirements_<suffix>.md`, không tạo `08_tasks_<suffix>.md`, không append `START` vào `05_progress_*.md`), `H4` cho qua vì `09_tasks_solution_*.md` nằm trong `/memory/workspaces/` chứ không phải `is_src()`.
- **Bằng chứng 4b — `H7` không chặn khi Agent trả lời "chay" trên chat (Shadow Response) hoặc khi chưa ghi `05_progress` trong lượt Brain (`agy_guard.py:346-371`):**
  - Ở Step 107–215 của `9d55dea5`, Agent đọc hàng chục file source của `data-hub` để điều tra bug rồi trả lời thẳng trên chat mà không đụng tới bất kỳ file nào trong `memory/workspaces/`. Lúc này:
    - `touched(st, r"09_tasks_solution")` = `False`
    - `touched(st, r"(audit_report|13_analysis)")` = `False`
    - `src` = `[]`
    - `touched(st, r"workspaces/")` = `False`
    - Kết quả: `miss = []` → `H7` trả về `{}` (cho phép dừng mà không bắt lỗi No Shadow Files!).
  - Thậm chí khi Agent có đụng tới `workspaces/` (nhưng không sửa `src`), kiểm tra `if not touched(st, r"05_progress", last):` lại nằm bên trong `if src:` (`agy_guard.py:354-357`), nên lượt Brain không sửa code không bao giờ bị `H7` bắt lỗi quên ghi `05_progress`!
- **Bằng chứng 4c — Bộ đếm `.stop` của `H7` tính gộp trên toàn bộ conversation thay vì theo từng lượt `USER_INPUT` (`agy_guard.py:336-339`):**
  - `cnt_file = sdir / f"{conv_id}.stop"` chỉ cho phép `n < 2` trong suốt vòng đời conversation. Sau 2 lần chặn ở task đầu, mọi task tiếp theo trong cùng conversation bị mất hoàn toàn cổng `H7`!

### Root Cause 5 (`VERIFIED`): `verify_governance.py` cho PASS "ăn theo" các file cũ khi tái sử dụng thư mục Workspace (`tooling/verify_governance.py:98-121`)
- **Bằng chứng trong code (`tooling/verify_governance.py:63-67, 111-115`):**
  - `match_any(ws_path, ("01_requirements*.md",))` chỉ kiểm tra xem trong thư mục workspace có *bất kỳ* file `01_requirements*.md` nào hay không.
- **Bằng chứng thực tế trong `9d55dea5` (Step 221 & Step 346):**
  - Agent dùng lại workspace cũ `fix-recon-daily-flow-cud-2026-10-06` (vốn đã có sẵn các file từ task trước), chỉ tạo thêm `09_tasks_solution_config_Hierarchy_v2.md` mà **không hề tạo `01_requirements_config_Hierarchy_v2.md` hay `08_tasks_config_Hierarchy_v2.md`**, thậm chí **không ghi `05_progress` cho task mới** (vì file `05_progress` đã có sẵn 1 dòng của ngày `2026-10-16` từ sáng!).
  - Thế nhưng khi chạy `verify_governance.py --workspace fix-recon-daily-flow-cud-2026-10-06`, linter vẫn báo `PASSED 🟢`!

---

## 3. ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT (Single Best Solution & Code Demo)

Để đảm bảo **mọi task chạy qua `/agent` đều tuân thủ 100% Master Workflow `S0 → S6` và bị cưỡng chế bởi Hooks `H1–H7`**, chúng ta thực hiện đồng bộ **4 hạng mục sửa lỗi gốc rễ** dưới đây:

### Hạng mục 1: Giải phóng Context Budget `<skills>` & Cưỡng chế nạp Core Skills ở Priority 1–3 + Inline Workflow Checklist vào `GEMINI.md`
1. **Dọn Context Budget ở `~/.gemini/config/skills/`:**
   - Di chuyển các skill ngoại lai không thuộc core sang `~/.gemini/config/skills_archive/` (bảo toàn 100% file, không xóa), và tạo symlink **10 Core Skills** của `/Users/trainguyen/Documents/work/agent/skills/*` trực tiếp vào `~/.gemini/config/skills/` (Priority 3) cũng như khai báo `/Users/trainguyen/Documents/work/data-hub/.agents/skills.json` + `/Users/trainguyen/Documents/work/ data-hub/.agents/hooks.json` (Priority 2).
   - Kết quả: 10/10 Core Skills (`brain-delegate`, `laya-router`, `muscle-execute`, `qa-agent`, `security-agent`, `debug-agent`, `context-manager`, `audit-propose-only`, `workspace-docs`, `session-wrapup`) luôn hiển thị 100% trong `<skills>` System Prompt mà không bao giờ bị cắt do tràn budget.
2. **Inline Checklist bắt buộc của `S1 → S6` vào `GEMINI.md` Rule #9 (`GEMINI.md:66-75`):**
   - Ngay cả khi Agent chưa gọi `view_file` lên `SKILL.md`, `GEMINI.md` đã ghi rõ ràng điều kiện rẽ nhánh `T0` vs `T1–T3/Audit` và đường dẫn tuyệt đối tới `/Users/trainguyen/Documents/work/agent/skills/<skill>/SKILL.md` bắt buộc phải đọc ở từng bước.

#### Code Demo — Cập nhật `GEMINI.md` (Rule #9, dòng 66–75):
```markdown
### 9. Master Workflow (`S0 → S6`) & Nguyên tắc "Plan & Verify"
- **Chỉ coi là `T0`** khi User hỏi lý thuyết/tra cứu nhỏ (<3 file) và KHÔNG yêu cầu phân tích lỗi, thiết kế giải pháp, audit hay sửa hệ thống. **Mọi yêu cầu kiểm tra lỗi / phân tích luồng / đề xuất giải pháp / sửa code đều tối thiểu là `T2` hoặc `Audit` — CẤM trả lời chay trên chat (No Shadow Files)!**
1. **S0 BOOT:** Hook H1 kiểm tra Laya (`:8200`); đọc `GEMINI.md` + `lessons/INDEX.md`; xác định `T0` | `Audit` | `T1–T3` và chọn `<suffix>` định danh cho task hiện tại.
2. **S1 INTAKE (T1–T3 — BẮT BUỘC TRƯỚC KHI ĐIỀU TRA SÂU):** Đọc `/Users/trainguyen/Documents/work/agent/skills/workspace-docs/SKILL.md`. Tạo/chọn workspace `<ws>`; tạo `01_requirements_<suffix>.md` (`tier: T1|T2|T3`), append `START` vào `05_progress_*.md`, tạo `08_tasks_<suffix>.md`. (Hook H4 sẽ chặn tạo `09_tasks_solution` hoặc sửa code nếu thiếu bước S1 này).
3. **S2 INVESTIGATE:** Đọc `/Users/trainguyen/Documents/work/agent/skills/debug-agent/SKILL.md` (nếu là bug). Chỉ đọc (`view_file`, `grep`, log, DB read-only). Phân biệt `VERIFIED` và `HYPOTHESIS`.
4. **S3 SOLUTION:** Đọc `/Users/trainguyen/Documents/work/agent/skills/brain-delegate/SKILL.md`. Trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** (cấm Option 1/2/3) kèm code demo vào `09_tasks_solution_<suffix>.md` (hoặc `13_analysis_<suffix>.md` nếu chỉ Audit).
5. **DG1 + DG2:** Đọc `/Users/trainguyen/Documents/work/agent/skills/laya-router/SKILL.md`. Chạy `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run` (`--stage dg1`, `--stage dg2`). Append kết quả vào `05_progress_*.md`. Đạt `PASS` → trình User, **DỪNG CHỜ `APPROVE`**.
6. **S4 EXECUTE (sau `APPROVE`):** Đọc `/Users/trainguyen/Documents/work/agent/skills/muscle-execute/SKILL.md`. Tạo git restore-point local → sửa code → chạy `checks`, `test`, `runtime_check` theo `.agent-project.json` + gọi request thật.
7. **S5 SELF-AUDIT:** Chạy `DG3` (`laya_gate.py run --stage dg3` / `review`) + đọc `skills/security-agent/SKILL.md` & `skills/qa-agent/SKILL.md`. **Từ S5 là Propose-Only:** phát hiện ngoài phạm vi đã duyệt chỉ đề xuất, chờ `APPROVE` mới sửa.
8. **S6 CLOSE:** Đọc `/Users/trainguyen/Documents/work/agent/skills/session-wrapup/SKILL.md`. Ghi `06_validation_<suffix>.md`, `11_report_<suffix>.md`, `12_implementation_plan_<suffix>.md` (T3), append `DONE` vào `05_progress_*.md`, chạy `verify_governance.py --workspace <ws> --suffix <suffix>`.
```

---

### Hạng mục 2 & 3: Sửa lỗi `tgt()` strip quotes và Nâng cấp Hooks `H1`, `H4`, `H7` trong `hooks/agy/agy_guard.py`

1. **Chuẩn hóa `clean_val()` / `tgt()` (`agy_guard.py:82-95`):**
   - Antigravity lưu giá trị string trong `transcript.jsonl` với dấu `"` bao ngoài (`"\"/path/to/file.go\""`). Ta thêm hàm `_unquote(v)` để bóc cặp dấu `"` bao ngoài cho mọi trường trong `args` khi đọc từ `transcript.jsonl` cũng như trong `tgt(args)` và `cmd`.
2. **Nâng cấp `H1 (handle_pre_invocation)` chạy theo từng lượt `USER_INPUT` mới:**
   - Thay vì tạo 1 file `{conv_id}.boot` rồi câm luôn cả cuộc hội thoại, `H1` đọc `last_user_step_idx(st)` từ transcript và lưu vào `{conv_id}.boot`. Mỗi khi có lượt `USER_INPUT` mới (`last_user_idx > saved_idx`), `H1` sẽ tiêm thông điệp nhắc nhở `BOOT S0` + trạng thái Laya `:8200` + nhắc tạo `01_requirements_<suffix>.md`, `08_tasks_<suffix>.md`, `05_progress` `START` trước khi lập `09_tasks_solution_<suffix>.md`!
3. **Nâng cấp `H4 (handle_pre_write)` chặn làm tắt bước `S3` khi chưa qua `S1`:**
   - Khi Agent gọi `write_to_file` hoặc `replace_file_content` vào `09_tasks_solution_<suffix>.md` trong `memory/workspaces/<ws>/`:
     - Bắt buộc trong `<ws>` phải có `01_requirements_<suffix>.md` (hoặc `01_requirements_*.md` được tạo/cập nhật) và `08_tasks_<suffix>.md`. Nếu chưa có → trả về `{"decision": "deny", "reason": "Rule 9 (S1 -> S3): Bắt buộc tạo 01_requirements_<suffix>.md, 08_tasks_<suffix>.md và append START vào 05_progress_*.md trước khi viết 09_tasks_solution_<suffix>.md!"}`.
     - Bắt buộc `05_progress_*.md` đã được chạm tới (`touched(st, r"05_progress", last_user_idx)`) trong lượt hiện tại!
4. **Nâng cấp `H7 (handle_stop)` theo từng lượt `USER_INPUT` (`turn_start = last_user_step_idx(st)`):**
   - Bộ đếm `.stop` gắn theo lượt User: `sdir / f"{conv_id}.{turn_start}.stop"` (tối đa 2 lần chặn cho mỗi lượt User, không bị liệt ở các lượt sau).
   - Chỉ xét các thao tác từ `turn_start` (`after=turn_start`):
     - **Chặn Shadow Analysis / Investigation không có Workspace (`No Shadow Files`):** Nếu trong lượt này Agent gọi `> 4` tool calls để đọc/tra cứu file code dự án (`view_file`, `grep_search`, `run_command` trên project) hoặc sửa file trong `workspaces/` hoặc sửa `src`, mà chưa cập nhật `05_progress` (`not touched(st, r"05_progress", turn_start)`) → chặn yêu cầu ghi hồ sơ workspace và append `05_progress`!
     - Nếu trong lượt này chạm tới `09_tasks_solution_<suffix>.md`: bắt buộc `DG1` và `DG2` đạt `PASS`, đồng thời chạy `verify_governance.py --workspace <ws> --suffix <suffix> --stage S3` để kiểm tra đủ `01_requirements_<suffix>.md`, `08_tasks_<suffix>.md`, `09_tasks_solution_<suffix>.md`, và `05_progress_*.md`!

#### Code Demo — `hooks/agy/agy_guard.py`:
```python
def _unquote(val: Any) -> Any:
    if isinstance(val, str) and len(val) >= 2 and val[0] == '"' and val[-1] == '"':
        try:
            return json.loads(val)
        except Exception:
            return val.strip('"')
    return val


def calls(st: List[Dict[str, Any]]) -> Iterator[Tuple[int, Optional[str], Dict[str, Any]]]:
    for i, s in enumerate(st):
        for tc in s.get("tool_calls") or []:
            raw_args = tc.get("args") or {}
            clean_args = {k: _unquote(v) for k, v in raw_args.items()}
            yield i, tc.get("name"), clean_args


def last_user_idx(st: List[Dict[str, Any]]) -> int:
    for i in range(len(st) - 1, -1, -1):
        if st[i].get("type") == "USER_INPUT":
            return i
    return -1


def extract_suffix(filename: str, prefix: str) -> Optional[str]:
    m = re.match(rf"^{re.escape(prefix)}_(.+)\.md$", filename)
    return m.group(1) if m else None
```
Và trong `handle_pre_write` (`H4`):
```python
    # H4a: S1 Intake Gate before S3 Solution (Rule #4, #9, #16)
    sol_match = re.search(r"/memory/workspaces/([^/]+)/09_tasks_solution_([^/]+)\.md$", f)
    if sol_match and is_mem_path(f, mem_root):
        ws_name, suffix = sol_match.group(1), sol_match.group(2)
        ws_dir = (ws_root or WS_ROOT) / ws_name
        st = steps(payload)
        u_idx = last_user_idx(st)
        req_ok = (ws_dir / f"01_requirements_{suffix}.md").exists() or touched(st, r"01_requirements_", u_idx)
        tsk_ok = (ws_dir / f"08_tasks_{suffix}.md").exists() or touched(st, r"08_tasks_", u_idx)
        prg_ok = touched(st, r"05_progress", u_idx)
        missing_s1 = []
        if not req_ok:
            missing_s1.append(f"01_requirements_{suffix}.md")
        if not tsk_ok:
            missing_s1.append(f"08_tasks_{suffix}.md")
        if not prg_ok:
            missing_s1.append("append START vào 05_progress_*.md trong lượt này")
        if missing_s1:
            return {
                "decision": "deny",
                "reason": (
                    f"Rule 9 (S1 Intake trước S3 Solution): Không được nhảy cóc tạo 09_tasks_solution_{suffix}.md "
                    f"khi chưa hoàn thành S1: {', '.join(missing_s1)}."
                ),
            }
```

---

### Hạng mục 4: Nâng cấp `tooling/verify_governance.py` hỗ trợ `--suffix <suffix>` và `--stage S3|S6`
- **Vấn đề cũ:** Khi tái sử dụng một workspace cũ (ví dụ `fix-recon-daily-flow-cud-2026-10-06`), các file `01_requirements_*.md` và `08_tasks_*.md` từ task cũ làm `verify_governance.py` báo `PASSED` ảo dù task mới thiếu `01` và `08`. Ngoài ra, ở giai đoạn `S3` (trình giải pháp chờ `APPROVE`), Agent chưa thể có `06_validation_*.md` hay `11_report_*.md` (vốn thuộc `S6`), nên cần phân biệt `--stage S3` (kiểm tra hồ sơ `S1–S3`: `01`, `05`, `08`, `09`) và `--stage S6` (mặc định, kiểm tra toàn bộ hồ sơ đóng phiên `01`, `05`, `06`, `08`, `09`, `11`...).
- **Thiết kế mới:**
  - Hỗ trợ `--suffix <suffix>`: Khi truyền `--suffix <suffix>` (hoặc tự động phát hiện nếu file `09_tasks_solution_<suffix>.md` mới nhất trong workspace không có `01_requirements_<suffix>.md` / `08_tasks_<suffix>.md` tương ứng và workspace có `>1` file `09_tasks_solution_*.md`), linter kiểm tra đúng cặp tài liệu theo `<suffix>`.
  - Hỗ trợ `--stage S3|S6` (mặc định `S6`):
    - Ở `S3`: Kiểm tra đủ `01_requirements`, `05_progress`, `08_tasks`, `09_tasks_solution` + summary `06_laya_summary_dg1_*.json` (`PASS`) & `06_laya_summary_dg2_*.json` (`PASS`).
    - Ở `S6`: Kiểm tra đầy đủ theo `TIER_RULES[tier]` + format và ngày của `05_progress`.

---

## 4. Phân tích Tác động & Tương thích (Impact Analysis)

| Thành phần | File tác động | Rủi ro | Kiểm chứng bắt buộc |
|---|---|---|---|
| System Prompt `<skills>` | `~/.gemini/config/skills/` (archive 300+ skill ngoại lai sang `skills_archive/`, symlink 10 Core Skills) & `/Users/trainguyen/Documents/work/data-hub/.agents/{skills,hooks}.json` | Không ảnh hưởng code dự án; giảm ngay ~18KB rác khỏi System Prompt của mọi phiên. | Kiểm tra `ls -la ~/.gemini/config/skills/` chỉ gồm 10 Core Skills. |
| Core Rules | `/Users/trainguyen/Documents/work/agent/GEMINI.md` | Tăng ~900 bytes (từ 11.9KB lên ~12.8KB, vẫn rất xa ngưỡng 24KB). | Kiểm tra `wc -c GEMINI.md < 15000`. |
| Lifecycle Hooks H1–H7 | `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py` & `hooks/agy/tests/test_agy_guard.py` | Sửa đúng bug quote-stripping trên `transcript.jsonl` thật và thêm kiểm tra S1→S3. | Chạy unit tests (bao gồm test fixture với `TargetFile` bọc dấu `"` như `transcript.jsonl` thật của `9d55dea5`). |
| Process Linter | `/Users/trainguyen/Documents/work/agent/tooling/verify_governance.py` & `tooling/tests/test_verify_governance.py` | Tương thích ngược 100% với CLI hiện tại (`--workspace <ws>`), bổ sung `--suffix` và `--stage`. | Chạy toàn bộ test suite `unittest` PASS 100%. |
