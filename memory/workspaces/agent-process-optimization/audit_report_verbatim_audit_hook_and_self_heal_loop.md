---
tier: T2
suffix: verbatim_audit_hook_and_self_heal_loop
workspace: agent-process-optimization
created_at: 2026-10-20T17:03:10+07:00
---

# BIÊN BẢN AUDIT PHẢN BIỆN CUỐI WORKFLOW (S5 MANDATORY FINAL ADVERSARIAL AUDIT)

## Nguyên văn mệnh lệnh Audit của User đã nhúng vào `hooks/agy/agy_guard.py` (`VERBATIM_USER_AUDIT_PROMPT`)
> «
> Audit lại "quá trình" vừa thực hiện, sử dụng tư duy phản biện cho các task đã làm, các file đã update, từng dòng update.
> Kiểm tra các xem có suy diễn, báo cáo láo không.
> Áp dụng Vòng Lặp Phản Tỉnh & Khắc Phục (Self-Improvement Loop).
> Chạy 1 tiến trình QC gắt gao, soạn context đầy đủ & gọi /laya_router ra nhận xét lại 1 lần xem quyết định làm như vậy là ổn chưa cùng
> Lưu toàn bộ vào file audit_report_*.md trong workspace của task
> Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value.
> »

---

## Mục 1: Phản biện từng dòng update / `git diff` & từng task (`Line-by-Line Diff & Task Critique`)

### 1.1. Phản biện theo từng Task (`08_tasks_verbatim_audit_hook_and_self_heal_loop.md`)
- **`T9.1` (Cập nhật Hiến pháp & Skills):**
  - `GEMINI.md:74-80`: Bổ sung quy định Hook H6/H7 tiêm nguyên văn Prompt Audit của User và quy định `Self-Improvement & Autonomous Self-Healing Loop (S5 → S3 → S4 → S5)` — cấm sửa mò khi chưa cập nhật Plan nhưng đồng thời cấm dừng lại bắt User `APPROVE` lần 2 sau khi đã cập nhật `09_tasks_solution` ở chặng S5.
  - `skills/qa-agent/SKILL.md:71` & `skills/muscle-execute/SKILL.md:73`: Đồng bộ chính xác quy trình `Autonomous Self-Healing Loop (S5 → S3 → S4 → S5)`.
- **`T9.2` (Cập nhật `hooks/agy/agy_guard.py` & `hooks/agy/tests/test_agy_guard.py`):**
  - `hooks/agy/agy_guard.py:47-50`: Bổ sung `vòng\s+lặp\s+phản\s+tỉnh|self-improvement\s+loop|update\s+agy_g` vào regex `APPROVE`.
  - `hooks/agy/agy_guard.py:78-109`: Khai báo hằng số `VERBATIM_USER_AUDIT_PROMPT` giữ nguyên 100% từng chữ của User và ghép trực tiếp vào `FINAL_AUDIT_MSG` (`H6`).
  - `hooks/agy/agy_guard.py:595-623` (`H4b`): Giữ `if last_aud_idx > last_sol_idx: return {"decision": "deny", ...}` để ép cập nhật `09_tasks_solution_<suffix>.md` + `08_tasks_<suffix>.md` trước khi sửa code ở S5, nhưng kiểm tra `APPROVE` sau `first_sol_idx = sol[0] if sol else -1` (`st[(first_sol_idx + 1 if first_sol_idx >= 0 else 0) :]`) để khi Agent cập nhật `09_tasks_solution` trong vòng tự chữa lành `S5 -> S3 -> S4`, Agent được tự động mở khoá sửa dứt điểm source code ngay trong cùng lượt mà không bắt User `APPROVE` lần 2.
  - `hooks/agy/agy_guard.py:711-814` (`H7` `handle_stop`): Tăng ngưỡng `n >= 3` (dòng 711); bổ sung cổng `if run_linter and src and n == 0:` (dòng 787-803) bắt buộc dội ngược (`decision: "continue"`, `cnt_file = "1"`) ở lần dừng đầu tiên kèm `VERBATIM_USER_AUDIT_PROMPT` để buộc Agent thực hiện 1 pass phản biện thực sự trước khi báo OK; đồng thời đính kèm `VERBATIM_USER_AUDIT_PROMPT` vào `reason_msg` mỗi khi `miss` khác rỗng và có sửa source code (`src`).
  - `hooks/agy/tests/test_agy_guard.py:930-962` & `1018-1062`: Cập nhật `test_29` kiểm chứng `S5 -> S3 -> S4` mở khoá ngay sau khi cập nhật `09_tasks_solution` mà không cần `USER_INPUT` `APPROVE` thứ hai; bổ sung `test_31` kiểm chứng `VERBATIM_USER_AUDIT_PROMPT` trong `H6` và `H7 Pass #1` (`n == 0 -> 1`).
- **`T9.3` (Kiểm thử đột biến 3 bản hỏng, `DG3` & Đóng hồ sơ S5/S6):**
  - Đã chạy `mutation_gate.py` (`3/3 KILLED`) và `laya_gate.py run --stage dg3` (`PASS`).

### 1.2. Phản biện từng dòng `git diff` (`Line-by-Line Critique`)
- **Dòng `78-87` (`hooks/agy/agy_guard.py`):** Đối chiếu từng ký tự của `VERBATIM_USER_AUDIT_PROMPT` với câu gốc của User (kể cả dấu ngoặc kép `"quá trình"`, cụm `"Kiểm tra các xem có suy diễn, báo cáo láo không."`, `"xem quyết định làm như vậy là ổn chưa cùng"`, `"Đặc biệt : ko test bằng mockup và Hallucination kết quả, nếu cần phải nói phối hợp với anh trace runtime value."`) -> khớp 100% nguyên văn, không tóm tắt hay cắt chữ.
- **Dòng `601-617` (`hooks/agy/agy_guard.py`):** Kiểm tra edge-case khi `sol` rỗng (`first_sol_idx = -1`) -> `st[0:]` quét từ đầu transcript; khi `sol` có nhiều phần tử (`sol = [10, 25]`, `aud = [20]`, `APPROVE` tại step `12`):
  - Trước khi cập nhật `09_tasks_solution` lần 2 (`sol = [10]`, `aud = [20]`): `last_aud_idx (20) > last_sol_idx (10)` -> `deny` (chặn sửa mò).
  - Sau khi cập nhật `09_tasks_solution` lần 2 (`sol = [10, 25]`, `aud = [20]`): `last_aud_idx (20) < last_sol_idx (25)` và `first_sol_idx = 10`, step `12 > 10` có `APPROVE` -> `ok = True` -> trả về `{"decision": "ask"}` (tự động mở khoá sửa code mà không đòi `APPROVE` lần 2).
- **Dòng `787-803` (`hooks/agy/agy_guard.py`):** Kiểm tra điều kiện `if run_linter and src and n == 0:`:
  - Trong runtime thật của Antigravity (`dispatch("stop", payload)` gọi `handle_stop(payload)` với `run_linter=True` mặc định): khi có `src` và `n == 0`, luôn ghi `cnt_file = "1"` và trả về `decision: "continue"` kèm `VERBATIM_USER_AUDIT_PROMPT` + danh sách `extra_miss` (nếu có).
  - Ở lần dừng thứ 2 (`n == 1`): nhánh `n == 0` không kích hoạt nữa; nếu mọi `miss` đã được giải quyết (`not miss`), trả về `{}` (cho phép hoàn tất lượt trả lời).

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

- **Kiểm tra Zero-Mockup:** Không gắn bất kỳ closure giả lập (`*Fn`) hay mock struct nào vào `hooks/agy/agy_guard.py`.
- **Trace Runtime thật:**
  1. Ngay trong lượt chạy này, khi công cụ `replace_file_content` sửa `hooks/agy/agy_guard.py` và `hooks/agy/tests/test_agy_guard.py`, Hook `H6` (`post-invocation`) của Antigravity đã thực thi thật tiến trình `/Users/trainguyen/Documents/work/agent/hooks/agy/agy_guard.py` và bơm trực tiếp nội dung `VERBATIM_USER_AUDIT_PROMPT` vào luồng hội thoại thật.
  2. Chạy bộ test thật `python3 -m unittest discover -s hooks/agy/tests -p 'test_*.py' -v` (31 tests PASS) và `python3 -m unittest discover -s tooling/tests -p 'test_*.py' -v` (32 tests PASS).
  3. Chạy `mutation_gate.py` tạo 3 đột biến thật trên file vật lý `hooks/agy/agy_guard.py`, chạy subprocess unittest thật, xác nhận cả 3 đột biến đều bị đánh trượt (`KILLED`, returncode `1`) và khôi phục nguyên vẹn SHA-256 (`f6537b9a4e8fcc87370b58d4961aa050dad534a8a3bb63945aa7f8395d5278ed`).

---

## Mục 3: Kiểm tra Concurrency / Multi-worker / Timeout-Heartbeat / Data Continuity

- **Concurrency & State Isolation (`conv_id` + `u_idx`):**
  - Trong `hooks/agy/agy_guard.py:673-675` (`H6`) và `701-703` (`H7`), file trạng thái `.loop` và `.stop` được phân tách theo khoá `f"{conv_id}.{u_idx}.loop"` và `f"{conv_id}.{u_idx}.stop"`, đảm bảo nhiều phiên (`multi-worker` / nhiều cửa sổ hội thoại song song) không ghi đè counter của nhau và mỗi lượt `USER_INPUT` mới (`u_idx` tăng) tự động có counter `n = 0` mới (Data Continuity / Watermark theo `u_idx`).
- **Timeout & Subprocess Safety:**
  - Các lệnh kiểm tra trong `H6` (`timeout=40`) và `verify_governance.py` trong `H7` (`timeout=30`) đều có ngưỡng timeout tường minh, không gây treo vĩnh viễn (`AckWait` / Heartbeat stall) và có chặn trên `if n >= 3: return {}` để chống vòng lặp vô hạn.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (`Anti-Speculation & Honesty Check`)

- Mọi khẳng định trong báo cáo này đều có đối chiếu `file:line` thật và output lệnh thực tế:
  - `hooks/agy/agy_guard.py:47-50` (`APPROVE`), `78-109` (`VERBATIM_USER_AUDIT_PROMPT`, `FINAL_AUDIT_MSG`), `595-623` (`H4b`), `705-814` (`H7`).
  - `hooks/agy/tests/test_agy_guard.py:930-962` (`test_29`), `1018-1062` (`test_31`).
  - `06_mutation_summary_verbatim_audit_hook_and_self_heal_loop_20261020T170034.json` (`3/3 KILLED`).
  - `06_laya_summary_dg3_20261020T170256.json` (`gate: PASS`).
- Không có bất kỳ chi tiết nào suy diễn hay báo cáo láo.

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (`Self-Improvement Loop`)

- **Bài học đã ghi nhận (`Rule #5` & `Rule #6`):**
  - Đã ghi **Lesson `L257` (`[2026-10-20] Paraphrased Verbatim Audit Prompt & Interrupted S5->S3 Self-Healing Loop`)** vào `/Users/trainguyen/Documents/work/agent/memory/global/lessons.md` (`1993 -> 2000` dòng) và đồng bộ `/Users/trainguyen/Documents/work/agent/memory/global/lessons/INDEX.md` (`257/257/257`, `14,329B`).
- **Kiểm tra vòng lặp tự chữa lành (`S5 -> S3 -> S4 -> S5`):**
  - Trong quá trình chạy `T9.3`, không phát sinh lỗi logic trong source code `hooks/agy/agy_guard.py` (63/63 unit tests và 3/3 mutants đều PASS ngay từ lần đầu).

---

## Mục 6: Kết quả QC, 3-Mutant Gate & nhận xét từ `/laya_router` (`DG3`)

- **Kết quả 3-Mutant Gate (`mutation_gate.py`):**
  - File biên bản: `06_mutation_summary_verbatim_audit_hook_and_self_heal_loop_20261020T170034.json`
  - Kết quả: `gate: "PASS"`, `total_mutants: 3`, `killed_count: 3`, `survived_count: 0`, `all_restored_sha256: true`.
    1. `M1_drop_verbatim_prompt_in_h6`: `KILLED`
    2. `M2_break_self_healing_approve_check`: `KILLED`
    3. `M3_disable_h7_first_stop_bounce`: `KILLED`
- **Kết quả `/laya_router` (`DG3`):**
  - Lệnh: `python3 /Users/trainguyen/Documents/work/agent/tooling/laya_gate.py run --stage dg3 --workspace agent-process-optimization --claims .../06_laya_claims_dg3_verbatim_audit_hook_and_self_heal_loop.json`
  - File biên bản: `06_laya_summary_dg3_20261020T170256.json`
  - Kết quả: `gate: "PASS"` (100% `REF:` khớp dòng thực tế trên đĩa; các claim `DG3-VERBATIM-PROMPT`, `DG3-SELF-HEALING-H4B`, `DG3-H7-STOP-BOUNCE` đều đạt `CORRECT`/`YES` ở cả 3 vòng `r1`, `r2`, `r3` với độ tự tin vòng 3 lần lượt `0.8783`, `0.7629`, `0.9546`; do calibration control `UNVALIDATED` trên model local nên kết luận cuối cùng xác lập bằng ground truth code và test thực tế theo Rule #10).
