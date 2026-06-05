# 09 Tasks Solution — Ghost Collection

## Code/Config diff demo

### docker-compose.yml — revert
```diff
- confluent-hub install --no-prompt debezium/debezium-connector-mongodb:2.7.4
- confluent-hub install --no-prompt debezium/debezium-connector-postgresql:2.7.4
- confluent-hub install --no-prompt debezium/debezium-connector-mysql:2.7.4
+ confluent-hub install --no-prompt debezium/debezium-connector-mongodb:2.5.4
+ confluent-hub install --no-prompt debezium/debezium-connector-postgresql:2.5.4
+ confluent-hub install --no-prompt debezium/debezium-connector-mysql:2.5.4
```

### Connector config — add ghost collection
```jsonc
{
  // ...existing keys...
  "topic.prefix": "cdc.goopay",
  "collection.include.list": "centralized-export-service.export-jobs",
  "signal.enabled.channels": "source,kafka",
  "signal.kafka.topic": "cdc.signal.commands",
  "signal.kafka.bootstrap.servers": "kafka:9092",
  "signal.kafka.group.id": "debezium-signal-goopay-local",
  // NEW — fixes Bug C (NPE in emitWindowOpen)
  "signal.data.collection": "cdc_system.debezium_watermarks"
}
```

## Shell commands (sequential)

```bash
# G2: recreate kafka-connect
docker compose -f /Users/trainguyen/Documents/work/data-hub/centralized-data-service/docker-compose.yml rm -sfv kafka-connect
docker compose -f /Users/trainguyen/Documents/work/data-hub/centralized-data-service/docker-compose.yml up -d kafka-connect

# G2 poll plugins available
until curl -fsS http://127.0.0.1:18083/connector-plugins 2>/dev/null | grep -q MongoDbConnector; do
  echo "wait plugin..."; sleep 5;
done

# G3 create ghost collection
docker exec gpay-mongo mongosh --quiet --eval "\
  db.getSiblingDB('cdc_system').createCollection('debezium_watermarks');\
  print('OK collections:'); printjson(db.getSiblingDB('cdc_system').getCollectionNames());\
  print('count:'); print(db.getSiblingDB('cdc_system').debezium_watermarks.countDocuments({}));\
"

# G4 PATCH goopay-local
curl -s http://127.0.0.1:18083/connectors/goopay-local/config \
  | jq '. + {"signal.data.collection":"cdc_system.debezium_watermarks"} | del(.name)' \
  > /tmp/goopay-local.cfg.json
curl -s -o /tmp/put-resp.json -w "HTTP %{http_code}\n" \
  -X PUT -H 'Content-Type: application/json' \
  --data-binary @/tmp/goopay-local.cfg.json \
  http://127.0.0.1:18083/connectors/goopay-local/config

# G5 restart
curl -s -X POST "http://127.0.0.1:18083/connectors/goopay-local/restart?includeTasks=true&onlyFailed=false"
# wait running
until curl -fsS http://127.0.0.1:18083/connectors/goopay-local/status | jq -e '.connector.state=="RUNNING" and (.tasks|length>0) and (.tasks[0].state=="RUNNING")' >/dev/null; do
  echo "wait running..."; sleep 3;
done

# G6 capture shadow BEFORE
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -tAc \
  "select count(*) from shadow_goopay.sd_export_jobs_local;" | tee /tmp/shadow-before.txt

# G7 capture source Mongo
docker exec gpay-mongo mongosh --quiet --eval \
  "print(db.getSiblingDB('centralized-export-service').getCollection('export-jobs').countDocuments({}))" \
  | tee /tmp/source-count.txt

# G8 trigger via NATS
docker run --rm --network cdc-bridge natsio/nats-box:latest \
  nats --server 'nats://cdc_worker:worker_secret_2026@nats:4222' \
  pub cdc.cmd.debezium-snapshot \
  '{"type":"incremental","database":"centralized-export-service","collection":"export-jobs","table":"sd_export_jobs_local","trace_id":"ghost-collection-verify","action":"snapshot_now","origin":"verify"}'

# G9 wait + capture Connect log
sleep 60
docker logs gpay-kafka-connect --since 90s 2>&1 \
  | grep -E "Requested|NullPointerException|emitWindowOpen|window opened|window closed|debezium_watermarks" \
  | tee /tmp/connect-after.log

# G10 capture shadow AFTER
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -tAc \
  "select count(*) from shadow_goopay.sd_export_jobs_local;" | tee /tmp/shadow-after.txt

# Verify delta
echo "delta = $(($(cat /tmp/shadow-after.txt) - $(cat /tmp/shadow-before.txt)))"
```
