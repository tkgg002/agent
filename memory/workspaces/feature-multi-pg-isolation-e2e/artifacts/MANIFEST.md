# Phase D Artifact Manifest

**Phase**: Track D Hardening + Phase D Auto-Pipeline E2E
**Status**: 🟢 ARCHITECT APPROVED — closed 2026-04-29
**DoD evidence**: source 26 (`orders_e2e_d_v5`) — single REST `/advance` → state=`running` ở T+34s autonomously, 8 step log entries, JobMonitor close-loop confirmed.

## Binaries

| File | Size | sha256 |
|---|---|---|
| `cdc-worker_phaseD_done_2026-04-29` | 50,046,306 B | `592d56407eaf1bbf10092b67cd824adfae905ff4c8b522336c5d1909d1b95d6c` |
| `cdc-cms_phaseD_done_2026-04-29`    | 57,502,626 B | `891cdbc14cfcd7c7896af8b096aff7e6f14f40619bc817b6a909c25161e5835f` |

## Build environment

- Host: macOS 26.0.1 (Build 25A362), darwin/arm64
- Toolchain: go1.26.1
- Build commands:
  - Worker: `cd centralized-data-service && go build -o /tmp/cdc-worker ./cmd/worker`
  - CMS:    `cd cdc-cms-service && go build -o /tmp/cdc-cms ./cmd/server`

## Runtime config (env vars required to reproduce E2E)

```
PROVISIONING_ORCHESTRATOR_ENABLED=1
PROVISIONING_DEFAULT_MASTER_CONNECTION_CODE=master_local_pg_dest
PROVISIONING_DEFAULT_SHADOW_CONNECTION_CODE=shadow_local_pg_cdc
# Optional override for transmute scheduler tick:
# PROVISIONING_DEFAULT_CRON_EXPR='*/1 * * * *'
```

Worker MUST be started from `centralized-data-service/` cwd (viper loads `config/config-local.yml` relative to cwd — non-positive-NewTicker panic if config missing).

CMS MUST be started from `cdc-cms-service/` cwd. Listens on :8083; auth via JWT HS256 with secret `change-me-in-production` and role=`admin` (NOT `ops-admin`). Destructive endpoints require `Idempotency-Key` header AND `reason >= 10 chars`.

## Re-test recipe

```bash
# 1. Start worker (from centralized-data-service cwd):
cd /Users/trainguyen/Documents/work/cdc-system/centralized-data-service
PROVISIONING_ORCHESTRATOR_ENABLED=1 \
  PROVISIONING_DEFAULT_MASTER_CONNECTION_CODE=master_local_pg_dest \
  PROVISIONING_DEFAULT_SHADOW_CONNECTION_CODE=shadow_local_pg_cdc \
  /path/to/artifacts/cdc-worker_phaseD_done_2026-04-29 &

# 2. Start CMS (from cdc-cms-service cwd):
cd /Users/trainguyen/Documents/work/cdc-system/cdc-cms-service
PROVISIONING_DEFAULT_MASTER_CONNECTION_CODE=master_local_pg_dest \
  /path/to/artifacts/cdc-cms_phaseD_done_2026-04-29 &

# 3. Generate ops-admin JWT:
python3 -c "
import hmac,hashlib,base64,json,time
b=lambda x: base64.urlsafe_b64encode(x).rstrip(b'=').decode()
h=b(json.dumps({'alg':'HS256','typ':'JWT'},separators=(',',':')).encode())
p=b(json.dumps({'sub':'retest','role':'admin','iat':int(time.time()),'exp':int(time.time())+3600},separators=(',',':')).encode())
s=b(hmac.new(b'change-me-in-production',f'{h}.{p}'.encode(),hashlib.sha256).digest())
print(f'{h}.{p}.{s}')
"

# 4. Create source draft + advance once:
docker exec -i gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c "
  INSERT INTO cdc_system.source_object_registry
    (object_code, source_connection_id, source_engine_type, source_database, source_schema,
     source_object_name, source_object_type, normalized_source_key, primary_key_field, primary_key_type,
     cdc_mode, sync_engine, is_active, profile_status, provisioning_mode, provisioning_state)
  VALUES ('retest_phaseD', 4, 'postgresql', 'goopay_source', 'public',
          'orders_retest', 'table', 'pg.goopay_source.public.orders_retest',
          'id', 'BIGINT', 'incremental', 'debezium', true, 'active', 'auto', 'draft')
  RETURNING id;"

curl -X POST http://localhost:8083/api/v1/cms/sources/<ID>/provisioning/advance \
  -H "Authorization: Bearer <JWT>" \
  -H "Idempotency-Key: retest-$(date +%s)" \
  -d '{"actor":"retest","reason":"phaseD-retest-from-archived-binary"}'

# 5. Poll: SELECT provisioning_state FROM cdc_system.source_object_registry WHERE id=<ID>;
#    Expected: state=running ở T+30~60s (depending on scheduler tick alignment).
```

## Modified files (frozen at this build)

- `centralized-data-service/internal/service/schema_adapter.go` — pgx.Identifier
- `centralized-data-service/internal/service/job_monitor.go` — Q3 impacted_sources log
- `centralized-data-service/internal/service/provisioning_orchestrator.go` — seedMasterBindingForAdvance + lookupSourceTableForSource + discover payload extras
- `centralized-data-service/internal/handler/provisioning_step_handlers.go` — JOIN connection_registry + upsertShadowBinding rewrite + HandleScheduleEnable rewrite
- `cdc-cms-service/internal/service/provisioning_orchestrator.go` — Option A seed + Q5 Resume + lookupSourceTableForSource + discover payload extras

## Cleanup note

Binaries archived here are FROZEN — do not use for Track E development. Track E will produce its own artifacts. If Phase D regression is suspected, reproduce with these exact binaries first to isolate variable.
