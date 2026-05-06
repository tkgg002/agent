# 09 — Tasks Solution — Phase 01 Split E2E (drafts)

> Đây là các draft kỹ thuật cho Muscle thực thi. Chi tiết subject to user approve.

## T-A1 + T-A2 — Docker Compose draft (delta)

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: gpay-postgres
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: auth_dw           # đổi tên (or keep goopay_dw)
    ports: ["5432:5432"]
    volumes: [pg_auth_data:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "user"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres-cdc:
    image: postgres:15-alpine
    container_name: gpay-postgres-cdc
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: cdc_dw
    ports: ["5433:5432"]
    volumes: [pg_cdc_data:/var/lib/postgresql/data]
    healthcheck: [...] 

  postgres-dest:
    image: postgres:15-alpine
    container_name: gpay-postgres-dest
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: goopay_dest
    ports: ["5434:5432"]
    volumes: [pg_dest_data:/var/lib/postgresql/data]

  postgres-source:
    image: postgres:15-alpine
    container_name: gpay-postgres-source
    command:
      - "postgres"
      - "-c"
      - "wal_level=logical"
      - "-c"
      - "max_wal_senders=10"
      - "-c"
      - "max_replication_slots=10"
    environment:
      POSTGRES_USER: srcuser
      POSTGRES_PASSWORD: srcpass
      POSTGRES_DB: goopay_source
    ports: ["5435:5432"]
    volumes:
      - pg_source_data:/var/lib/postgresql/data
      - ./cdc-source-test/sql:/docker-entrypoint-initdb.d:ro

volumes:
  pg_auth_data: {}
  pg_cdc_data: {}
  pg_dest_data: {}
  pg_source_data: {}
```

## T-B7 — Source DB seed draft

```sql
-- cdc-source-test/sql/init_source_local.sql
-- Auto-loaded by Postgres at first init via /docker-entrypoint-initdb.d/

CREATE TABLE IF NOT EXISTS public.orders (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL,
    amount       NUMERIC(15,2) NOT NULL,
    status       VARCHAR(32) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE public.orders REPLICA IDENTITY FULL;

CREATE TABLE IF NOT EXISTS public.users (...);
CREATE TABLE IF NOT EXISTS public.payments (...);

INSERT INTO public.orders (user_id, amount, status) VALUES
  (1, 100.00, 'paid'),
  (2, 250.50, 'pending'),
  ... (10 rows);
INSERT INTO public.users ... (10 rows);
INSERT INTO public.payments ... (10 rows);
```

## T-C2 + T-C3 + T-C4 — Multi-DSN config draft

```yaml
# centralized-data-service/config/config-local.yml
database:
  control_plane:
    host: localhost
    port: 5433
    database: cdc_dw
    user: user
    password: password
    sslmode: disable
  destination:
    host: localhost
    port: 5434
    database: goopay_dest
    user: user
    password: password
    sslmode: disable
```

```go
// pkgs/database/multi.go (NEW)
package database

import (
    "fmt"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type Multi struct {
    conns map[string]*gorm.DB
}

func NewMulti(cfgs map[string]Config) (*Multi, error) {
    m := &Multi{conns: make(map[string]*gorm.DB)}
    for name, cfg := range cfgs {
        dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
            cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)
        db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
        if err != nil { return nil, fmt.Errorf("open %s: %w", name, err) }
        m.conns[name] = db
    }
    return m, nil
}

func (m *Multi) Get(name string) *gorm.DB {
    return m.conns[name]
}
```

## T-D2 — Debezium connector register script draft

```bash
#!/usr/bin/env bash
# deployments/connect/register_pg_source.sh
set -euo pipefail

CONNECT_URL="${CONNECT_URL:-http://localhost:18083}"
CONN_NAME="goopay-source-pg"

curl -sS -X POST "$CONNECT_URL/connectors" \
  -H "Content-Type: application/json" \
  -d @- <<JSON
{
  "name": "$CONN_NAME",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres-source",
    "database.port": "5432",
    "database.user": "srcuser",
    "database.password": "srcpass",
    "database.dbname": "goopay_source",
    "database.server.name": "goopay_source",
    "topic.prefix": "cdc.goopay.source",
    "plugin.name": "pgoutput",
    "publication.autocreate.mode": "filtered",
    "table.include.list": "public.orders,public.users,public.payments",
    "snapshot.mode": "initial",
    "key.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "key.converter.schemas.enable": "false",
    "value.converter.schemas.enable": "false"
  }
}
JSON

# Wait until RUNNING
for i in {1..30}; do
  status=$(curl -sS "$CONNECT_URL/connectors/$CONN_NAME/status" | jq -r '.connector.state')
  if [ "$status" = "RUNNING" ]; then
    echo "Connector $CONN_NAME RUNNING"
    exit 0
  fi
  sleep 1
done
echo "Timeout waiting for connector RUNNING"
exit 1
```

## T-E1 — E2E test script draft

```bash
#!/usr/bin/env bash
# scripts/e2e_test_split.sh
set -euo pipefail

# D1: 4 PG containers up
for c in gpay-postgres gpay-postgres-cdc gpay-postgres-dest gpay-postgres-source; do
  docker inspect -f '{{.State.Health.Status}}' "$c" | grep -q healthy || { echo "FAIL D1: $c not healthy"; exit 1; }
done
echo "PASS D1"

# D2: auth login
TOKEN=$(curl -sS -X POST http://localhost:8081/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.access_token')
[ -n "$TOKEN" ] && [ "$TOKEN" != "null" ] || { echo "FAIL D2"; exit 1; }
echo "PASS D2"

# D3: cms healthcheck
curl -sS http://localhost:8083/api/v1/system/connectors -H "Authorization: Bearer $TOKEN" | jq . > /dev/null || exit 1
echo "PASS D3"

# D4: worker logs clean
errs=$(grep -ciE "error|fatal|panic" /tmp/worker.log || true)
[ "$errs" -lt 5 ] || { echo "FAIL D4: $errs errors"; exit 1; }
echo "PASS D4"

# D5: cdc_internal=0 trên 3 DB
for port in 5432 5433 5434; do
  n=$(docker exec -i gpay-postgres psql -U user -d auth_dw -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name='cdc_internal'" 2>/dev/null || echo 0)
done
echo "PASS D5"

# D6-D10: source insert + shadow + transmute verify
docker exec -i gpay-postgres-source psql -U srcuser -d goopay_source -c \
  "INSERT INTO public.orders (user_id, amount, status) VALUES (999, 1234.56, 'test_e2e');"
sleep 4
src=$(docker exec -i gpay-postgres-source psql -U srcuser -d goopay_source -tAc "SELECT count(*) FROM public.orders WHERE status='test_e2e'")
shd=$(docker exec -i gpay-postgres-cdc psql -U user -d cdc_dw -tAc "SELECT count(*) FROM shadow_goopay_source.orders WHERE status='test_e2e'")
[ "$src" = "$shd" ] || { echo "FAIL D9: src=$src shd=$shd"; exit 1; }
echo "PASS D9"

curl -sS -X POST http://localhost:8083/api/v1/transmute/run -H "Authorization: Bearer $TOKEN" -d '{"binding":"orders"}'
sleep 3
dw=$(docker exec -i gpay-postgres-dest psql -U user -d goopay_dest -tAc "SELECT count(*) FROM dw_orders.orders WHERE status='test_e2e'")
[ "$dw" = "$src" ] || { echo "FAIL D10: dw=$dw src=$src"; exit 1; }
echo "PASS D10"

echo "=== ALL DoD PASS ==="
```

## Notes
- Tất cả các draft trên là **starting point**. Khi Muscle thực thi sẽ adapt theo code thực tế.
- Decision pending: tên DB cho `gpay-postgres` (giữ `goopay_dw` hay đổi `auth_dw`)?
- Decision pending: migration directory split (subfolder vs flag) — recommend subfolder.
