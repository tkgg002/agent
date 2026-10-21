# 13 — Analysis: `/agent` đã đảm bảo REA Clean-Room & Harness² chưa? (`rea_cleanroom_gap`)

type: Audit (Propose-Only) | tier: T2 | date: 2026-10-21

## 1. Kết luận nhanh
| Hạng mục | Trạng thái | Ghi chú |
|---|---|---|
| Clean-Room (REA → Jev → builder mới → 3 mutant → loop) | 🟡 **CÓ KHUNG, CHƯA KÍN** | Có hook/gate thật, nhưng 2 lỗ hổng P0 đã tái hiện runtime |
| Harness² (google-research) | 🟡 **CÓ CODE, CHƯA ĐƯỢC NỐI VÀO WORKFLOW** | `harness2_engine.py` + test tồn tại nhưng không file điều phối nào gọi tới |

## 2. Ma trận độ phủ Clean-Room (từng bước của User)
| Bước User | Thành phần `/agent` tương ứng | Bằng chứng | Đánh giá |
|---|---|---|---|
| 1. REA điều tra xuống binary | Không có. `rea` chưa cài (`which rea` → not found; `npm ls -g rea-agents` → empty) | runtime 2026-10-21 09:28 | ❌ Chỉ điều tra source bằng `view_file/grep`. Đủ cho repo có source; thiếu cho binary/APK/Electron |
| 2. SPEC.md, mỗi rule có evidence | Mục `## Clean-Room Approved Spec` trong `09_tasks_solution_*` | `hooks/agy/agy_guard.py:546-553`, `tooling/verify_governance.py:217-224` | 🟡 Chỉ regex kiểm **tiêu đề** tồn tại (`CLEAN_ROOM_SPEC_RE`, `agy_guard.py:56`), **không** kiểm từng rule có `REF:` |
| 3. Jev duyệt, trả rule thiếu bằng chứng | Laya `DG1` Mode A verify `REF: path:start-end` khớp đĩa + ≥2 control_false, ≥1 control_true | `tooling/laya_gate.py:109-137`, `:294` | 🟡 Verify REF thật, nhưng **không liên kết** rule-ID trong Spec ↔ claim DG1. Rule bị BLOCK không tự bị loại khỏi Spec |
| 4. Builder mới tinh, không thấy code gốc | H4b chặn ghi code nếu phiên đã đọc `clean_room_forbidden_sources` | `agy_guard.py:583-594`, `:300-319` | 🔴 **G-02**: chỉ soi `view_file/grep_search/find_by_name`; đọc qua `run_command` (`cat/sed/git show`) **lọt** |
| 5. Test + 3 bản cố tình hỏng | `mutation_gate.py` (≥3 mutant, baseline PASS, khôi phục SHA-256) + chặn đóng phiên | `tooling/mutation_gate.py:1-12,29`; `agy_guard.py:322-350,754-760`; `verify_governance.py:228-255` | ✅ Tốt. (Không có Haiku trong enum model; dùng `flash` thay) |
| 6. Case unclear → Opus | Laya `FLAG/UNVALIDATED` → `resolve` kèm REF | `laya_gate.py:364-368`, GEMINI Rule #10 | 🟡 Có cơ chế, không route model |
| 7. /goal lặp tới khi rebuild PASS & 3 mutant FAIL | Vòng `S5→S3→S4→S5` + H7 chặn Done khi mutation ≠ PASS | `agy_guard.py:754-760` | ✅ |
| 8. Hiện SPEC trước, chỉ build khi "BẮT ĐẦU" | Regex `APPROVE` mở khoá sửa code | `agy_guard.py:47-50`, `:615-621` | 🔴 **G-01**: (a) không có "BẮT ĐẦU"; (b) false-positive: chữ "duyệt" bất kỳ đâu trong tin nhắn đều mở khoá |

## 3. Findings
### G-01 [P0][VERIFIED] Cổng APPROVE quá lỏng & thiếu "BẮT ĐẦU"
- REF `hooks/agy/agy_guard.py:47-50`: regex `(approve|duyệt|...|triển khai|...)` không neo đầu/cuối câu.
- Runtime: chính tin nhắn User hôm nay ("...SPEC.md đã được Jev duyệt...") → `APPROVE.search` = **True**; `"BẮT ĐẦU"` → **False**.
- Hệ quả: một tin nhắn mô tả quy trình có thể mở khoá S4 trước khi User thật sự duyệt SPEC.

### G-02 [P0][VERIFIED] Rò rỉ Clean-Room qua `run_command`
- REF `hooks/agy/agy_guard.py:300-319`: `contaminated_clean_room_reads` chỉ xét `view_file`, `grep_search`, `find_by_name`.
- Runtime: trajectory `run_command: cat /repo/legacy/payment.go` → `[]` (không phát hiện); `view_file` cùng file → `['legacy/payment.go']`.
- Hệ quả: "model thấy code gốc vẫn viết code" — vi phạm đúng câu Clean-Room rule.

### G-03 [P1][VERIFIED] Spec không được kiểm từng rule ↔ evidence
- REF `agy_guard.py:56` + `verify_governance.py:217`: chỉ tìm chuỗi `clean-room approved spec`.
- Một Spec có tiêu đề nhưng rule không REF (hoặc rule bị DG1 BLOCK) vẫn qua gate → mất vai trò "Jev trả lại rule".

### G-04 [P1][HYPOTHESIS] Prompt subagent builder không bị kiểm nhiễm
- `invoke_subagent` không bị hook soi nội dung Prompt; Brain có thể dán code gốc vào prompt. Subagent `Workspace: inherit` vẫn đọc được repo gốc; việc hook H4b có chạy trong trajectory của subagent hay không **chưa trace được** → cần phối hợp User trace 1 lần thật.

### G-05 [P1][VERIFIED] Harness² là "thư viện mồ côi"
- `tooling/orchestrator/harness2_engine.py` (F1–F6) + `tooling/tests/test_harness2_engine.py` chạy `unittest` → OK; workspace `feature-harness2-recursive-integration-2026-10-20` đã DONE.
- Nhưng `grep -rlI harness2 GEMINI.md OPERATOR_MAP.md skills hooks scripts workflows rules tooling/*.py` → **không kết quả** (exit=1). Không skill/hook/workflow nào kích hoạt Propose–Probe–Compose.
- Phần "improvement harness / evidence-supported lessons" của Harness² hiện được thay thế tương đương bởi vòng lessons (Rule #5) — đạt về tinh thần, không phải bản tích hợp engine.

### G-06 [P2][VERIFIED] REA chưa kết nối
- Không có MCP/CLI REA. Chỉ cần khi feature nằm trong binary/APK/Electron không có source.

## 4. Phương án tối ưu duy nhất (chờ APPROVE)
**"Clean-Room Hardening + Harness² wiring" trong core agnostic (không hardcode dự án):**
1. **G-01** — Tách cổng clean-room: khi `mode: clean-room`, chỉ mở S4 nếu tin nhắn User **khớp nguyên dòng** `^\s*(BẮT ĐẦU|APPROVE)\s*$` (neo, không search tự do). Giữ regex cũ cho task thường nhưng thêm neo từ khoá ở đầu câu.
   ```python
   CLEAN_ROOM_START = re.compile(r"^\s*(BẮT\s*ĐẦU|APPROVE)\s*[.!]?\s*$", re.I | re.M)
   ```
2. **G-02** — `contaminated_clean_room_reads` xét thêm `run_command.CommandLine`, `read_url_content.Url`, `search_web`, nếu chuỗi chứa path forbidden.
3. **G-03** — Định dạng Spec bắt buộc `R<n>: <hành vi> — REF: path:start-end`; `verify_governance.py` + H4 parse từng `R<n>`, từ chối rule thiếu REF hoặc có ID nằm trong DG1 BLOCK/FLAG chưa resolve.
4. **G-04** — H4b soi `invoke_subagent.Prompt`: khi clean-room, Prompt chỉ được chứa Spec (cấm path forbidden & cấm khối code > N dòng không thuộc Spec); khuyến nghị `Workspace: branch` loại bỏ forbidden sources.
5. **G-05** — Thêm skill `clean-room-rebuild` (hoặc mục trong `muscle-execute`) gọi `harness2_engine.py` ở chặng S4 với `--k 1` cho builder; khai báo trigger trong `OPERATOR_MAP.md`.
6. **G-06** — Tuỳ chọn: hướng dẫn `npx rea-agents setup` trong skill khi target là binary; không cài mặc định.
7. Test Red→Green cho G-01..G-04 trong `hooks/agy/tests/test_agy_guard.py`, chạy `mutation_gate.py` cho chính các gate mới.

## 5. Decision Gate
Xem `05_progress_rea_cleanroom_gap.md` (kết quả DG3).

## 6. Đính chính sau khi User chất vấn (2026-10-21T09:37)
- Lượt đầu chỉ đọc README harness2 → vi phạm lesson README-only. Nay đã đối chiếu source scratch `223ce70e.../scratch/harness2/harness2/core/{ledger,tree}.py` (khớp mô tả của User: `Ledger.assert_identity` R=4k/P=3k tại `ledger.py:89-113`; `always_on_max` tại `tree.py:46,197,347`).
- **G-05 nâng cấp [P0][VERIFIED] — "Propose" trong `/agent` là stub viết cứng:** `tooling/orchestrator/harness2_engine.py:1020-1047` — CLI `run-recursion` truyền proposer là one-liner Python ghi `VALUE = 2` + `skills/dom-<lbl>/SKILL.md` giả, và rollout scorer trả `1.0/0.6/0.2` theo việc 2 file đó tồn tại. Không gọi model/CLI agent nào. Đồng thời dùng điểm số làm tiêu chí chọn — trái tinh thần harness2 "không ground-truth feedback". Khi không truyền template, `run_proposer` (`:587`) trả lại base tree nguyên vẹn.
- `validate-surface` chạy thật: `always_on_chars=18313`, `always_on_max=24000` (harness2 gốc theo mô tả User = 12000), `errors=[]`.
- Chưa thấy `memory/global/harness_library/` (Domain Library chưa từng được ghi bởi run thật).
- Phạm vi chưa đọc: `recursion.py`, `proposer.py`, `selection.py`, `library.py`, `context.py`, `prompts.py` của harness2 gốc — chưa đối chiếu dòng-dòng.

## 7. G-08 [P0][VERIFIED] Cơ chế lesson chỉ "ghi biên bản", không phòng ngừa (User chỉ ra 09:42)
Lesson "Reading Only README.md of Reference Repo" đã tồn tại từ 2026-10-20 nhưng không chặn được lỗi tái phạm hôm nay. Nguyên nhân:
1. **INDEX vô dụng để tra:** `tooling/build_lessons_index.py:93-101` `concise_label` cắt nhãn còn ~15 ký tự (`Reading Only RE…`) — không đọc ra nội dung/trigger.
2. **ID theo vị trí, trôi mỗi lần chèn:** lesson README-only là `L006` lúc boot, sau khi chèn lesson mới thì `L006` thành `Post-Compaction…` (`INDEX.md:27`) — không thể tham chiếu ổn định.
3. **Nội dung thật không ai đọc:** topic files 21–207KB (`01_governance` 207KB, 84 lesson); Rule #21 cấm đọc file lớn ⇒ tại S0 chỉ đọc INDEX rỗng nghĩa.
4. **Hook không truy xuất theo task:** `hooks/agy/agy_guard.py:70-74` H1 chỉ tiêm câu "Đọc INDEX.md"; không match prompt User với lesson liên quan, không có cổng chặn hành động trùng anti-pattern đã biết.
5. **Định dạng lesson là tường thuật hậu kiểm** (đoạn văn dài), không có `trigger` máy đọc được và `check` tiền điều kiện.

