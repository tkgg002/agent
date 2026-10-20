---
tier: T2
workspace: agent-process-optimization
suffix: verbatim_audit_hook_and_self_heal_loop
---

# 09 — Tasks Solution: Nhúng Nguyên Văn Prompt Audit của User vào Hook & Tự Động Khép Kín Vòng Lặp Tự Khắc Phục (`S5 → S3 → S4 → S5`) Không Bắt User `APPROVE` Lần 2

## 1. Root Cause Analysis (Truy vết nguyên nhân gốc rễ kèm `file:line` — `VERIFIED`)

Từ thực tế chạy thử trên các workspace `audit-shadow-master-update-drop-2026-10-20` (`commit ae7e8c1`) và `feature-harness2-recursive-integration-2026-10-20` (`commit eda499a`), có **3 nguyên nhân gốc rễ (`RC-1`, `RC-2`, `RC-3`)** khiến Agent báo "OK" khi vẫn còn 5–7 bug/lỗ hổng, và sau đó khi chạy Audit lại bắt User phải `APPROVE` lần 2:

### 1.1. `RC-1` (`VERIFIED` tại [`hooks/agy/agy_guard.py:78-91`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L78-L91)): Tự ý biên tập lại câu chữ thay vì lấy nguyên văn Prompt Audit của User
- Trong `hooks/agy/agy_guard.py:78-91`, hằng số `FINAL_AUDIT_MSG` đã tự ý viết lại/tóm tắt yêu cầu của User thành các gạch đầu dòng thay vì nhúng **nguyên văn 100% từng chữ** prompt mà User đã chỉ đạo:
  ```text
  Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update. 
  Kiểm tra các xem có suy diễn, báo cáo láo không. 
  Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).
  Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng 
  Lưu toàn bộ vào file audit_report_*.md trong workspace của task
  Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.
  ```

### 1.2. `RC-2` (`VERIFIED` tại [`hooks/agy/agy_guard.py:630-668, 685-772`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L630-L772)): Bơm Audit sai thời điểm (bị xóa bởi Context Compaction) & Cho phép vượt cổng `H7` ngay lượt đầu (`n == 0`) bằng báo cáo hình thức
- Tại `handle_post_invocation` (`H6`, `L630-668`), `FINAL_AUDIT_MSG` chỉ được bơm khi `new_edits` (sửa source code) khác rỗng — tức là ngay lúc Agent đang gõ dở các file code ở đầu chặng `S4`.
- Khi Agent chạy test/build kéo dài dẫn tới **Nén ngữ cảnh (`Context Compaction`)**, các `ephemeralMessage` của `H6` bị xóa khỏi cửa sổ ngữ cảnh. Đến cuối lượt, Agent đang ở tâm lý "người viết code muốn đóng task" (Builder Bias) nên tự điền nhanh một file `audit_report_*.md` toàn chữ `PASS` theo trí nhớ (thậm chí không chạy `git diff` sau lần sửa code cuối, dẫn tới ghi sai cả tên file/port như tại `ae7e8c1:audit_report_shadow_master_update_drop.md`).
- Khi Agent gọi kết thúc lượt (`handle_stop` — `H7`, `L685-772`), vì thấy file `audit_report_*.md` đã tồn tại trên đĩa nên ở ngay lần dừng đầu tiên (`n == 0`), `H7` trả về `{}` cho phép Agent báo "OK" luôn mà **không hề chặn lại bằng một lượt Phản Biện Độc Lập (Stop-Bounce) chứa nguyên văn Prompt Audit của User** như khi User paste thủ công!

### 1.3. `RC-3` (`VERIFIED` tại [`hooks/agy/agy_guard.py:577-602`](file:///Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py#L577-L602) & [`GEMINI.md:78`](file:///Users/trainguyen/Documents/work/agent/GEMINI.md#L78)): Khóa cứng quyền tự sửa lỗi trong `S5` bắt User phải `APPROVE` lần 2
- Tại `hooks/agy/agy_guard.py:585-597` (được thêm ở commit `ae7e8c1`):
  ```python
  if last_aud_idx > last_sol_idx:
      return {"decision": "deny", "reason": "... chờ User APPROVE."}
  ok = any(
      s.get("type") == "USER_INPUT" and APPROVE.search(s.get("content", ""))
      for s in st[(last_sol_idx + 1 if last_sol_idx >= 0 else 0) :]
  )
  ```
- Đoạn code này gây ra **2 tác hại kép**:
  1. Trong lúc tự Audit ở `S5`, vì biết nếu ghi nhận bug vào `audit_report_*.md` và cập nhật `09_tasks_solution_*.md` (`last_sol_idx > u_idx`) thì `st[last_sol_idx + 1 :]` sẽ không có `USER_INPUT` `APPROVE` mới và bị `H4b` chặn cứng không cho sửa code, Agent có xu hướng tuyên bố *"Không phát hiện lỗi mới, không cần mở vòng lặp S5 -> S3"* để được báo Done.
  2. Khi phát hiện ra bug trong chặng Audit, thay vì thực hiện đúng **"Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop)"** (tự cập nhật Plan `S3` + `DG1/DG2` $\rightarrow$ tự sửa code `S4` $\rightarrow$ tự Audit lại `S5` + `DG3` cho đến khi sạch bug rồi mới báo OK), Agent bị `H4b` khóa tay và buộc phải dừng lại bắt User `APPROVE` lần 2!

---

## 2. Giải pháp Kỹ thuật Duy nhất (Simplicity First, Minimal Impact)

### 2.1. Giải pháp cho `RC-1`: Nhúng nguyên văn 100% Prompt Audit của User vào `hooks/agy/agy_guard.py` (`L47-50, 78-105`)
- Khai báo hằng số `VERBATIM_USER_AUDIT_PROMPT` tại `hooks/agy/agy_guard.py:78` chứa **chính xác từng chữ, từng dấu câu** trong prompt của User:
  ```python
  VERBATIM_USER_AUDIT_PROMPT = (
      'Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update.\n'
      "Kiểm tra các xem có suy diễn, báo cáo láo không.\n"
      "Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).\n"
      "Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng\n"
      "Lưu toàn bộ vào file audit_report_*.md trong workspace của task\n"
      "Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."
  )
  ```
- Đưa `VERBATIM_USER_AUDIT_PROMPT` vào đầu `FINAL_AUDIT_MSG` (`H6`) và vào thông điệp chặn của `handle_stop` (`H7`).

### 2.2. Giải pháp cho `RC-2`: Cưỡng chế Vòng Phản Biện Độc Lập tại `H7` (`handle_stop` — `hooks/agy/agy_guard.py:685-785`)
1. **Bắt buộc chạy `git diff` / `git show` thật sau lần sửa code cuối (`last = src[-1][0]`):**
   - Trong khối `if src:` của `handle_stop` (`L718-739`), kiểm tra `touched(st, r"\bgit\b.*\b(diff|show)\b", last)` (khi `run_linter=True` hoặc khi kiểm tra quy trình đầy đủ).
2. **Cưỡng chế Mandatory Adversarial Stop-Bounce ở lần dừng đầu tiên (`n == 0` khi `run_linter=True` và `src` khác rỗng):**
   - Khi Agent vừa hoàn tất lượt code (`src` khác rỗng) và định dừng lại để báo "OK" lần đầu tiên (`n == 0` và `run_linter=True`), **kể cả khi Agent đã viết sẵn `audit_report_*.md` trong lúc đang code**, `handle_stop` sẽ **chặn lại đúng 1 nhịp (`n = 0 -> 1`, trả về `decision: "continue"`)** và bơm **nguyên văn `VERBATIM_USER_AUDIT_PROMPT`** vào phiên!
   - Cơ chế này tái hiện **chính xác 100%** hành vi khi User tự tay paste prompt Audit sau khi Agent vừa code xong: buộc Agent phải thực hiện một lượt rà soát phản biện độc lập trên thành phẩm cuối cùng (`git diff` thật, kiểm tra từng file/dòng, đối chiếu Plan ban đầu, chạy `/laya_router` `DG3`).
   - Nâng giới hạn `stop` counter từ `n >= 2` lên `n >= 3` (`L691`) để đảm bảo:
     - Lần dừng 1 (`n == 0`): Bắt buộc kích hoạt lượt Phản Biện Độc Lập với `VERBATIM_USER_AUDIT_PROMPT`.
     - Trong lượt Phản Biện Độc Lập đó, nếu phát hiện bug/lỗ hổng, Agent tự sửa luôn (`S5 -> S3 -> S4 -> S5`).
     - Lần dừng 2 (`n == 1`): Kiểm tra toàn bộ hồ sơ S5/S6 (`05_progress`, `06_validation`, `audit_report_`, `DG3`, `mutation_gate`, `verify_governance.py`) sau lần sửa code cuối cùng. Nếu đã sạch bug và đủ hồ sơ (`not miss`) thì cho phép dừng (`return {}`); nếu còn thiếu bước nào sau khi vá bug thì nhắc nốt ở `n == 1 -> 2`.

### 2.3. Giải pháp cho `RC-3`: Mở khóa Vòng lặp Tự khắc phục Khép kín (`S5 → S3 → S4 → S5`) không bắt User `APPROVE` lần 2 (`hooks/agy/agy_guard.py:47-50, 577-603`, `GEMINI.md:78`, `skills/qa-agent/SKILL.md:71`)
1. **Tại `hooks/agy/agy_guard.py:577-603` (`H4b`):**
   - Giữ nguyên kiểm tra `if last_aud_idx > last_sol_idx:` để bắt buộc khi phát hiện bug trong `audit_report_<suffix>.md`, Agent **phải quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` trước** (không được sửa mò trực tiếp mà bỏ rơi tài liệu thiết kế).
   - Sửa thông điệp tại `L589-592` và điều kiện `ok` tại `L594-598`:
     - Thay vì kiểm tra `APPROVE` chỉ ở sau `last_sol_idx` (`st[last_sol_idx + 1 :]` — làm mất hiệu lực lệnh `APPROVE` ban đầu của User sau khi cập nhật `09_tasks_solution`), kiểm tra `APPROVE` (hoặc lệnh kích hoạt `Self-Improvement Loop`) ở sau **`first_sol_idx = sol[0] if sol else -1`** (`st[(first_sol_idx + 1 if first_sol_idx >= 0 else 0) :]`).
     - Đồng thời bổ sung cụm `r"|vòng\s+lặp\s+phản\s+tỉnh|self-improvement\s+loop"` vào regex `APPROVE` (`L47-50`).
   - **Kết quả:**
     - Khi chưa có `APPROVE` lần đầu của User (chặng `S1–S3`): `st[first_sol_idx + 1 :]` chưa có `APPROVE` $\rightarrow$ vẫn chặn cứng không cho sửa code (`force_ask`).
     - Khi đã có `APPROVE` của User: nếu ở `S5 Audit` phát hiện bug/lỗ hổng, Agent chỉ cần quay về `S3` cập nhật `09_tasks_solution_<suffix>.md` (`last_sol_idx > last_aud_idx`) + `08_tasks_<suffix>.md` + chạy `DG1/DG2` (`PASS`), rồi **được phép sửa thẳng source code (`S4`), chạy lại test và Audit lại (`S5`) ngay trong cùng lượt thực thi mà KHÔNG dừng lại bắt User `APPROVE` lần 2!**

---

## 3. Blast Radius & Rollback Plan

- **Các file thay đổi:**
  1. `GEMINI.md` (`L78`)
  2. `skills/qa-agent/SKILL.md` (`L71`)
  3. `hooks/agy/agy_guard.py` (`L47-50`, `L78-105`, `L577-603`, `L685-785`)
  4. `hooks/agy/tests/test_agy_guard.py` (cập nhật `test_29` và thêm `test_31` kiểm thử `VERBATIM_USER_AUDIT_PROMPT` + `n == 0` Adversarial Stop-Bounce + `S5 -> S3 -> S4 -> S5` Self-Healing không cần `APPROVE` lần 2).
- **Tương thích ngược (Backward Compatibility):**
  - Toàn bộ 30 unit tests hiện có trong `hooks/agy/tests/test_agy_guard.py` và 20 unit tests trong `tooling/tests/` giữ nguyên tương thích.
- **Rollback Plan:**
  - Tạo git restore-point local trước khi thực thi; nếu cần có thể khôi phục tức thì qua commit restore-point.

---

## 4. Code Demo Chi Tiết

### 4.1. Code Demo tại `hooks/agy/agy_guard.py` (`L47-50` & `L78-105`)
```python
APPROVE = re.compile(
    r"(approve|duyệt|ok\s*(sửa|làm|fix)|làm đi|sửa đi|fix đi|triển khai|thực hiện đi"
    r"|chặn\s*đứng|thêm.*vào.*cho\s*tao|vòng\s+lặp\s+phản\s+tỉnh|self-improvement\s+loop)",
    re.I,
)
...
VERBATIM_USER_AUDIT_PROMPT = (
    'Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update.\n'
    "Kiểm tra các xem có suy diễn, báo cáo láo không.\n"
    "Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).\n"
    "Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng\n"
    "Lưu toàn bộ vào file audit_report_*.md trong workspace của task\n"
    "Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."
)
FINAL_AUDIT_MSG = (
    "🔍 MANDATORY FINAL ADVERSARIAL AUDIT & ANTI-SMOKE-SPECULATION PROTOCOL (Rule #9 S5/S6, #14 G3/G7/G8, #18):\n"
    "Đã phát hiện chỉnh sửa source code ({files}). TRƯỚC KHI báo cáo hoàn tất, BẮT BUỘC tự động thực thi nguyên văn lệnh Audit sau:\n\n"
    + VERBATIM_USER_AUDIT_PROMPT
    + "\n\n"
    "Chi tiết cưỡng chế khép kín (Autonomous Self-Healing Loop `S5 -> S3 -> S4 -> S5`):\n"
    "1. [Line-by-Line Diff & Task Critique]: Chạy `git diff` / `git show` thật, rà soát phản biện từng task trong 08_tasks, từng file & từng dòng git diff.\n"
    "2. [Anti-Smoke-Speculation & Runtime Trace (CẤM Mockup/Smoke Test/Hallucination)]:\n"
    "   - CẤM tạo/dùng mock closure (*Fn), struct literal giả lập, hoặc smoke script rời để suy diễn luồng thật đúng.\n"
    "   - Kiểm tra đủ 3 trục lỗi thực tế: (a) Concurrency/Multi-worker (Race condition, HoL blocking, Debouncer), "
    "(b) Timeout/Heartbeat (AckWait vs InProgress), (c) Data Continuity (Watermark/Cursor qua Outage gap).\n"
    "   - Nếu cần trace runtime value trên môi trường thật mà không tự chạy được -> BẮT BUỘC nêu rõ để phối hợp với User trace runtime value!\n"
    "3. [Anti-Speculation & Honesty Check]: Kiểm tra xem có suy diễn hay báo cáo láo không (mọi khẳng định phải có file:line + output lệnh thật).\n"
    "4. [Self-Improvement & Autonomous Fix Loop (`S5 -> S3 -> S4 -> S5`)]: Đối chiếu lessons/INDEX.md; nếu phát hiện bug/lỗ hổng/anti-pattern mới, ghi vào lessons.md + chạy build_lessons_index.py, quay về cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` + chạy `DG1`/`DG2`, rồi TỰ SỬA DỨT ĐIỂM SOURCE CODE + chạy lại test + Audit lại ngay trong lượt này cho tới khi sạch bóng bug (KHÔNG dừng lại bắt User APPROVE lần 2!).\n"
    "5. [QC, 3-Mutant Gate & /laya_router DG3]: Chạy `mutation_gate.py` (nếu task có `mode: clean-room` — kiểm chứng >=3 bản cố tình làm hỏng đều bị test đánh trượt KILLED) + gọi `laya_gate.py run --stage dg3` (và/hoặc `review`) để nhận xét lại quyết định thực thi.\n"
    "6. [Lưu báo cáo vật lý]: Lưu toàn bộ kết quả 6 mục trên vào `audit_report_<suffix>.md` trong workspace của task (H7 sẽ chặn nếu thiếu!)."
)
```

### 4.2. Code Demo tại `hooks/agy/agy_guard.py` (`H4b` `L577-603` & `H7` `L685-785`)
```python
        sol = [i for i, _, x in calls(st) if "09_tasks_solution" in tgt(x)]
        aud = [
            i
            for i, nm, x in calls(st)
            if nm in ("write_to_file", "replace_file_content") and "audit_report_" in tgt(x)
        ]
        first_sol_idx = sol[0] if sol else -1
        last_sol_idx = sol[-1] if sol else -1
        last_aud_idx = aud[-1] if aud else -1
        if last_aud_idx > last_sol_idx:
            return {
                "decision": "deny",
                "reason": (
                    "Rule 9/13 (S5 -> S3 -> S4 Self-Healing Loop): Đã bước vào chặng S5 Audit (audit_report_). "
                    "Nếu phát hiện bug/gap mới, CẤM sửa mò thẳng source code! Bắt buộc quay về S3 cập nhật "
                    "09_tasks_solution_<suffix>.md + 08_tasks_<suffix>.md và chạy lại DG1/DG2 (PASS), sau đó "
                    "sửa dứt điểm source code ngay trong lượt này (KHÔNG bắt User APPROVE lại)."
                ),
            }
        ok = any(
            s.get("type") == "USER_INPUT" and APPROVE.search(s.get("content", ""))
            for s in st[(first_sol_idx + 1 if first_sol_idx >= 0 else 0) :]
        )
        if not ok:
            return {
                "decision": "force_ask",
                "reason": "Rule 13: chưa thấy User APPROVE sau 09_tasks_solution trong phiên này. Cho phép sửa code?",
            }
```
Và tại `handle_stop` (`H7`):
```python
    if n >= 3:
        return {}
...
    if run_linter and src and n == 0:
        sdir.mkdir(parents=True, exist_ok=True)
        cnt_file.write_text("1", encoding="utf-8")
        bounce_msg = (
            "🛑 [H7 MANDATORY ADVERSARIAL AUDIT PASS — CHẶN BÁO OK SỚM]:\n"
            + VERBATIM_USER_AUDIT_PROMPT
            + "\n\n⚠️ YÊU CẦU BẮT BUỘC TRƯỚC KHI ĐƯỢC PHÉP BÁO OK CHO USER:\n"
            "- Chạy `git diff` / `git show` kiểm tra thực tế từng file & từng dòng vừa sửa (CẤM viết báo cáo từ trí nhớ/tóm tắt).\n"
            "- Nếu phát hiện BẤT KỲ bug, lỗ hổng hay thiếu sót nào so với Plan ban đầu (`09_tasks_solution` / `08_tasks`): "
            "BẮT BUỘC ghi `lessons.md` + chạy `build_lessons_index.py`, cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` + chạy `DG1`/`DG2`, "
            "rồi TỰ SỬA DỨT ĐIỂM SOURCE CODE + chạy lại test + chạy lại `/laya_router` (`dg3`/`review`) và cập nhật `audit_report_<suffix>.md`, `06_validation_<suffix>.md`, `05_progress_*.md` ngay trong lượt này (TUYỆT ĐỐI KHÔNG dừng lại bắt User APPROVE lần 2!)."
        )
        if miss:
            bounce_msg += "\n\nCác mục Done-gate (H7) đang thiếu:\n- " + "\n- ".join(miss)
        return {"decision": "continue", "reason": bounce_msg}

    if not miss:
        return {}

    sdir.mkdir(parents=True, exist_ok=True)
    cnt_file.write_text(str(n + 1), encoding="utf-8")
    reason_msg = "⛔ Done-gate (H7) chưa đạt:\n- " + "\n- ".join(miss)
    if src:
        reason_msg += "\n\n🔍 BẮT BUỘC THỰC THI NGUYÊN VĂN LỆNH AUDIT PHẢN BIỆN:\n" + VERBATIM_USER_AUDIT_PROMPT
    return {"decision": "continue", "reason": reason_msg}
```

---

## 5. Kết quả Kiểm định Decision Gate (`/laya_router` `DG1` & `DG2`)
*(Cập nhật tự động sau khi chạy `laya_gate.py run --stage dg1` và `--stage dg2`)*
