#!/usr/bin/env python3
"""Read-only architecture rule checker for data-hub (draft, lives in agent workspace).

Usage:
  arch_check.py <data-hub-root> [--allowlist FILE] [--write-allowlist FILE]
Exit 1 if a violation exists that is NOT in the allowlist (allowlist may only shrink).
"""
import json
import re
import sys
from pathlib import Path

SQL_RE = re.compile(r"\.(Exec|Raw)\(")
GORM_FIELD_RE = re.compile(r"^\s+\w+\s+\*gorm\.DB\b", re.M)
HTTP_DIRECT_RE = re.compile(r"\baxios\.|\bfetch\(")
BAD_NAME_RE = re.compile(r"(_v\d+|legacy|_bk|_old|_copy)", re.I)
MAX_LINES = 600


def go_files(base: Path, sub: str):
    root = base / sub
    if not root.exists():
        return []
    return [p for p in root.rglob("*.go") if not p.name.endswith("_test.go")]


def imports(text: str):
    return re.findall(r'"([^"]+)"', "\n".join(re.findall(r"import\s*\((.*?)\)", text, re.S)) + "\n" +
                      "\n".join(re.findall(r'^import\s+(".*?")', text, re.M)))


def check(root: Path):
    v = []  # (rule, relpath)

    def add(rule, p):
        v.append(f"{rule}|{p.relative_to(root).as_posix()}")

    # ---- centralized-data-service (layered: handler -> service -> repository -> model)
    ds = root / "centralized-data-service"
    mod = "centralized-data-service/internal"
    for p in go_files(ds, "internal"):
        rel = p.relative_to(ds).as_posix()
        t = p.read_text(encoding="utf-8", errors="ignore")
        imps = imports(t)
        if SQL_RE.search(t) and not rel.startswith("internal/repository/"):
            add("DS-R1-sql-outside-repository", p)
        if rel.startswith("internal/handler/"):
            if any(i.startswith(f"{mod}/repository") for i in imps):
                add("DS-R2-handler-imports-repository", p)
            if GORM_FIELD_RE.search(t):
                add("DS-R3-handler-holds-gorm", p)
        if rel.startswith("internal/service/") and any(i.startswith(f"{mod}/handler") for i in imps):
            add("DS-R4-service-imports-handler", p)
        if rel.startswith("internal/repository/") and any(i.startswith(f"{mod}/service") or i.startswith(f"{mod}/handler") for i in imps):
            add("DS-R5-repository-imports-upper", p)
        if BAD_NAME_RE.search(p.stem):
            add("ALL-R6-parallel-version-file", p)
        if t.count("\n") > MAX_LINES:
            add("ALL-R7-file-over-600-lines", p)
    for d in ("docs", "scratch"):
        if any((ds / d).rglob("*.go")):
            v.append(f"ALL-R8-go-code-in-{d}|centralized-data-service/{d}")

    # ---- cdc-cms-service (hexagonal: api -> app(ports) -> domain; infra implements ports)
    cs = root / "cdc-cms-service"
    cmod = "cdc-cms-service/internal"
    for p in go_files(cs, "internal"):
        rel = p.relative_to(cs).as_posix()
        t = p.read_text(encoding="utf-8", errors="ignore")
        imps = imports(t)
        if rel.startswith("internal/api/"):
            if any(i.startswith(f"{cmod}/infra") for i in imps):
                add("CS-R1-api-imports-infra", p)
            if SQL_RE.search(t):
                add("CS-R2-api-raw-sql", p)
        if rel.startswith("internal/domain/") and any(i.startswith(f"{cmod}/infra") or i.startswith(f"{cmod}/app") or "gorm.io" in i for i in imps):
            add("CS-R3-domain-impure", p)
        if rel.startswith("internal/app/") and any(i.startswith(f"{cmod}/infra") for i in imps):
            add("CS-R4-app-imports-infra", p)
        if any(i.startswith(f"{cmod}/model") for i in imps) and not rel.startswith("internal/infra/persistence/") and not rel.startswith("internal/model/"):
            add("CS-R5-legacy-model-outside-persistence", p)
        if BAD_NAME_RE.search(p.stem):
            add("ALL-R6-parallel-version-file", p)
        if t.count("\n") > MAX_LINES and not rel.startswith("docs/"):
            add("ALL-R7-file-over-600-lines", p)
    if any((cs / "scratch").rglob("*.go")) if (cs / "scratch").exists() else False:
        v.append("ALL-R8-go-code-in-scratch|cdc-cms-service/scratch")

    # ---- cdc-cms-web (pages -> hooks -> services/api.ts)
    w = root / "cdc-cms-web" / "src"
    for p in list((w / "pages").rglob("*.tsx")) + list((w / "components").rglob("*.tsx")):
        t = p.read_text(encoding="utf-8", errors="ignore")
        if HTTP_DIRECT_RE.search(t):
            add("WEB-R1-ui-calls-http-directly", p)
        if t.count("\n") > MAX_LINES:
            add("ALL-R7-file-over-600-lines", p)
    return sorted(set(v))


def main():
    root = Path(sys.argv[1]).resolve()
    args = sys.argv[2:]
    viol = check(root)
    if "--write-allowlist" in args:
        Path(args[args.index("--write-allowlist") + 1]).write_text("\n".join(viol) + "\n", encoding="utf-8")
    allow = set()
    if "--allowlist" in args:
        allow = set(Path(args[args.index("--allowlist") + 1]).read_text().split())
    new = [x for x in viol if x not in allow]
    summary = {}
    for x in viol:
        summary[x.split("|")[0]] = summary.get(x.split("|")[0], 0) + 1
    print(json.dumps({"total": len(viol), "by_rule": summary, "new_not_allowlisted": new[:50]}, indent=1, ensure_ascii=False))
    sys.exit(1 if new else 0)


if __name__ == "__main__":
    main()
