# 06 — Validation `rea_cleanroom_gap` (2026-10-21)

## 1. Test suites (lệnh thật, output thật)
| Lệnh | Kết quả |
|---|---|
| `cd hooks/agy && python3 -m unittest discover -s tests -p 'test_*.py'` | `Ran 47 tests` — `OK` (31 cũ + 16 mới trong `test_tripwire_cleanroom.py`) |
| `cd tooling && python3 -m unittest discover -s tests -p 'test_*.py'` | `Ran 37 tests` — `OK` (32 cũ + 5 mới `test_tripwires_index.py`; `test_verify_governance` thêm case G-03; `test_harness2_engine` thêm case G-05) |
| Baseline trước khi sửa (restore-point `296e77a`) | hooks `Ran 31 OK`, tooling `Ran 32 OK` |

## 2. Red → Green (G2)
- Hooks: lần chạy đầu `test_tripwire_cleanroom.py` → 5 FAIL + 8 ERROR (chưa có code) → Green sau S4.
- Root cause lỗi phát sinh trong S4: `SPEC_RULE_RE` dùng `\s*` nuốt newline (repro: `''` vs `'- R1: a — REF: x:1'`) → đổi `[ \t]*`.
- AF-1/AF-2 (S5): 2 test hồi quy FAIL trước fix → PASS sau fix.

## 3. 3-Mutant Gate (G3)
`python3 tooling/mutation_gate.py --workspace audit-rea-cleanroom-harness2-2026-10-21 --suffix rea_cleanroom_gap --mutations .../06_mutations_rea_cleanroom_gap.json`
→ `gate=PASS`, `killed 8/8`, `all_restored_sha256=True` (`06_mutation_summary_rea_cleanroom_gap_20261021T095905.json`). M1..M8 phủ G-01, G-02, G-03, G-05, G-08, AF-1, AF-2.
Lần chạy trước đó FAIL baseline do `skills/reverse-engineer-anything` xuất hiện ngoài phiên (AF-3) → đã đăng ký trong `OPERATOR_MAP.md`, chạy lại PASS.

## 4. Runtime thật
- **H1 qua đúng đường gọi của IDE** (`echo payload | python3 hooks/agy/agy_guard.py pre-invocation`, transcript chứa `github.com/google-research/harness2`): ephemeralMessage có mục `🧷 LESSON TRIPWIRES` với `LS-20261021-prior-workspace-lookup` + `LS-20261020-readme-only`.
- **Replay transcript thật của phiên này** (`transcript_full.jsonl`, 31 step trước lần ghi `13_analysis` đầu tiên): `tripwire_violations` = `['LS-20261021-prior-workspace-lookup', 'LS-20261020-readme-only']` ⇒ gate mới chặn đúng lỗi lượt 1.
- `build_lessons_index.py` thật: `entries=262/262/262`, `tripwires.json` 2 mục.
- `harness2_engine.py validate-surface` thật: `VALIDATED`, `errors=[]`.

## 5. Chưa kiểm được trên môi trường thật (cần User phối hợp)
- H4t/H7 G-04 chạy trong IDE thật khi gặp clean-room task thật (chưa có task clean-room thật nào trong phiên).
- Hook có chạy trong trajectory của subagent không (G-04 HYPOTHESIS cũ) — chưa trace.
- `run-recursion` với CLI agent thật (opencode/codex/claude) — chưa chạy; hiện chỉ đảm bảo không còn stub và thiếu lệnh → exit 2.

## 6. Vòng 2 (H7 stop-audit, 10:02–10:07)
- Replay hợp lệ: `transcript_full.jsonl` 300 step, `calls()` parse 150 tool call.
- AF-4/AF-5 repro trên transcript thật (step 102, step 244) → fix → hooks `Ran 49 OK`, tooling `Ran 37 OK`.
- `mutation_gate.py` → `PASS 10/10 KILLED`, `all_restored_sha256=True` (`06_mutation_summary_rea_cleanroom_gap_20261021T100546.json`).
- Replay sau fix: turn1 `['LS-20261021-prior-workspace-lookup','LS-20261020-readme-only']`; toàn phiên trước khi đọc tài liệu cũ `['LS-20261021-prior-workspace-lookup']` (đúng sự thật) → sau `view_file` audit_report workspace harness2 cũ → thoả.
