# 09 — Solution: Skill `/diff-review` (1 phương án duy nhất — đã APPROVE)

## 1. Kiến trúc 3 lớp luật + Engine
```
L3 PROJECT PROFILE   memory/global/project/<name>/review_profile.md   (kiến trúc, chokepoints, contracts, god-files)
L2 DOMAIN PACK       rules/review/packs/<pack>.md                     (fintech: tiền, ledger, idempotency, PCI/PII)
L1 BASE RULES        rules/review/base/{security,logic,design,contract}.md
L0 ENGINE            skills/diff-review/SKILL.md + tooling/diff_review/scope.py + report_template.md
```
Manifest `<repo>/.agent-project.json` khoá `review` quyết định L2/L3 nào được nạp. Không manifest → chỉ L1.

## 2. Pipeline
| Bước | Ai | Token | Việc |
| :--- | :--- | :--- | :--- |
| R0 SCOPE | `scope.py` | 0 | diff name-status + numstat; gắn layer/group theo `review.layers`; cờ `chokepoints_touched`, `god_files` (> `review.god_file_loc`, mặc định 600), `money_path` (regex từ pack/manifest), `layer_violations` (import ngược theo `review.layer_order`); xuất JSON |
| R1–R4 | 4 lens (sub-agent song song hoặc tuần tự) | model | Mỗi lens nhận: hunks liên quan + checklist L1(+L2/L3) của lens + profile; trả findings `REF:` |
| R5 GATE | `laya_gate.py review --workspace <ws> --repo <repo> [--diff-file <f>] [--max-rounds 3] [--fencing <json>]` (VERIFIED `laya_gate.py review --help`, 2026-11-17) + `laya_gate.py run --stage dg3 --claims <json>` cho findings | 0 | Mode B ≤3 vòng (engine tự gửi corrupted-diff control, `laya_gate.py:22`); Mode A claims `kind: claim|demo|control_true|control_false`, `evidence: "REF: <abs>:<start>-<end>\n<quoted lines>"`, cần ≥2 control_false + ≥1 control_true (`laya_gate.py:283-286`) |
| R6 REPORT | Brain | model | Gộp/khử trùng → `review_report_<suffix>.md`, verdict |

## 3. Severity & Verdict
- `S0 BLOCK`: mất tiền / sai số dư / lộ PAN-PII-secret / phá tương thích ngược contract đang chạy.
- `S1 MUST-FIX`: sai nghiệp vụ, race, thiếu idempotency nhưng chưa chứng minh mất tiền.
- `S2 SHOULD`: nợ kỹ thuật, vi phạm pattern/layer.
- `S3 NIT`: style.
- Verdict: `BLOCK` nếu ≥1 S0; `REQUEST_CHANGES` nếu ≥1 S1; còn lại `APPROVE` (S2/S3 chỉ ghi nhận, không kéo vòng).

## 4. Schema manifest `.agent-project.json` (mở rộng, backward compatible)
```json
"review": {
  "packs": ["fintech"],
  "profile": "memory/global/project/data-hub/review_profile.md",
  "god_file_loc": 600,
  "layer_order": ["server", "handler", "service", "repository", "model"],
  "layers": { "handler": "internal/handler/", "service": "internal/service/", "repository": "internal/repository/", "model": "internal/model/", "server": "internal/server/" },
  "chokepoints": [ { "id": "CP-1", "paths": ["internal/service/source/metadata_registry_service.go"], "symbols": ["ResolveSourceRoutes"] } ],
  "money_path_keywords": ["amount", "balance", "ledger", "payment", "refund", "fee", "settle", "wallet", "txn"],
  "forbidden_ifelse_files": ["internal/service/master/transmuter.go"]
}
```

## 5. Code Demo — `scope.py` (lõi)
```python
# usage: scope.py --repo <path> --base <ref> --head <ref> [--manifest <path>] [--pack <path>...] --out scope.json
diff = git(repo, "diff", "--name-status", f"{base}..{head}")
numstat = git(repo, "diff", "--numstat", f"{base}..{head}")
for f in files:
    f.layer = match_layer(f.path, cfg.layers)
    f.loc_after = count_lines(repo, head, f.path)        # git show head:path | wc -l
    f.god_file = f.loc_after > cfg.god_file_loc and f.added > 0
    f.money_path = any(rx.search(hunk_text) for rx in money_rx)
cps = [cp for cp in cfg.chokepoints if touches(cp, files, hunks)]
violations = check_layer_imports(repo, head, changed_go_files, cfg.layer_order, cfg.layers)
json.dump({...}, out)
```
Toàn bộ từ khoá/đường dẫn nằm trong manifest/pack → `scope.py` 0 hardcode dự án (TC-1).

## 6. Đăng ký (VERIFIED 2026-11-17 sau khi đọc được core)
- `.agents/skills.json` thật = `{"entries":[{"path":"/Users/trainguyen/Documents/work/agent/skills"}]}` → đăng ký **theo thư mục**, không theo từng skill. Tạo `skills/diff-review/SKILL.md` với frontmatter YAML `name` + `description` (giống `skills/audit-propose-only/SKILL.md:1-7`) là đủ, **không cần sửa** `skills.json`.
- Biên lai test: `python3 tooling/evidence_run.py --workspace diff-review-skill-2026-11 --kind unit -- <cmd>` (VERIFIED `evidence_run.py:4-6`).
- Engine gate: VegaML `:8222` `/health` = ok (VERIFIED curl 2026-11-17).
- `OPERATOR_MAP.md` bảng 1, thêm dòng (sửa file này là sửa core doc, em sẽ làm ở S4 cùng skill):
  `| \`diff-review\` | QA / Brain | \`S5\` hoặc Audit độc lập | Review diff theo 4 lens (Security, Logic, Design, Contract) với luật 3 lớp (Base → Pack → Profile), Propose-Only, xuất \`review_report_<suffix>.md\` + verdict. |`
- `.agent-project.json` của từng repo: thêm khoá `review` (mục 4).

## 7. Đối chiếu 5 Chokepoints / Blast-radius
Không sửa code dự án nào; chỉ thêm file mới trong `agent/`. Blast-radius = 0 với các skill/tool hiện có (không sửa file cũ).
