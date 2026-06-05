# 06_validation.md — Test Plan + Verification Gate

> Mỗi phase có verification gate riêng. File này định nghĩa command + expected result + tool config.

---

## 1. Verification Pyramid

```
                     ┌──────────────┐
                     │  Smoke 53    │  (1-5 min, manual after deploy)
                     │  routes      │
                     └──────┬───────┘
                ┌───────────┴──────────┐
                │  Integration tests   │  (5-15 min, CI)
                │  (DB + NATS + Redis) │
                └───────────┬──────────┘
        ┌───────────────────┴──────────────────┐
        │   Unit tests (test/...)             │  (< 1 min, IDE + CI)
        │   - commands / queries / handlers   │
        └─────────────────────────────────────┘
              ↑
              │
        ┌─────┴───────────────────┐
        │ Static analysis         │  (< 30s, pre-commit)
        │ - go vet                │
        │ - golangci-lint         │
        │ - go-arch-lint          │
        └─────────────────────────┘
```

---

## 2. Test commands cheatsheet

| Command | Mục đích | Khi nào chạy |
|---|---|---|
| `make test` | Unit (test/... -short) | Sau mỗi task |
| `make test-integration` | Cần DB/NATS/Redis up | Cuối mỗi phase |
| `make test-cover` | Coverage report | Phase 0 baseline + cuối mỗi phase |
| `go vet ./...` | Static check builtin | Pre-commit |
| `golangci-lint run ./...` | Style + lint | Pre-commit |
| `make arch-lint` | Architecture rule | Pre-commit (sau Phase 0) |
| `make swagger` | Regenerate OpenAPI | Sau mỗi phase |
| `bash scripts/smoke_routes.sh` | 53 routes API | Cuối mỗi phase |

---

## 3. PHASE 0 — Validation

### 3.1 Coverage baseline
```bash
make test-cover
go tool cover -func=coverage.out | tail -20
# Expect: total coverage record
```
**Expected:** baseline saved trong workspace.

### 3.2 Coverage after T0
```bash
go tool cover -func=coverage.out | grep "internal/app"
# Expect: ≥ 60%

go tool cover -func=coverage.out | grep "internal/server"
# Expect: ≥ 60%

go tool cover -func=coverage.out | grep "internal/bootstrap"
# Expect: ≥ 70%
```

### 3.3 `go-arch-lint` skeleton
```bash
go-arch-lint check --arch-file tools/lint/arch-lint.yml
# Expect: chạy được, có thể FAIL (vì 18 cmd còn gorm — sẽ fix Phase 3)
```

### 3.4 Smoke 53 routes baseline
```bash
make run &
SERVICE_PID=$!
sleep 5
bash scripts/smoke_routes.sh > smoke_baseline_2026-06-01.txt
kill $SERVICE_PID
grep -c "PASS" smoke_baseline_2026-06-01.txt
# Expect: 53
```

### Gate G0 verify script
```bash
#!/usr/bin/env bash
set -e

# Coverage check
APP_COV=$(go tool cover -func=coverage.out | grep "internal/app" | awk '{print $3}' | head -1 | tr -d '%')
SERVER_COV=$(go tool cover -func=coverage.out | grep "internal/server" | awk '{print $3}' | head -1 | tr -d '%')
BOOT_COV=$(go tool cover -func=coverage.out | grep "internal/bootstrap" | awk '{print $3}' | head -1 | tr -d '%')

[ "$(echo "$APP_COV >= 60" | bc)" -eq 1 ] || { echo "FAIL: app coverage $APP_COV < 60%"; exit 1; }
[ "$(echo "$SERVER_COV >= 60" | bc)" -eq 1 ] || { echo "FAIL: server coverage $SERVER_COV < 60%"; exit 1; }
[ "$(echo "$BOOT_COV >= 70" | bc)" -eq 1 ] || { echo "FAIL: bootstrap coverage $BOOT_COV < 70%"; exit 1; }

# Smoke check
PASS=$(grep -c "PASS" smoke_baseline_2026-06-01.txt)
[ "$PASS" -eq 53 ] || { echo "FAIL: smoke $PASS/53"; exit 1; }

echo "G0 PASS: app=$APP_COV%, server=$SERVER_COV%, boot=$BOOT_COV%, smoke=$PASS/53"
```

---

## 4. PHASE 1 — Validation

### 4.1 AC-2: 0 legacy interface import
```bash
grep -rn "ports.MasterRepo\|ports.MappingRuleRepo\|ports.SourceRepo\|ports.JobRepo\|ports.ReconRepo\|ports.WizardRepo\|ports.ConnectorRepo\|ports.RegistryRepo\|ports.TransformRepo" internal/
# Expect: empty
```

### 4.2 AC-3: `repository.go` deleted
```bash
test ! -f internal/app/ports/repository.go && echo "PASS" || echo "FAIL"
```

### 4.3 Build + test
```bash
go build ./... && make test && make test-integration
```

### 4.4 Coverage không regression
```bash
make test-cover
# Compare with baseline — must be ≥
diff <(grep "total" coverage_baseline_2026-06-01.txt) <(grep "total" coverage.out)
```

### 4.5 Port files ≤ 80 LOC
```bash
for f in internal/app/ports/*_port.go; do
  lines=$(wc -l < "$f")
  echo "$f: $lines"
  [ "$lines" -le 80 ] || { echo "FAIL: $f $lines > 80"; exit 1; }
done
```

### 4.6 Smoke 53 routes
```bash
bash scripts/smoke_routes.sh
# Expect: 53/53 PASS
```

### Gate G1 verify
- AC-2 PASS
- AC-3 PASS
- All test PASS
- Coverage ≥ baseline
- Smoke 53/53

---

## 5. PHASE 2 — Validation

### 5.1 AC-4: `server.go` ≤ 80 LOC
```bash
wc -l internal/server/server.go
# Expect: ≤ 80
```

### 5.2 AC-5: 5 file mới tồn tại
```bash
ls internal/server/{infra,repos,bus,routes,workers}.go
# Expect: all exist
```

### 5.3 Pure function check (manual review)
```bash
# Mỗi sub-builder phải nhận input explicit, không capture state
grep -n "func build" internal/server/*.go
# Manually verify: KHÔNG có receiver-state mutation
```

### 5.4 NFR-3: 26 NATS subject preserve
```bash
# So với baseline
grep -roh '"cdc\.cms\.[a-z._]*"' internal/server/ | sort -u > nats_subjects_after_phase2.txt
diff nats_subjects_baseline.txt nats_subjects_after_phase2.txt
# Expect: empty diff
```

### 5.5 NFR-5: Start time ≤ baseline + 100ms
```bash
# Measure 3 lần, lấy trung bình
for i in 1 2 3; do
  /usr/bin/time -p ./bin/cms 2>&1 | grep "real" &
  sleep 5
  pkill -f "./bin/cms"
done
```

### 5.6 Integration + smoke
```bash
make test-integration && bash scripts/smoke_routes.sh
```

### Gate G2 verify
- AC-4 + AC-5 PASS
- 26 NATS subject preserve
- Start time NFR-5 PASS
- Integration + smoke PASS

---

## 6. PHASE 3 — Validation

### 6.1 AC-6: 0 command import gorm
```bash
grep -rn "gorm.io/gorm" internal/app/commands/
# Expect: empty
```

### 6.2 `go-arch-lint` strict PASS
```bash
make arch-lint
# Expect: PASS (vì rule `app cannot depend gorm` giờ thoả)
```

### 6.3 Per-command verify (mỗi commit)
```bash
# Trong mỗi commit refactor T3.X:
git show --stat HEAD | grep "internal/app/commands/<name>.go"
# Expect: file modified

# Test riêng cmd đó
go test ./test/app/commands/<name>_test.go -count=1 -v
# Expect: PASS
```

### 6.4 Coverage không regression
```bash
make test-cover
diff <(grep "total" coverage_baseline_2026-06-01.txt) <(grep "total" coverage.out)
```

### 6.5 NFR-4: DB schema không đổi
```bash
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "\d+ cdc_cms.*" > schema_after_phase3.txt
diff schema_baseline.txt schema_after_phase3.txt
# Expect: empty diff
```

### 6.6 Integration + smoke
```bash
make test-integration && bash scripts/smoke_routes.sh
```

### Gate G3 verify
- AC-6 PASS
- arch-lint PASS
- 18 commits độc lập (`git log --oneline | grep refactor(cmd)` = 18)
- DB schema unchanged
- Coverage ≥ baseline
- Smoke 53/53

---

## 7. PHASE 4 — Validation (OPTIONAL)

### 7.1 AC-7: 0 cross-BC import (trừ wizard)
```bash
make arch-lint
# Expect: PASS với rule strict
```

### 7.2 Git blame preserve (NFR-8)
```bash
git log --follow --oneline internal/bc/source/commands/register_source.go | wc -l
# Expect: > 1 (có history từ thời file ở internal/app/commands/)
```

### 7.3 Smoke + integration
```bash
make test-integration && bash scripts/smoke_routes.sh
make swagger  # confirm no drift
```

### 7.4 Per-BC verify (sau mỗi T4.X)
```bash
# BC self-contained: tất cả file của BC X nằm trong internal/bc/X/
find internal/bc/source -type f -name "*.go" | wc -l
# Expect: matching số file ban đầu của BC source
```

### Gate G4 verify
- AC-7 PASS
- 8 BC self-contained
- Wizard exception documented + linter pass
- Swagger no drift
- Smoke 53/53

---

## 8. Regression checklist (CHẠY SAU MỖI PHASE)

```bash
#!/usr/bin/env bash
# regression_check.sh
set -e

echo "[1/8] go vet"
go vet ./...

echo "[2/8] golangci-lint"
golangci-lint run ./...

echo "[3/8] go-arch-lint"
make arch-lint

echo "[4/8] make test"
make test

echo "[5/8] make test-integration"
make test-integration

echo "[6/8] coverage"
make test-cover
go tool cover -func=coverage.out | tail -1

echo "[7/8] swagger"
make swagger
git diff --exit-code docs/ || { echo "FAIL: swagger drift"; exit 1; }

echo "[8/8] smoke 53 routes"
make run &
PID=$!
sleep 5
bash scripts/smoke_routes.sh
kill $PID

echo "ALL REGRESSION CHECKS PASSED"
```

---

## 9. Test data + fixtures

### 9.1 Test database
- Local: docker container `gpay-postgres-cdc` (đã có sẵn từ Makefile)
- CI: testcontainers Postgres init từ `migrations/schema/*.sql`

### 9.2 Mock NATS
- Local: docker container NATS
- CI: `nats-server -DV` embedded process

### 9.3 Mock Redis
- Local/CI: `miniredis` (in-memory cho unit), real Redis cho integration

### 9.4 Test fixtures
- `test/fixtures/sources.json` — sample source register payload
- `test/fixtures/mapping_rules.json` — sample mapping
- `test/fixtures/master_bindings.json` — sample master

---

## 10. Performance regression (NFR-5, NFR-6)

### 10.1 Start time
```bash
# Baseline (Phase 0)
for i in 1 2 3; do
  /usr/bin/time -p ./bin/cms 2>&1 | grep "real"
  sleep 2
  pkill -f "./bin/cms"
done > start_time_baseline.txt
```

### 10.2 Memory footprint
```bash
# Idle 10s sau start
./bin/cms &
PID=$!
sleep 10
ps -o rss= -p $PID
kill $PID
```

NFR-5: start time ≤ baseline + 100ms
NFR-6: memory ≤ baseline + 5%

---

## 11. Security verification (rule §8)

Sau mỗi phase, chạy `/security-agent`:
- Static analysis: gosec
- Dependency vuln: govulncheck
- Secrets scan: gitleaks

DoD security:
- 0 high/critical finding
- 0 secret leak
- 0 known CVE trong dependency

---

## 12. Rollback verification

### 12.1 Phase 1-3 rollback
```bash
# Tag trước mỗi phase
git tag pre-phase1
git tag pre-phase2
git tag pre-phase3

# Revert
git revert <commit-range>
# Re-run G0 + smoke 53 routes — expect PASS
```

### 12.2 Phase 4 rollback
- Mỗi BC move = 1 commit riêng → revert riêng từng BC
- Worst case: `git revert` toàn bộ Phase 4, giữ Phase 1-3

---

## 13. Acceptance Test (AT) format cho User

Sau khi Muscle báo phase done, User chạy:
```bash
cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
bash regression_check.sh
```

Kết quả expected: `ALL REGRESSION CHECKS PASSED` + cụ thể cho phase đó AC PASS.

User approve trong workspace `05_progress.md`:
```
[2026-06-XX HH:MM] [User] APPROVE Phase X — AC verified
```
