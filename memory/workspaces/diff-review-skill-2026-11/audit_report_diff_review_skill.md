# Audit Report — `diff_review_skill` (tier: T2, hợp nhất validation + report + 6 mục audit)

- **Ngày:** 2026-11-17 | **Agent:** Claude Fable 5.1 (OpenCode) | **Workspace:** `diff-review-skill-2026-11`
- **Bối cảnh:** Session S4 đầu (13:39–13:47) viết xong T2 rồi đứt `ECONNRESET`. Session này resume 14:39, hoàn tất T1, T3–T8. Không ghi đè file nào của session trước.

## 1. Đối chiếu `01_requirements` ↔ kết quả (G1)
| R | Yêu cầu | File vật lý | Kích thước | Trạng thái |
| :--- | :--- | :--- | :---: | :---: |
| R-1 | `SKILL.md` pipeline R0→R6, 3 lớp luật, severity, verdict, giới hạn vòng, sub-agent, output | `skills/diff-review/SKILL.md` | 7.0KB | ✅ |
| R-2 | Base Rules L1 × 4 lens | `rules/review/base/{security,logic,design,contract}.md` (session trước, 13:47) | 3.9–4.6KB | ✅ |
| R-3 | Fintech Pack L2 | `rules/review/packs/fintech.md` | 5.1KB | ✅ |
| R-4 | `scope.py` 0-token, đọc `.agent-project.json.review` | `tooling/diff_review/scope.py` (+ `selftest.sh`) | 10.3KB | ✅ |
| R-5 | Template báo cáo | `tooling/diff_review/report_template.md` | 2.1KB | ✅ |
| R-6 | Profile data-hub ≤8KB + template | `memory/global/project/data-hub/review_profile.md` (**7962 B**), `rules/review/profile_template.md` | 7.9KB / 2.0KB | ✅ |
| R-7 | Đăng ký skill | `skills.json` là folder registry → không cần sửa (VERIFIED `09` §6); `OPERATOR_MAP.md` +1 dòng (`git diff --stat`: `1 insertion(+)`) | — | ✅ |

## 2. Bằng chứng thực thi (G3, G8) — biên lai `evidence_run.py`
| EV | Kind | Lệnh | exit |
| :--- | :--- | :--- | :---: |
| `EV-20261117T145831-fa8e` | unit | `python3 -m py_compile tooling/diff_review/scope.py` | 0 |
| `EV-20261117T145831-2675` | unit | `bash tooling/diff_review/selftest.sh` → repo git tạm, **có manifest**: `files=3 cps=1 god=2 money=1 viol=1`; **không manifest**: `files=3 cps=0 god=0 money=0 viol=0`; 9 assert PASS | 0 |
| `EV-20261117T145904-4a09` | runtime | `scope.py --repo centralized-data-service --base HEAD~1 --head HEAD` (repo thật, 53 file, +4988/−1254, không manifest) → `scope_smoke_cds.json`, `layers=[]` đúng kỳ vọng | 0 |

## 3. Decision Gate (G7 — DG3)
- `claims_dg3_diff_review_skill.json`: 7 claim + 1 `control_true` + 2 `control_false` (đạt calibration `GEMINI.md:82`).
- `06_laya_summary_dg3_20261117T150131.json`: **`gate=PASS`**, 7/7 `evidence_error=''` (REF `file:lines` khớp thật trên đĩa), `trusted_rounds=[]` → 7 claim `UNVALIDATED`: engine không phân biệt được control → **kết luận bằng code** (`GEMINI.md:83`), đối chiếu tay mục 4.

## 4. Adversarial line-by-line (G7a) — kết luận bằng code
| Claim | Đối chiếu tay | Kết quả |
| :--- | :--- | :---: |
| 3 lớp luật | `SKILL.md:30-39` khối `L1/L2/L3` + "Pack/Profile chỉ nâng severity" | ✅ |
| Giới hạn vòng | `SKILL.md:66`: R1–R4 đúng 1 lần, Mode B `--max-rounds 3`, BLOCK → kiểm 1 lần → drop | ✅ |
| F-LOG-01 S0 cấm float | `fintech.md:20` | ✅ |
| scope.py agnostic | `scope.py:190` `money_rx = ... if keywords else None`; `grep -rn "/Users/"` trên 7 file mới = 0 hit (ADR-07, TC-1, TC-4) | ✅ |
| layer_violations | `scope.py:164` `rank[lname] < rank[f["layer"]]`; selftest bắt đúng `repository → handler` | ✅ |
| Profile CP-1 | `review_profile.md:22` khớp `system_overview…md:130` (ResolveSourceRoutes `:469`, cấm `routes[0]`) | ✅ |
| OPERATOR_MAP | `OPERATOR_MAP.md:19` dòng `diff-review` | ✅ |

## 5. Edge-case / blast-radius (G4, G5)
- **Rename/copy** (`R`, `C` trong name-status): parse lấy path mới; numstat `{a => b}` xử lý bằng regex. Chưa có test cho rename → ghi nhận **HYPOTHESIS**, selftest chỉ cover A/M.
- **Binary file**: `numstat` trả `-` → `binary=true`, bỏ qua hunk text (không crash).
- **File xoá (`D`)**: `loc_after=0`, không đọc `git show`.
- **Không manifest**: mọi cờ rỗng, exit 0 (EV `-2675`, `-4a09`).
- **Manifest lỗi JSON / không tồn tại**: `die()` exit 3 (`scope.py:54-66`).
- **Blast-radius core:** chỉ thêm file mới + 1 dòng `OPERATOR_MAP.md`; không sửa `laya_gate.py`, `evidence_run.py`, `verify_governance.py`, hook — đúng Out-of-Scope `01` §3.
- **Repo dự án:** 0 file source bị động (`git status` 3 repo data-hub không có thay đổi từ task này).

## 5b. Concurrency / Timeout / Data Continuity (G4)
- `scope.py` là CLI chạy 1 lần, không có worker/goroutine/multi-worker, không giữ state giữa các lần chạy → không có bài toán heartbeat/timeout/watermark continuity. Mỗi lần chạy tạo `scope_<suffix>.json` mới; file cũ không bị ghi đè trừ khi cùng `--out`.
- `SKILL.md` §5: khi chạy lens song song bằng sub-agent, mỗi sub-agent nhận tập hunk **disjoint theo lens**, kết quả gộp ở R6 theo `REF` → không có race ghi chung file; chỉ Brain ghi `review_report`.
- `selftest.sh` dùng `mktemp -d` riêng mỗi lần → 2 lần chạy đồng thời không đụng nhau.

## 5c. Chống suy diễn & báo cáo láo (honesty check)
- Mọi `PASS`/`exit=0` trong mục 2 đều có biên lai `EV-` thật từ `evidence_run.py`; không có khẳng định nào dựa vào smoke test in-memory.
- 7 claim DG3 ở trạng thái `UNVALIDATED` được **ghi rõ** là engine không phân biệt được (không báo "PASS" khống), kết luận bằng đối chiếu tay mục 4 với `file:line`.
- Điểm chưa test (rename/copy) ghi rõ là HYPOTHESIS ở mục 5, không gộp vào "đã verify".
- `.agent-project.json` khoá `review` **chưa tồn tại** ở 3 repo data-hub → skill chưa pilot thật; không claim "đã chạy end-to-end trên dự án".

## 6. Security (G7 — `/security-agent` tự quét)
- `grep -rnE "(sk-[A-Za-z0-9]{10,}|password\s*[:=]|api[_-]?key\s*[:=]|/Users/)"` trên 7 file mới → **0 hit** (pycache đã xoá). `review_profile.md` chỉ chứa tên bảng/symbol, không có credential/PII.
- `selftest.sh` dùng `mktemp -d` + `trap rm -rf` → không để lại repo tạm.

## 7. Còn lại / Đề xuất (Propose-Only, chưa làm)
- **P5** (ngoài scope `01` §3): hook "không có `review_report_*` → không DONE" — làm sau pilot.
- Pilot thật: tạo `.agent-project.json` khoá `review` cho 3 repo data-hub (hiện **chưa có** file này ở cả 3 repo — VERIFIED `cat` 14:55) rồi chạy `/diff-review` trên 1 nhánh feat để đo thời gian/tokens.
- Thêm case rename vào `selftest.sh`.

## 8. Lessons
- **Áp dụng:** `GEMINI.md:15` (mọi REF đối chiếu file thật), `:29` ADR-07, `:44` append-only `05`, `:83` UNVALIDATED → kết luận bằng code, `:110` không overwrite memory; L047 (Pure Agnostic Core), L059 (Zero-Mockup Evidence — chạy trên repo thật), L063.
- **Mắc phải phiên trước (đã ghi lesson 13:31 "Halt On Environment Read Denial"):** không dừng khi `Operation not permitted`, đóng gói HYPOTHESIS như VERIFIED.

## 9. Pilot thật (sau APPROVE 15:13) — `review_report_pilot_cds.md`
- Tạo 3 manifest **mới** `.agent-project.json` (khoá `components` + `review`) cho `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web` (file chưa từng tồn tại — VERIFIED `find`); JSON valid.
- R0 trên `ab37d47..e82ed6e` (27 file): bắt đúng `CP-4` qua path + symbol `TriggeredBy`, 5 god-files, 1 `forbidden_ifelse_touched`, 0 layer violation — `EV-20261117T151447-12a7`.
- R5 Mode B `laya_gate.py review` 3 file CP-4: `gate=PASS` (`06_laya_summary_dg3review_20261117T151644.json`).
- Lens R2/R3/R4 tay: verdict **APPROVE** (0 S0/S1, 2 S2 nợ cũ). `go build` + `go test ./internal/handler/master/` PASS (`EV-…-7dce`, `EV-…-9992`). 1 test fail `TestResolveSourceRoutes_GenericConnectionFallback` **có sẵn ở base** (verify bằng `git worktree` tại `ab37d47`) → không phải regression (`INV-5`).
- Phát hiện tooling: `evidence_run.py` không giữ `|` trong tham số `-run 'A|B'` → phải tách lệnh (ghi `review_report_pilot_cds.md` §6). Đề xuất, chưa sửa (ngoài scope `01` §3).
