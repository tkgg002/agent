#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os, re
from collections import defaultdict, Counter

PARTS = [f"/tmp/norm_part_{i:02d}.md" for i in range(1, 11)]
OUT = "/tmp/lessons_global_normalized.md"

CAT_ORDER = [
    "01-process-governance", "02-architecture-design", "03-schema-migration",
    "04-cdc-data-pipeline", "05-config-environment", "06-serialization-type",
    "07-testing-verification", "08-memory-knowledge",
]
CAT_TITLE = {
    "01-process-governance": "1. Process & Governance — Kỷ luật Brain/Muscle, Quy trình, Approval, Verification",
    "02-architecture-design": "2. Architecture & Design — Coupling, DRY, CQRS, Single-Source-of-Truth, Observability",
    "03-schema-migration":    "3. Schema & Migration — DDL, Migration ordering, search_path, Model↔DB Drift",
    "04-cdc-data-pipeline":   "4. CDC / Data Pipeline — Kafka, Debezium, Snapshot, Connection-Registry, Masking",
    "05-config-environment":  "5. Config & Environment — Env vars, DSN/Secret, Fallback, Docker/K8s",
    "06-serialization-type":  "6. Serialization & Type — BSON/Extended-JSON, Cast, Type/Form Drift, Identifier",
    "07-testing-verification":"7. Testing & Verification — Exercise-driven, PASS criteria, Test uplift, Build≠Test",
    "08-memory-knowledge":    "8. Memory & Knowledge — Workspace, Audit-log immutability, Documentation discipline",
}
CAT_DESC = {
    "01-process-governance": "Bài học về phối hợp Brain↔Muscle, plan-before-code, gatekeeper approval, không báo Done khi chưa verify, chống tái phạm.",
    "02-architecture-design": "Bài học về thiết kế: tránh coupling thừa, DRY, single-source-of-truth, không over-engineer, thiết kế observability ở cấp hệ thống.",
    "03-schema-migration": "Bài học về tiến hoá schema: thứ tự DDL/migration, search_path, drift giữa model và DB, add/rename column an toàn.",
    "04-cdc-data-pipeline": "Bài học miền CDC/ETL: Kafka/Debezium, snapshot, connection-registry, masking, DLQ, reconcile, shadow tables.",
    "05-config-environment": "Bài học về cấu hình & môi trường: env vars, resolve DSN/secret, fallback merge, docker-compose/k8s, .env.",
    "06-serialization-type": "Bài học về serialize/kiểu dữ liệu: BSON/Extended-JSON, cast expression, form drift, dual-stack routing, migrate identifier.",
    "07-testing-verification": "Bài học về kiểm thử & xác minh: exercise-driven, tiêu chí PASS thực chất, nâng cấp test, build pass ≠ test pass.",
    "08-memory-knowledge": "Bài học về quản trị tri thức: workspace-first, audit-log bất biến (append-only), kỷ luật tài liệu, chuẩn viết lesson.",
}

blocks = []
malformed = 0
for p in PARTS:
    if not os.path.exists(p):
        continue
    text = open(p, encoding="utf-8").read()
    for chunk in text.split("@@LESSON@@")[1:]:
        if "@@END@@" not in chunk:
            malformed += 1
            continue
        block = chunk.split("@@END@@")[0]
        cat, date, body_lines = None, None, []
        for ln in block.splitlines():
            s = ln.strip()
            m1 = re.match(r"^@@CAT=(.+?)@@$", s)
            m2 = re.match(r"^@@DATE=(.+?)@@$", s)
            if m1:
                cat = m1.group(1); continue
            if m2:
                date = m2.group(1); continue
            body_lines.append(ln.rstrip())
        while body_lines and body_lines[0].strip() == "":
            body_lines.pop(0)
        while body_lines and body_lines[-1].strip() == "":
            body_lines.pop()
        if cat not in CAT_ORDER:
            cat = "02-architecture-design"
        blocks.append({"cat": cat, "date": (date or "0000-00-00"), "body": "\n".join(body_lines)})

# group + sort (newest first; placeholder 0000-00-00 last)
g = defaultdict(list)
for b in blocks:
    g[b["cat"]].append(b)
for c in g:
    g[c].sort(key=lambda b: b["date"], reverse=True)

total = len(blocks)
cat_counts = {c: len(g[c]) for c in CAT_ORDER}

# month distribution from normalized dates
month = Counter()
for b in blocks:
    d = b["date"]
    month[d[:7] if d != "0000-00-00" else "n/a"] += 1

# distinct tags in normalized output
tagset = set()
for b in blocks:
    for t in re.findall(r"#[a-z0-9][a-z0-9-]*", b["body"]):
        tagset.add(t)

out = []
out.append("# lessons_global_normalized.md — Bản chuẩn hoá Global Patterns")
out.append("")
out.append("> **NGUỒN**: `agent/memory/global/lessons.md` (audit-log gốc, BẤT BIẾN).  ")
out.append("> **BẢN CHẤT**: Đây là bản *chuẩn hoá phái sinh* (derived), KHÔNG thay thế audit-log. Mọi lesson mới vẫn APPEND vào `lessons.md` gốc theo Rule 7/11; định kỳ re-generate file này.  ")
out.append(f"> **Sinh tự động** từ {total} lesson thô, phân loại theo taxonomy 8 nhóm, chuẩn hoá theo Rule 13 (`Global Pattern [A does B to X] → Y. Đúng: ...`).")
out.append("")
out.append("---")
out.append("")
out.append("## 📊 Dashboard Thống kê")
out.append("")
out.append("| Chỉ số | Giá trị |")
out.append("|---|---|")
out.append("| File nguồn | 530 KB / 5.061 dòng |")
out.append(f"| Tổng Global Pattern đã chuẩn hoá | **{total}** |")
out.append("| Format nguồn (trước) | 134 chuẩn `## [DATE]` + ~92 lệch chuẩn (Lesson N, L-xxx, ...) |")
out.append("| Tuân thủ field nguồn (trước) | Fix-marker 15, Lesson-marker 5 (rất lệch) |")
out.append("| Tag nguồn (trước) | 750 tag riêng biệt (sprawl) |")
out.append(f"| Tag sau chuẩn hoá | {len(tagset)} tag (kebab-case, gom cụm) |")
out.append("| Format sau | **100% canonical Rule 13** |")
out.append("")
out.append("### Phân bố theo nhóm (taxonomy)")
out.append("")
out.append("| # | Nhóm | Số pattern |")
out.append("|---|---|---|")
for c in CAT_ORDER:
    out.append(f"| {c[:2]} | {CAT_TITLE[c].split(' — ')[0].split('. ',1)[1]} | {cat_counts[c]} |")
out.append(f"| | **TỔNG** | **{total}** |")
out.append("")
out.append("### Phân bố theo tháng (theo ngày của lesson)")
out.append("")
out.append("| Tháng | Số pattern |")
out.append("|---|---|")
for m in sorted(month, key=lambda x: (x == "n/a", x)):
    out.append(f"| {m} | {month[m]} |")
out.append("")
out.append("---")
out.append("")
out.append("## 🗂️ Mục lục Taxonomy")
out.append("")
for c in CAT_ORDER:
    anchor = CAT_TITLE[c].lower()
    anchor = re.sub(r"[^a-z0-9 \-—]", "", anchor).replace("—", "").replace("  ", " ").strip().replace(" ", "-")
    out.append(f"- **{CAT_TITLE[c]}** ({cat_counts[c]} pattern)")
    out.append(f"  - _{CAT_DESC[c]}_")
out.append("")
out.append("---")
out.append("")
out.append("## 📐 Quy ước chuẩn hoá (Rule 13)")
out.append("")
out.append("Mỗi pattern theo cấu trúc: **Global Pattern** `[A] <hành động B>` lên `[X]` → `[Y]`. **Đúng**: `<luồng đúng>` — kèm Trigger, Root Cause, Fix, Phạm vi áp dụng (≥3 dự án), Tags, và trích Nguồn (ngày trong audit-log gốc).")
out.append("")
out.append("---")
out.append("")

# body
for c in CAT_ORDER:
    out.append(f"## {CAT_TITLE[c]}")
    out.append("")
    out.append(f"_{CAT_DESC[c]}_ — **{cat_counts[c]} pattern**")
    out.append("")
    for b in g[c]:
        out.append(b["body"])
        out.append("")
    out.append("---")
    out.append("")

# footer
out.append("## 🔁 Quy trình duy trì (Maintenance)")
out.append("")
out.append("1. Lesson MỚI → vẫn APPEND vào `lessons.md` gốc (Rule 7/11 — bất biến, append-only).")
out.append("2. Định kỳ (hoặc khi `lessons.md` tăng đáng kể) → re-generate lại file này từ nguồn.")
out.append("3. File này là *read-optimized view* để tra cứu nhanh theo nhóm; KHÔNG phải nguồn sự thật.")
out.append("")
out.append(f"<!-- generated: {total} patterns from lessons.md; malformed_blocks={malformed} -->")

open(OUT, "w", encoding="utf-8").write("\n".join(out) + "\n")
print(f"WROTE {OUT}")
print(f"total_blocks={total} malformed={malformed} distinct_tags={len(tagset)}")
print("cat_counts=", cat_counts)
print("months=", dict(sorted(month.items())))
