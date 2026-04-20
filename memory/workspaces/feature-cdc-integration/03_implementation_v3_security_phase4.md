# 03_implementation_v3_security_phase4.md

Phase 4 — Security hardening for destructive admin endpoints
Owner: Muscle (Chief Engineer) | Date: 2026-04-17
Scope: cdc-cms-service only (Worker and FE untouched)

## 1. Context

Reference plans:
- `02_plan_data_integrity_v3.md` §13 — RBAC + Idempotency + Audit
- `02_plan_observability_v3.md` §8 — Restart Connector RBAC

Phase 0 (silent-bug fix + background collector) and Phase 1 Worker are
already done. This Phase 4 layers five defences in front of every
destructive POST route, so a double-click, a replay, a compromised
viewer token, or a flaky CI retry can never corrupt data or flood the
pipeline.

## 2. Deliverables (one-line summary)

| Task | File | Status |
|------|------|--------|
| T1 Migration | `cdc-cms-service/migrations/005_admin_actions.sql` | Applied to gpay-postgres |
| T2 RBAC | `cdc-cms-service/internal/middleware/rbac.go` (+jwt.go patch) | Unit tests green |
| T3 Idempotency | `cdc-cms-service/internal/middleware/idempotency.go` | Unit tests green |
| T4 Audit | `cdc-cms-service/internal/middleware/audit.go` | Unit+runtime tests green |
| T5 Route wiring | `cdc-cms-service/internal/router/router.go`, `server/server.go` | Curl evidence below |
| T6 Rate limit | `cdc-cms-service/internal/middleware/ratelimit.go` | Runtime verified |

## 3. Task 1 — `admin_actions` audit table

Partitioned by RANGE on `created_at`, monthly partitions for 2026-04,
05, 06, plus a DEFAULT partition (ops continuity — we never want a
write to fail because someone forgot to create next month's partition).
Primary key is `(created_at, id)` because Postgres requires the
partition key to be part of the PK. Indexes on `(user_id, created_at
DESC)` and `(action, created_at DESC)` for dashboard queries, plus a
partial index on `idempotency_key WHERE NOT NULL`.

Apply evidence:
```
CREATE TABLE (admin_actions)
CREATE TABLE (admin_actions_2026_04 .. 2026_06 + _default)
CREATE INDEX (user, action, idem)
```

## 4. Task 2 — RBAC middleware

**File**: `internal/middleware/rbac.go` (new).
**Patch**: `internal/middleware/jwt.go` now also exposes `roles` claim
if present (`c.Locals("roles", claims["roles"])`), without dropping
the legacy `role` string claim.

Key design:
- `RequireAnyRole(roles ...string)` accepts both legacy `role: string`
  and forward-compat `roles: []string` shapes, plus an `ADMIN_USERS`
  env var fallback (grants `ops-admin` to a comma-separated allowlist).
- `RequireOpsAdmin()` currently accepts `ops-admin` OR `admin` roles —
  this is BACKWARD-COMPAT widening while the IdP rolls out the new
  claim. See TODO(phase-4) below: drop `"admin"` after IdP migration.
- When upstream JWT locals are all empty (unauthenticated request that
  somehow reached the handler), returns **401** instead of 403 so
  monitoring can separate "who are you?" from "you can't do that".

## 5. Task 3 — Idempotency middleware

**File**: `internal/middleware/idempotency.go` (new).
**Support**: `pkgs/rediscache/redis_client.go` extended with
`Client()`, `SetNX`, `Incr`, `Expire`, `TTL` accessors (idempotency +
rate-limit need these primitives; the existing `Get/Set/Del` wrapper
wasn't enough).

Protocol:
- Missing/empty `Idempotency-Key` → 400.
- Key with injection-risk chars (colon, whitespace, `:`) → 400.
- Redis down → 503 (fail-closed; we refuse to serve destructive
  traffic without idempotency enforcement).
- Active lock on same key → 409 with `retry_after: 30`.
- Cached success response (< 1h old) → 200 replay with
  `X-Idempotent-Replay: true` header.
- Only `status < 400` responses are cached; errors stay un-cached so
  clients can retry with the same key.

Key structure: `idem:<route-pattern>:<client-key>:{lock|response}`.
Using the route pattern (`/api/reconciliation/heal/:table`) as part of
the key means the same client-side UUID is safe to reuse across
different endpoints — no cross-route collisions.

## 6. Task 4 — Audit middleware

**File**: `internal/middleware/audit.go` (new).

Pipeline: **async** — every audited request enqueues an `AuditEvent`
into a buffered channel (size 100). A single goroutine drains the
channel in 2-second ticks (or 16-event batches) and does a
multi-row parameterized INSERT. If the channel is full we DROP the
oldest pending event and bump `cdc_audit_log_dropped_total`; stale
events have less value than fresh ones for security response.

Contract enforced by the middleware:
- `reason` field (JSON body) must be >= 10 chars → otherwise 400 and
  the handler is NOT invoked (saves a wasted NATS publish).
- `payload` is capped at 64 KiB; larger bodies truncated with a
  `"<truncated>"` marker to protect Postgres.
- `user_agent` capped at 512 chars.
- `result` records the final status code and up to 1 KiB of the
  response body on errors — useful for forensics when a destructive
  call silently 500s.

Action name: derived via an `ActionMap` (route pattern → canonical
action). Unknown routes fall back to `<method>_<path>` so nothing is
silently un-audited.

## 7. Task 5 — Route wiring

**File**: `internal/router/router.go` + `internal/server/server.go`
(updated to wire audit logger + destructive middleware bundle).

Routes now gated by the full stack:

| Route | Chain |
|-------|-------|
| POST /api/reconciliation/check | Ops-Admin → Idem → Audit |
| POST /api/reconciliation/check/:table | Ops-Admin → Idem → Audit |
| POST /api/reconciliation/heal/:table | Ops-Admin → Idem → Audit |
| POST /api/failed-sync-logs/:id/retry | Ops-Admin → Idem → Audit |
| POST /api/tools/reset-debezium-offset | Ops-Admin → Idem → Audit |
| POST /api/tools/trigger-snapshot/:table | Ops-Admin → Idem → Audit |
| POST /api/tools/restart-debezium | Ops-Admin → **RateLimit(3/h)** → Idem → Audit |

**Fiber quirk discovered**: `apiGroup.Group("", mw)` calls
`grp.app.register(methodUse, prefix, ...)` which installs Use-style
middleware on the PARENT group and leaks onto all subsequent handlers.
The existing router used `shared := apiGroup.Group("", RequireRole("admin","operator"))`
and `admin := apiGroup.Group("", RequireRole("admin"))`; any destructive
routes mounted AFTER these would inherit the admin|operator gate and
reject `ops-admin` tokens.

Fix: register destructive routes BEFORE the shared/admin groups, and
pass middleware as per-route handlers rather than via a Group-with-Use.
Helpers `registerDestructive` / `registerDestructiveRestart` encapsulate
this so the call sites stay readable. Documented the quirk in router.go
so the next engineer doesn't re-introduce the bug.

## 8. Task 6 — Rate limiter

**File**: `internal/middleware/ratelimit.go` (new).

Redis-backed counter (`INCR` + `EXPIRE` on first increment). Per-user
key `ratelimit:<scope>:<username>`. Only applied to
`/api/tools/restart-debezium` for now (`Max=3, Window=1h`) since
connector restart has the highest blast radius. Emits `Retry-After`
header on 429.

If Redis is unreachable the limiter **fails closed** (503) rather than
silently allowing a flood — same principle as the idempotency layer.

## 9. Unit tests

Location: `internal/middleware/*_test.go`. Run: `go test ./internal/middleware/...`.

- `TestRBACForbids403` — viewer role → 403.
- `TestRBACAllowsOpsAdmin` — ops-admin → 200.
- `TestRBACAdminBackCompat` — legacy admin role still passes.
- `TestRBACAdminUsersFallback` — env `ADMIN_USERS=ops1,ops2` bypass.
- `TestRBACNoAuth` — missing locals → 401.
- `TestIdempotencyMissingHeader` — no header → 400.
- `TestIdempotencyReplay` — 2nd call returns cached body, handler ran
  exactly once.
- `TestIdempotencyConflict` — held lock → 409.
- `TestIdempotencyFailNotCached` — 500 response not cached.
- `TestIdempotencyInvalidKey` — unsafe chars → 400.
- `TestAuditLogInsert` — actual row inserted into `admin_actions`
  (integration test; skips if Postgres unavailable).
- `TestAuditRejectsShortReason` — reason < 10 chars → 400.
- `TestRateLimitRestart` — 4th call → 429 with Retry-After.

Full `go test ./...` also green for `internal/api` and `internal/service`.

## 10. Runtime verification (curl evidence)

JWT tokens generated with HS256 against `change-me-in-production`:
- VIEWER = role=viewer
- OPS = role=ops-admin

```
1) No JWT                 → HTTP=401 {"error":"missing authorization header"}
2) Viewer role            → HTTP=403 {"error":"forbidden","required_roles":["ops-admin","admin"]}
3) ops-admin, no Idem     → HTTP=400 {"error":"missing Idempotency-Key header"}
4) ops-admin, short reason→ HTTP=400 {"error":"missing or too-short `reason`","min_length":10}
5) ops-admin, full stack  → HTTP=202 {"message":"heal dispatched","table":"test_table"}
6) Replay same Idem-Key   → HTTP=200 {"message":"heal dispatched","table":"test_table"}
                             Header: X-Idempotent-Replay: true
7) admin_actions row      → (ops-dan, heal, test_table, success, heal-run-...abc1, "phase 4 runtime...")
8) Rate-limit 4th restart → HTTP=429 {"error":"rate limit","max":3,"retry_after":3599,...}
```

## 11. Security self-audit

| Threat | Mitigation |
|--------|-----------|
| SQL injection via payload | Parameterized INSERT (`?` placeholders, `::jsonb` cast). Never concatenates user bytes into SQL. |
| SQL injection via reason/target | Same — all values pass through GORM parameter binding. |
| Header injection via Idempotency-Key | Whitelist `[A-Za-z0-9._-]` only, length 8-128; explicit reject of `:` / whitespace. Key is also namespaced by route so cross-route collisions are impossible. |
| Redis key collision | Namespace `idem:*` and `ratelimit:*` — separate from `system_health:snapshot` used by Phase 0 collector. Keys include route pattern + user. |
| Replay attack (network retry) | Idempotency-Key TTL 1h — identical replays serve cached response, no double execution. |
| Double execution (concurrent) | SETNX lock with 30s TTL; concurrent second request gets 409. |
| Leaked JWT used by attacker | Viewer token cannot reach destructive routes (403 at RBAC). ADMIN_USERS env fallback is an operational bypass; documented TODO to remove. |
| DB overload via oversized payload | `payload` capped at 64 KiB, `user_agent` at 512, `target` at 256; reason minimum 10 chars enforced at handler. |
| Audit log flood | Bounded channel (100) + drop-oldest + `cdc_audit_log_dropped_total` metric hook. |
| Redis outage masking destructive calls | Idempotency + RateLimit fail-closed (503). No silent fallthrough. |

## 12. Known limitations / TODOs

- **Plan routes not yet implemented in this service**: `/api/recon/heal`
  (generic), `/api/connectors/:name/restart` (generic), `/api/debezium/signal`
  (generic), `/api/kafka/reset-offset`. Current service has only specific
  variants (`/reconciliation/heal/:table`, `/tools/restart-debezium`,
  `/tools/reset-debezium-offset`). Documented via TODO in router.go.
- **RBAC back-compat**: `RequireOpsAdmin()` accepts `admin` role until
  IdP emits `ops-admin`. Tighten post-rollout.
- **ADMIN_USERS env fallback**: temporary — delete once the IdP issues
  real role claims to every operator.
- **Metric wiring**: `cdc_audit_log_dropped_total` is exposed on the
  AuditLogger struct (DroppedCount) but not yet registered with the
  Prometheus collector. Wire in observability package when the Phase 6
  metrics bundle is refactored.
- **Single-row cached response**: idempotency caches only the body, not
  the status code. Replays always return 200 even if the original was
  202. For the FE this is indistinguishable (same body); low priority
  but worth revisiting if we ever differentiate 200/202 semantics.

## 13. Files changed

Added:
- `cdc-cms-service/migrations/005_admin_actions.sql`
- `cdc-cms-service/internal/middleware/rbac.go`
- `cdc-cms-service/internal/middleware/rbac_test.go`
- `cdc-cms-service/internal/middleware/idempotency.go`
- `cdc-cms-service/internal/middleware/idempotency_test.go`
- `cdc-cms-service/internal/middleware/audit.go`
- `cdc-cms-service/internal/middleware/audit_test.go`
- `cdc-cms-service/internal/middleware/ratelimit.go`
- `cdc-cms-service/internal/middleware/ratelimit_test.go`

Modified:
- `cdc-cms-service/internal/middleware/jwt.go` (expose `roles` claim)
- `cdc-cms-service/pkgs/rediscache/redis_client.go` (SetNX/Incr/Expire/TTL/Client)
- `cdc-cms-service/internal/router/router.go` (destructive chain wiring)
- `cdc-cms-service/internal/server/server.go` (audit lifecycle)
