# 09 — Tasks Solution: Tích hợp Cổng Mandatory Final Adversarial Audit (`audit_report_*.md`) & Tự động Kích hoạt bằng Hook (`H6/H7`)

- **Workspace:** `agent-process-optimization`
- **Suffix:** `mandatory_final_audit`
- **Tier:** `T2`
- **Ngày lập:** `2026-10-19`
- **Trạng thái:** `PROPOSE-ONLY (Chờ User APPROVE)`

---

## 1. Phân tích Nguyên nhân Gốc rễ (`VERIFIED` kèm `file:line`)

User phản ánh:
> *"sau khi tao làm xong, tao chạy 'Audit lại quá trình vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. Kiểm tra các xem có suy diễn, báo cáo láo không. Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop). Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng Lưu toàn bộ vào file audit_report_*.md trong workspace của task Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.' => và vẫn ra rất nhiều bug. mày thêm cái audit vào vào cuối workflow cho tao. và luôn chạy hook script để nó luôn đc kích hoạt, chứ tao rất phiền vì phải đi nhắc"*

Qua đối chiếu trực tiếp mã nguồn hiện tại của `/agent`, có **3 lỗ hổng thiết kế (`VERIFIED`)** khiến Agent không tự động chạy bước Audit phản biện gắt gao này ở cuối mỗi task mà đợi User nhắc:

1. **Lỗ hổng 1 (`VERIFIED` — `GEMINI.md:41,73-74` & `tooling/verify_governance.py:31-52`):**
   - Trong `GEMINI.md:41` và `tooling/verify_governance.py:31-52`, bộ tài liệu bắt buộc của Tier `T2` và `T3` khi đóng phiên (`S6`) chỉ yêu cầu `06_validation_*.md` và `11_report_*.md`, **hoàn toàn chưa bắt buộc phải có `audit_report_<suffix>.md`**.
   - Trong `GEMINI.md:73` (`S5 SELF-AUDIT`), quy tắc mới chỉ ghi ngắn gọn *"Chạy DG3 + /security-agent + /qa-agent"*, chưa bắt buộc thực hiện quy trình **6 bước Audit Phản biện Toàn trình & Từng dòng (`Line-by-Line Diff Critique`, `Anti-Speculation/Anti-Lies`, `Self-Improvement Loop`, `Anti-Mockup & Cooperate User Runtime Trace`, `Full-Context /laya_router DG3 Review`, `Save audit_report_<suffix>.md`)**.

2. **Lỗ hổng 2 (`VERIFIED` — `hooks/agy/agy_guard.py:372-405` — Hook `H6 PostInvocation` im lặng khi unit test pass):**
   - Trong `handle_post_invocation` (`hooks/agy/agy_guard.py:401-405`):
     ```python
     if errs:
         return {"injectSteps": [{"ephemeralMessage": "⚠️ H6 fast-feedback:\n" + "\n".join(errs)}]}
     return {}
     ```
   - Khi Agent vừa sửa code xong và chạy unit test (thường dùng mockup/closure giả) thấy `returncode == 0`, `errs` rỗng nên `H6` trả về `{}` (im lặng tuyệt đối). Không có bất kỳ hook message nào tự động kích hoạt ép Agent bước vào pha **Audit phản biện từng dòng update (`git diff`) + chống mockup + gọi `/laya_router` + xuất `audit_report_<suffix>.md`**.

3. **Lỗ hổng 3 (`VERIFIED` — `hooks/agy/agy_guard.py:432, 443-458` — Hook `H7 Stop` không chặn khi thiếu `audit_report_*.md` hoặc khi dùng lại `DG3` cũ):**
   - Tại `hooks/agy/agy_guard.py:432`:
     ```python
     if touched(st, r"(audit_report|13_analysis)", u_idx) and laya_gate_status(ws, "dg3", ws_root) != "PASS":
     ```
     Câu lệnh này có nghĩa là: **nếu Agent KHÔNG tạo `audit_report_*.md` thì không bị kiểm tra**!
   - Tại `hooks/agy/agy_guard.py:443-458` (`if src:`):
     ```python
     if src:
         last = src[-1][0]
         if not touched(st, r"05_progress", last):
             miss.append("05_progress: chưa append kết quả sau lần sửa code cuối")
         if not touched(st, r"06_validation", last):
             miss.append("06_validation: chưa ghi bằng chứng test/runtime sau lần sửa code cuối")
         if laya_gate_status(ws, "dg3", ws_root) != "PASS":
             miss.append("DG3: chưa self-audit thay đổi bằng laya_gate")
     ```
     - Nhánh `if src:` **không hề kiểm tra `touched(st, r"audit_report_", last)`**!
     - Hơn nữa, `laya_gate_status(ws, "dg3", ws_root)` chỉ đọc file `06_laya_summary_dg3_*.json` bất kỳ có sẵn trong thư mục workspace — nếu trong workspace đã có sẵn 1 file `dg3` từ phase trước, Agent sửa code mới ở phase sau vẫn được `laya_gate_status` trả về `"PASS"` mà không cần gọi lại `/laya_router` sau lần sửa code cuối (`last`)!

---

## 2. ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT (Single Best Solution & Code Demo)

Chúng ta khoá chặt quy trình bằng **4 thay đổi đồng bộ** từ Hiến pháp `GEMINI.md` $\rightarrow$ Core Skills $\rightarrow$ Hook `H6 (PostInvocation)` tự động kích hoạt $\rightarrow$ Hook `H7 (Stop)` & `verify_governance.py` chặn cứng ở cuối workflow:

### 2.1. Hạng mục 1 (`T7.1` & `T7.2`): Chuẩn hóa Bước `S5 MANDATORY FINAL ADVERSARIAL AUDIT` trong `GEMINI.md` và 4 Core Skills
Cập nhật `GEMINI.md` (Rule #4, Rule #9 S5/S6, Rule #14 G3/G7/G8, Rule #18) và `skills/{qa-agent,session-wrapup,workspace-docs,muscle-execute}/SKILL.md`:
- Bổ sung `audit_report_<suffix>.md` vào danh sách file bắt buộc của **Tier T2** và **Tier T3**.
- Quy định rõ tại **S5 SELF-AUDIT (Mandatory Final Adversarial Audit Protocol)**:
  Ngay sau khi sửa code ở S4 (và trước khi báo Done ở S6), Agent **BẮT BUỘC tự động thực hiện 6 bước Audit phản biện (được kích hoạt bởi Hook H6 và chặn cứng bởi Hook H7)**:
  1. **Tư duy Phản biện Quá trình & Từng Dòng Code (`Line-by-Line Diff Critique`):** Chạy `git diff` trên mọi repo đã sửa; phản biện từng task đã làm, từng file đã update, và **từng dòng update** (kiểm tra race condition, context timeout vs heartbeat, nuốt lỗi, rò rỉ tài nguyên, sai lệch biên dữ liệu).
  2. **Kiểm tra Chống Suy diễn & Báo cáo Láo (`Anti-Speculation & Honesty Verification`):** Mọi khẳng định phải có `file:line` và output thực tế đối chứng; cấm kết luận vượt quá phạm vi dữ liệu đã kiểm chứng.
  3. **Cấm Test bằng Mockup & Hallucinate Kết quả Runtime (`Zero-Mockup & Real Runtime Trace`):**
     - **NGHIÊM CẤM** gắn closure/function mock vào struct production hoặc dùng unit test mockup để giả lập DB/Broker/Service nhằm báo PASS giả tạo.
     - **Nếu không thể tự trace runtime value thật** (do lỗi nằm trên môi trường Production/Staging hoặc cần thao tác/dữ liệu từ phía User), **BẮT BUỘC phải nói rõ và chủ động phối hợp với User để trace runtime value thật**, tuyệt đối không tự bịa (hallucinate) kết quả!
  4. **Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (`Self-Improvement Loop`):** Nếu phát hiện bất kỳ lỗi/thiếu sót nào trong quá trình Audit, phải ghi nhận bài học vào `lessons.md` (`build_lessons_index.py`) và khắc phục ngay (nếu nằm trong phạm vi đã `APPROVE`) hoặc báo cáo đề xuất rõ ràng (nếu ngoài phạm vi).
  5. **Chạy QC Gắt gao & Gọi `/laya_router` (`DG3`) Nhận xét Quyết định:** Soạn context đầy đủ (mục tiêu, luồng hàm, từng dòng diff, kết quả trace runtime, kèm `>=2 control_false` & `>=1 control_true`) và gọi `laya_gate.py run --stage dg3` (và/hoặc `laya_gate.py review`) **SAU lần sửa code cuối cùng** để `/laya_router` đánh giá độc lập xem quyết định và code thực thi đã thực sự ổn chưa.
  6. **Lưu Biên bản Vật lý vào `audit_report_<suffix>.md`:** Lưu toàn bộ nội dung 5 bước trên vào `agent/memory/workspaces/<ws>/audit_report_<suffix>.md`.

#### Code Demo — Cập nhật `GEMINI.md` (Rule #4 & Rule #9, dòng 40–76):
```markdown
  - **T2 (Feature/Bugfix):** `T1` + `09_tasks_solution_*.md`, `06_validation_*.md`, `11_report_*.md`, `audit_report_*.md` (`tier: T2`).
...
7. **S5 MANDATORY FINAL AUDIT (Tự động kích hoạt bởi Hook H6, chặn cứng bởi Hook H7):** Đọc `agent/skills/qa-agent/SKILL.md` & `agent/skills/security-agent/SKILL.md`. Thực hiện **6 bước QC phản biện bắt buộc**:
   - (a) **Phản biện từng dòng (`git diff`):** Rà soát phản biện toàn bộ quá trình, từng task đã làm, từng file đã update, từng dòng update.
   - (b) **Chống suy diễn & báo cáo láo:** Kiểm tra chéo mọi kết luận với `file:line` và output thật.
   - (c) **Zero-Mockup & Runtime Trace:** **CẤM** test bằng mockup/closure giả trên struct production hoặc hallucinate kết quả; nếu thiếu môi trường/dữ liệu phải chủ động báo **phối hợp với User trace runtime value thật**.
   - (d) **Self-Improvement Loop:** Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (ghi `lessons.md` + sửa triệt để lỗi phát hiện trong phạm vi đã duyệt; ngoài phạm vi thì giữ Propose-Only chờ `APPROVE`).
   - (e) **Hội đồng `/laya_router` (`DG3`):** Soạn context đầy đủ & chạy `laya_gate.py run --stage dg3` (hoặc `review`) **sau lần sửa code cuối** để nhận xét lại quyết định đã ổn chưa.
   - (f) **Lưu `audit_report_<suffix>.md`:** Ghi toàn bộ kết quả (a)–(e) vào `audit_report_<suffix>.md` trong workspace.
8. **S6 CLOSE:** Đọc `agent/skills/session-wrapup/SKILL.md`. Ghi `06_validation_<suffix>.md`, `11_report_<suffix>.md`, `12_implementation_plan_<suffix>.md` (T3), `audit_report_<suffix>.md`, append `DONE` vào `05_progress_*.md`, chạy `verify_governance.py --workspace <ws> --suffix <suffix>`.
```

---

### 2.2. Hạng mục 2 (`T7.3`): Nâng cấp `H6 (PostInvocation)` & `H7 (Stop)` trong `hooks/agy/agy_guard.py`

1. **Tại `H6 (handle_post_invocation)` — Tự động tiêm lệnh kích hoạt Audit ngay sau khi sửa code:**
   - Khi có bất kỳ thay đổi source code nào (`new_edits = [f for i, f in src_edits(st, mem_root) if i > last]`), dù `checks` thành công hay thất bại, `H6` luôn tiêm `AUDIT_TRIGGER_MSG` vào `injectSteps` để nhắc Agent bắt buộc chạy ngay quy trình Audit phản biện cuối workflow trước khi kết thúc lượt!
2. **Tại `H7 (handle_stop)` — Chặn cứng (`decision: "continue"`) nếu chưa hoàn thành Audit phản biện cuối workflow:**
   - Khi `src` khác rỗng (`last = src[-1][0]`):
     - **Kiểm tra 1:** Bắt buộc đã tạo/cập nhật `audit_report_<suffix>.md` **sau** lần sửa code cuối (`if not touched(st, r"audit_report_", last):`).
     - **Kiểm tra 2:** Kiểm tra nội dung file `audit_report_*.md` mới nhất trong workspace (`check_audit_report_sections(ws, ws_root)`): bắt buộc có đủ các phần:
       - Phản biện từng file / từng dòng update (`git diff` / `line-by-line` / `từng dòng`)
       - Kiểm tra suy diễn / báo cáo láo & Anti-Mockup / Runtime trace (`suy diễn` / `mockup` / `runtime`)
       - Vòng lặp Phản tỉnh & Khắc phục (`Self-Improvement` / `Phản Tỉnh` / `Phản tỉnh`)
       - Nhận xét của `/laya_router` (`DG3` / `laya_router` / `laya_gate`)
     - **Kiểm tra 3:** Kiểm tra `DG3` (`laya_gate.py` stage `dg3`) không chỉ đạt `PASS` mà còn **phải được chạy sau lần sửa code cuối** (`touched(st, r"(laya_gate\.py.*dg3|laya_gate\.py.*review|06_laya_.*dg3)", last)` hoặc file `06_laya_summary_dg3_*.json` được tạo trong lượt sau khi sửa code).

#### Code Demo — `hooks/agy/agy_guard.py` (Chặn đứng Suy diễn Smoke Test ở cả `H4`, `H6`, `H7`):
```python
PROD_MOCK_POLLUTION = re.compile(
    r"(\b\w+Fn\s+func\s*\(|\bsqlmock\b|\bfakemodel\b|\bmock[A-Z]\w+\s+func\s*\()",
)

FINAL_AUDIT_MSG = (
    "🔍 [H6 AUTO-AUDIT & ANTI-SMOKE-SPECULATION TRIGGER] Bạn vừa cập nhật source code ({files}). "
    "TRƯỚC KHI DỪNG HOẶC BÁO DONE, BẮT BUỘC thực hiện quy trình Mandatory Final Adversarial Audit (S5):\n"
    "1. [Line-by-Line Diff Critique] Tư duy phản biện cho toàn bộ quá trình vừa thực hiện, các task đã làm, các file đã update, và TỪNG DÒNG update (`git diff`).\n"
    "2. [CHẶN ĐỨNG SUY DIỄN SMOKE TEST] Tuyệt đối CẤM lấy kết quả unit test in-memory, closure mock, scratch script, hay happy-path 1 bản ghi (smoke test) để suy diễn rằng toàn bộ luồng runtime thực tế đã chạy đúng! CẤM gắn closure mock (`*Fn`) vào struct production.\n"
    "3. [Concurrency / Timeout / Continuity Check] Phản biện tương tác đa worker (`WorkerCount > 1`), gom lô (`batch/debouncer`), xung đột giữa `AckWait/Context Timeout` với `Heartbeat/InProgress`, và tính liên tục của `Watermark/Cursor` khi chạy song song hoặc sau sự cố Outage.\n"
    "4. [Zero-Mockup & Real Runtime Trace] Kiểm tra xem có suy diễn, báo cáo láo không. Không test bằng mockup và Hallucination kết quả; nếu không tự trace được trên môi trường thật thì BẮT BUỘC phải nói rõ phối hợp với User trace runtime value thật.\n"
    "5. [Self-Improvement Loop] Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục.\n"
    "6. [Laya Router DG3] Soạn context đầy đủ & gọi `/laya_router` (`laya_gate.py run --stage dg3`) SAU lần sửa code cuối để nhận xét lại xem quyết định làm như vậy đã ổn chưa, và lưu toàn bộ vào `audit_report_<suffix>.md` trong workspace."
)

AUDIT_REQUIRED_PATTERNS = [
    ("1. Phản biện từng dòng update (Line-by-Line Diff)", re.compile(r"(từng dòng|line-by-line|git\s+diff)", re.I)),
    ("2. Chống suy diễn Smoke Test & Đối chứng Runtime thật", re.compile(r"(smoke\s*test|anti-smoke|in-memory|mockup|runtime\s+value|trace\s+runtime)", re.I)),
    ("3. Phản biện Concurrency / Timeout / Data Continuity", re.compile(r"(concurrency|timeout|worker|batch|watermark|cursor|edge-case|biên)", re.I)),
    ("4. Kiểm tra suy diễn / báo cáo láo", re.compile(r"(suy diễn|báo cáo láo|hallucinat|speculat)", re.I)),
    ("5. Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop)", re.compile(r"(phản tỉnh|self-improvement)", re.I)),
    ("6. Nhận xét từ /laya_router (DG3)", re.compile(r"(laya_router|laya_gate|dg3)", re.I)),
]


def is_test_file(path_str: str) -> bool:
    name = Path(path_str).name.lower()
    return (
        name.endswith("_test.go")
        or name.endswith(".test.ts")
        or name.endswith(".test.tsx")
        or name.endswith(".spec.ts")
        or name.endswith(".spec.js")
        or name.startswith("test_")
        or "/tests/" in path_str.replace("\\", "/")
    )


def audit_report_issues(ws: Optional[str], ws_root: Optional[Path] = None) -> List[str]:
    if not ws:
        return ["Chưa xác định được workspace để kiểm tra audit_report_*.md"]
    ws_dir = (ws_root or WS_ROOT) / ws
    try:
        reports = sorted(ws_dir.glob("audit_report*.md"), key=lambda p: p.stat().st_mtime) if ws_dir.exists() else []
    except OSError:
        return []
    if not reports:
        return [f"Thiếu file audit_report_<suffix>.md trong {ws_dir}"]
    latest = reports[-1]
    try:
        content = latest.read_text(encoding="utf-8", errors="ignore")
    except OSError:
        return []
    missing_sec = [label for label, pat in AUDIT_REQUIRED_PATTERNS if not pat.search(content)]
    if missing_sec:
        return [f"{latest.name} thiếu các nội dung bắt buộc: {', '.join(missing_sec)}"]
    return []
```

---

### 2.3. Hạng mục 3 (`T7.4`): Nâng cấp `tooling/verify_governance.py` kiểm tra `audit_report_*.md`
- Bổ sung `("audit_report_*.md", ("audit_report*.md",))` vào `TIER_RULES["T2"]` và `TIER_RULES["T3"]` (chặng `S6`).
- Bổ sung `"audit_report"` vào `SUFFIX_SCOPED_PREFIXES` (để khi dùng `--suffix <suffix>` ở chặng `S6`, bắt buộc phải có `audit_report_<suffix>.md`).
- Trong `verify_workspace_docs()`, khi ở chặng `S6` với Tier `T2` hoặc `T3`, kiểm tra file `audit_report_*.md` mới nhất có đủ 5 nhóm nội dung bắt buộc (`AUDIT_REQUIRED_PATTERNS`).

---

## 3. Phân tích Tác động & Kế hoạch Rollback (Blast Radius & Rollback Plan)

| Thành phần | File thay đổi | Tác động | Kế hoạch Rollback |
|---|---|---|---|
| Hiến pháp Lõi | `GEMINI.md` | Chuẩn hóa S5/S6 bắt buộc có `audit_report_<suffix>.md` + 6 tiêu chí Audit phản biện (`~13.1 KB < 24 KB`). | Khôi phục từ `/tmp/agy/backup_phase7/GEMINI.md.bak` hoặc `git checkout -- GEMINI.md`. |
| Core Skills | `skills/{qa-agent,session-wrapup,workspace-docs,muscle-execute}/SKILL.md` | Bổ sung template chuẩn 5 phần của `audit_report_<suffix>.md` và quy tắc Zero-Mockup / Runtime Trace. | Khôi phục từ bản `.bak` trong `/tmp/agy/backup_phase7/`. |
| Lifecycle Hooks | `hooks/agy/agy_guard.py` & `hooks/agy/tests/test_agy_guard.py` | `H6` tự động tiêm `FINAL_AUDIT_MSG` ngay sau khi sửa code; `H7` chặn dừng nếu thiếu `audit_report_*.md` hợp lệ sau lần sửa code cuối. | Khôi phục từ `/tmp/agy/backup_phase7/agy_guard.py.bak`. |
| Process Linter | `tooling/verify_governance.py` & `tooling/tests/test_verify_governance.py` | Bắt buộc `audit_report_*.md` cho Tier `T2/T3` ở `S6` và kiểm tra đủ 5 mục phản biện. | Khôi phục từ `/tmp/agy/backup_phase7/verify_governance.py.bak`. |

---

## 4. Kết quả Kiểm định Decision Gate (`DG1` & `DG2`)

| Cổng | Summary File | Trusted Rounds | Các Claims đã kiểm chứng | Kết quả Gate |
| :--- | :--- | :---: | :--- | :---: |
| **DG1 (Solution)** | `06_laya_summary_dg1_20261019T114016.json` | `r3` | `S-RC` (`ACCEPT` 0.92), `S-FIX` (`ACCEPT` 0.94), `S-IMPACT` (`ACCEPT` 0.94), `S-PATTERN` (`ACCEPT` 0.89) | **`PASS`** |
| **DG2 (Code Demo)** | `06_laya_summary_dg2_20261019T114123.json` | `r2, r3` | `D-BEHAV` (`ACCEPT` 0.73), `D-EDGE` (`FLAG` resolved via `REF: hooks/agy/agy_guard.py:428-450`), `D-API` (`ACCEPT` 0.95) | **`PASS`** |
