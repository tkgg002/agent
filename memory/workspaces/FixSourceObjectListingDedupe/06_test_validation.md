# 06_test_validation — FixSourceObjectListingDedupe

## Verification Evidence

### 1. Compile
- `go build ./...` → ✅ PASS (no output, exit 0)
- `go vet ./...` → ✅ PASS
- `go build -o /tmp/cdc-cms-service-fixdedupe ./cmd/server` → ✅ PASS (58.3 MB binary)

### 2. Unit Tests
- `go test -count=1 ./internal/infra/persistence/...` → ✅ PASS (0.574s)
- `go test -count=1 ./internal/app/queries/...` → ✅ PASS (0.782s)
- `go test ./internal/...` → ✅ ALL PASS (no regression)

### 3. SQL Live Verification (DB direct, gpay-postgres-cdc:5433 / cdc_dw)

**Old JOIN (reproduce bug)**: returns **6 rows** with cross-connection bleed.
```
 so_id | tr_id | so_conn | tr_conn
-------+-------+---------+---------
     1 |     1 |       2 |       2
     1 |     4 |       2 |      42   ← BUG: so.conn=2 matched tr.conn=42
     5 |     2 |       5 |       5
    18 |     3 |      21 |      21
    36 |     1 |      42 |       2   ← BUG: so.conn=42 matched tr.conn=2
    36 |     4 |      42 |      42
```

**New JOIN (LATERAL + conn scope)**: returns **4 rows** clean.
```
 so_id | tr_id | so_conn | tr_conn
-------+-------+---------+---------
     1 |     1 |       2 |       2
     5 |     2 |       5 |       5
    18 |     3 |      21 |      21
    36 |     4 |      42 |      42
```

### 4. Acceptance Criteria
- AC-1 ✅ Listing → 4 rows (was 6).
- AC-2 ✅ id=1 → registry_id=1 (conn 2).
- AC-3 ✅ id=36 → registry_id=4 (conn 42).
- AC-4 ✅ Build + vet + test PASS.

---

## Security Self-Review

| Check | Result |
|-------|--------|
| Parameterized query (no concat) | ✅ Same `?` bind args, no change |
| New user-input vectors | ✅ None — only adds `so.source_connection_id` (internal column) |
| Auth/authz changes | ✅ None |
| Secrets exposure | ✅ None |
| Info disclosure | ✅ Same SELECT columns |
| DoS risk | ✅ LATERAL LIMIT 1 bounded; index `idx_ctr_source_connection` covers WHERE |
| Logic bypass via NULL | ⚠ Legacy `source_connection_id IS NULL` fallback could let a malicious tr row match any so. Risk accepted — registration of `cdc_table_registry` is admin-only via control plane, not user-facing. |

→ No blocking security findings. Safe to deploy.
