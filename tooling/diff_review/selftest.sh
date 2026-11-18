#!/usr/bin/env bash
# selftest.sh — dựng git repo tạm, chạy scope.py có/không manifest, kiểm tra JSON đủ cờ.
# Usage: bash tooling/diff_review/selftest.sh   (exit 0 = PASS)
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/scope_selftest.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

R="$TMP/repo"; mkdir -p "$R/internal/handler/pay" "$R/internal/service/pay" "$R/internal/repository"
git -C "$R" init -q -b main
git -C "$R" config user.email t@t; git -C "$R" config user.name t
cat > "$R/internal/service/pay/svc.go" <<'EOF'
package pay
func Settle(x int) int { return x }
EOF
cat > "$R/internal/repository/repo.go" <<'EOF'
package repository
func Save() {}
EOF
git -C "$R" add -A && git -C "$R" commit -qm base
git -C "$R" tag base

# head: handler mới gọi service (OK); repository import handler (violation); money keyword; sửa chokepoint symbol
cat > "$R/internal/handler/pay/h.go" <<'EOF'
package pay
import "example.com/x/internal/service/pay"
func H() { pay.Settle(1) }
EOF
cat > "$R/internal/repository/repo.go" <<'EOF'
package repository
import "example.com/x/internal/handler/pay"
func Save() { var balance float64 = 1; _ = balance; pay.H() }
EOF
cat > "$R/internal/service/pay/svc.go" <<'EOF'
package pay
func Settle(x int) int { return x + 1 }
EOF
git -C "$R" add -A && git -C "$R" commit -qm head
git -C "$R" tag head

cat > "$TMP/manifest.json" <<'EOF'
{ "review": {
  "god_file_loc": 2,
  "layer_order": ["handler", "service", "repository"],
  "layers": { "handler": "internal/handler/", "service": "internal/service/", "repository": "internal/repository/" },
  "chokepoints": [ { "id": "CP-T", "paths": ["internal/service/pay/svc.go"], "symbols": ["Settle"] } ],
  "money_path_keywords": ["balance"]
} }
EOF

python3 -m py_compile "$HERE/scope.py"

python3 "$HERE/scope.py" --repo "$R" --base base --head head --manifest "$TMP/manifest.json" --out "$TMP/with.json"
python3 "$HERE/scope.py" --repo "$R" --base base --head head --out "$TMP/without.json"

python3 - "$TMP/with.json" "$TMP/without.json" <<'EOF'
import json, sys
w = json.load(open(sys.argv[1])); n = json.load(open(sys.argv[2]))
for k in ["files", "god_files", "money_path", "chokepoints_touched", "layer_violations", "summary"]:
    assert k in w and k in n, f"missing key {k}"
assert w["summary"]["files"] == 3, w["summary"]
assert w["chokepoints_touched"] and w["chokepoints_touched"][0]["id"] == "CP-T", w["chokepoints_touched"]
assert w["money_path"] == ["internal/repository/repo.go"], w["money_path"]
assert any(v["from_layer"] == "repository" and v["imports_layer"] == "handler" for v in w["layer_violations"]), w["layer_violations"]
assert any(g["path"] == "internal/repository/repo.go" for g in w["god_files"]), w["god_files"]
assert n["chokepoints_touched"] == [] and n["layer_violations"] == [] and n["money_path"] == [], "no-manifest must yield empty flags"
assert n["summary"]["files"] == 3
print("SELFTEST PASS: with-manifest flags OK; without-manifest no crash, flags empty")
EOF
