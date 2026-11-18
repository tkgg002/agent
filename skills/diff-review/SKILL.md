---
name: diff-review
description: >-
  Propose-Only code review of a git diff through 4 lenses (Security, Logic,
  Design Pattern, Contract) using a 3-layer rule stack (L1 Base -> L2 Domain
  Pack -> L3 Project Profile) selected by <repo>/.agent-project.json. R0 scope is
  computed by tooling/diff_review/scope.py with 0 LLM tokens; findings require
  REF: path:line; verdict BLOCK / REQUEST_CHANGES / APPROVE; validated through
  laya_gate.py (Mode B review + Mode A dg3). Runs inside S5 or as a standalone Audit.
---

# Diff Review Skill (4 Lens × 3-Layer Rules, Propose-Only)

> **Quy tắc cốt lõi:** Rule #1A Propose-Only (`GEMINI.md:19`), ADR-07 Agnostic Core (`GEMINI.md:29`), S5 Adversarial Audit (`GEMINI.md:76`), Decision Gate (`GEMINI.md:80-83`), Memory Protection (`GEMINI.md:110`).
> **Nguyên tắc:** Finding không có `REF: <path>:<start>-<end>` + trích code → **không được tính**. Skill này chỉ đọc & báo cáo, không sửa bất kỳ file source nào.

## 1. Khi nào dùng
- User yêu cầu *"review diff / review PR / review nhánh X so với Y"*, *"check code trước merge"*.
- Tại **S5** của một task T2/T3 sau lần sửa code cuối (thay cho bước (a) + (e) của `GEMINI.md:76`).
- Audit độc lập một nhánh/commit range chưa có workspace → tạo workspace `review-<repo>-<ref>-<date>` (`tier: Audit`).

## 2. Input
| Tham số | Bắt buộc | Ghi chú |
| :--- | :---: | :--- |
| `--repo <path>` | ✔ | Git repo cần review |
| `--base <ref>` / `--head <ref>` | ✔ | Ví dụ `origin/develop` / `origin/feat/X`; dùng `base..head` |
| `<repo>/.agent-project.json` khoá `review` | ✘ | Không có → chỉ nạp L1. Schema tại mục 4 |
| `--workspace <ws>` | ✔ | Nơi lưu `scope_<suffix>.json`, `review_report_<suffix>.md`, biên lai `06_evidence/` |

## 3. Cách nạp luật 3 lớp (L1 → L2 → L3)
```
L1 BASE     agent/rules/review/base/{security,logic,design,contract}.md   luôn nạp, mã B-SEC/B-LOG/B-DES/B-CON-xx
L2 PACK     agent/rules/review/packs/<pack>.md                            nạp theo review.packs[], mã F-xx (fintech)
L3 PROFILE  <đường dẫn trong review.profile>                               1 file ≤ 8KB: layer, chokepoints, contracts, god-files
```
- Pack/Profile chỉ được **nâng** severity của mục L1, không hạ.
- Mỗi lens chỉ nhận checklist của lens đó + mục liên quan trong Pack/Profile (không nạp cả 3 lớp cho mọi lens → tiết kiệm token).
- Không có `review_profile` → Design lens đánh dấu mọi finding layer/chokepoint là `HYPOTHESIS`.

## 4. Schema `.agent-project.json` khoá `review` (backward compatible, mọi trường optional)
```json
"review": {
  "packs": ["fintech"],
  "profile": "memory/global/project/<name>/review_profile.md",
  "god_file_loc": 600,
  "layer_order": ["server", "handler", "service", "repository", "model"],
  "layers": { "handler": "internal/handler/", "service": "internal/service/" },
  "chokepoints": [ { "id": "CP-1", "paths": ["internal/service/x.go"], "symbols": ["ResolveX"] } ],
  "money_path_keywords": ["amount", "balance", "ledger"],
  "forbidden_ifelse_files": ["internal/service/master/transmuter.go"]
}
```
`layer_order` xếp từ ngoài vào trong; import từ tầng trong ra tầng ngoài = `layer_violation`.

## 5. Pipeline R0 → R6
| Bước | Ai | Token | Việc | Output |
| :--- | :--- | :---: | :--- | :--- |
| **R0 SCOPE** | `python3 $T/diff_review/scope.py --repo <r> --base <b> --head <h> [--manifest <m>] [--pack <p>] --out <ws>/scope_<suffix>.json` | 0 | name-status + numstat; gắn `layer`; cờ `god_files`, `chokepoints_touched`, `money_path`, `layer_violations` | `scope_<suffix>.json` |
| **R1 SECURITY** | lens (sub-agent) | model | `base/security.md` + mục Security của Pack/Profile; chạy lệnh grep 0-token trong file luật | findings `B-SEC-*`, `F-SEC-*` |
| **R2 LOGIC** | lens | model | `base/logic.md` + Pack/Profile; bắt buộc đọc **callers** của hàm bị đổi chữ ký (grep) | findings `B-LOG-*` |
| **R3 DESIGN** | lens | model | `base/design.md` + `layer_violations` + `god_files` + `forbidden_ifelse_files` từ scope | findings `B-DES-*` |
| **R4 CONTRACT** | lens | model | `base/contract.md` + `chokepoints_touched` + contracts trong Profile (event/table/API) | findings `B-CON-*` |
| **R5 GATE** | `laya_gate.py` | 0 | Mode B: `review --workspace <ws> --repo <r> --max-rounds 3 --objective "<...>"` (engine tự thêm corrupted-diff control). Mode A: `run --stage dg3 --workspace <ws> --claims <claims.json>` với mỗi finding = 1 claim `kind: claim`, `evidence: "REF: <abs>:<s>-<e>\n<quoted>"` + `≥2 control_false` + `≥1 control_true` | `06_laya_summary_*.json` |
| **R6 REPORT** | Brain | model | Gộp, khử trùng theo `REF`, xếp severity, tính verdict theo `tooling/diff_review/report_template.md` | `review_report_<suffix>.md` |

**Giới hạn vòng:** R1–R4 chạy đúng **1 lần** (song song qua sub-agent nếu diff > 10 file, tuần tự nếu ≤ 10). R5 Mode B tối đa `--max-rounds 3`. Finding bị `BLOCK` ở R5 → kiểm tra lại REF 1 lần, vẫn sai → loại bỏ, ghi vào mục "Dropped" của report. Không lặp R1–R4 lần 2.

**Sub-agent song song:** mỗi lens nhận đúng: (1) hunks của các file trong scope liên quan lens; (2) checklist 1 lens; (3) phần Profile liên quan. Sub-agent trả JSON `[{id, severity, ref, quote, why, fix_hint}]`. Brain không tin finding thiếu `ref`.

## 6. Severity & Verdict
| Mức | Ý nghĩa | Ví dụ |
| :--- | :--- | :--- |
| `S0 BLOCK` | mất tiền / sai số dư / lộ PAN-PII-secret / phá tương thích ngược contract đang chạy | `B-SEC-01`, `B-SEC-11`, `F-*` mức S0, phá CP-x |
| `S1 MUST-FIX` | sai nghiệp vụ, race, thiếu idempotency chưa chứng minh mất tiền | `B-LOG-*` concurrency, `B-CON-*` event schema |
| `S2 SHOULD` | nợ kỹ thuật, vi phạm pattern/layer | `B-DES-*`, god-file |
| `S3 NIT` | style | — |

**Verdict:** `BLOCK` nếu ≥1 S0; `REQUEST_CHANGES` nếu ≥1 S1; còn lại `APPROVE` (S2/S3 chỉ ghi nhận, không kéo vòng).

## 7. Output & ghi sổ
- `memory/workspaces/<ws>/scope_<suffix>.json`, `review_report_<suffix>.md`, `claims_dg3_<suffix>.json`, `06_laya_summary_*.json`.
- Mọi lệnh test/build chạy để chứng minh finding → qua `$T/evidence_run.py --workspace <ws> --kind unit|integration|runtime -- <cmd>` để có `EV-...`.
- Append `05_progress_<suffix>.md`: `START review` trước R0, `DONE review verdict=<...>` sau R6.
- Khi dùng trong S5: report này **là** phần (a)+(e) của `audit_report_<suffix>.md`; dán link hoặc nội dung vào đó.

## 8. Cấm
- Sửa source code, sửa file trong `agent/memory/` đã tồn tại, chạy `git push`/`reset --hard`.
- Hardcode từ khoá dự án vào `scope.py`, `base/*.md`, `packs/*.md` (ADR-07) — chỉ Profile/manifest được chứa tên service/path.
- Báo finding từ suy diễn không có REF; báo `PASS` không có `06_laya_summary` hoặc `EV-`.
