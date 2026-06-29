# Plan: Runtime Check for DB and API Gateway

## Steps

### Phase 1: Configuration Audit & Port Discovery
- [ ] Scan directory `cdc-cms-service` for configuration files (`config.yaml`, `.env`, `internal/config/`, etc.)
- [ ] Identify the HTTP gateway port configured for `cdc-cms-service`.

### Phase 2: DB Query Check (cdc_recon_smoke_result)
- [ ] Connect to the DB using `psql` (since we have `unsandboxed(psql)` permission) or a scratch script.
- [ ] Query: `SELECT COUNT(*) FROM cdc_system.cdc_recon_smoke_result;`
- [ ] Document the initial record count.

### Phase 3: Trigger Reconciliation Smoke Check (if empty)
- [ ] Check if the count is 0. If yes, proceed to trigger.
- [ ] Find command trigger endpoint or payload: either `POST /api/reconciliation/check` or NATS command.
- [ ] Execute `curl` trigger command.
- [ ] Re-query the DB to confirm if records are populated.

### Phase 4: API Verification
- [ ] Call `GET http://localhost:<port>/api/reconciliation/report`.
- [ ] Analyze the returned data or error.

### Phase 5: Final Report
- [ ] Document findings in a report.
