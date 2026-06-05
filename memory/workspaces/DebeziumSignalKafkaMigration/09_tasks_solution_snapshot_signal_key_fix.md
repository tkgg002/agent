# 09 — Solution diff: snapshot-signal-kafka-key-fix (2026-05-20)

## T2 — CMS `injectDebeziumSignalDefaults` force-overwrite

**File**: `cdc-cms-service/internal/api/system_connectors_handler.go`

```diff
 func (h *SystemConnectorsHandler) injectDebeziumSignalDefaults(connectorName string, cfg map[string]string) {
     if h.signalBootstrap == "" {
         return
     }
     if !strings.HasPrefix(cfg["connector.class"], "io.debezium.") {
         return
     }
-    defaults := map[string]string{
+    // signal.* keys are infrastructure config owned by the backend, NOT
+    // user/FE input. Force-overwrite to defend against Vite placeholder
+    // leakage (e.g. "__VITE_SIGNAL_KAFKA_TOPIC__") and against operator
+    // typos. Operator-supplied signal.* keys are silently replaced.
+    overrides := map[string]string{
         "signal.enabled.channels":        "source,kafka",
         "signal.kafka.topic":             h.signalTopic,
         "signal.kafka.bootstrap.servers": h.signalBootstrap,
         "signal.kafka.group.id":          "debezium-signal-" + connectorName,
     }
-    for k, v := range defaults {
-        if _, set := cfg[k]; !set {
-            cfg[k] = v
-        }
+    for k, v := range overrides {
+        if old, set := cfg[k]; set && old != v {
+            h.logger.Warn("overwriting signal.* key on debezium connector",
+                zap.String("connector", connectorName),
+                zap.String("key", k),
+                zap.String("from", old),
+                zap.String("to", v))
+        }
+        cfg[k] = v
     }
 }
```

## T3 — Worker ResolveTopicPrefix + key fix

**File**: `centralized-data-service/internal/service/debezium_signal.go`

```diff
+// ResolveTopicPrefix fetches the connector's topic.prefix via Kafka
+// Connect REST. Debezium 2.5+ KafkaSignalChannel filters incoming
+// signal records by matching message key against topic.prefix — a
+// mismatched key is silently dropped (no log, no error). Returns
+// the resolved prefix or an error when the connector is missing /
+// topic.prefix is unset.
+func (d *DebeziumSignalClient) ResolveTopicPrefix(ctx context.Context, connectorName string) (string, error) {
+    if d.cfg.KafkaConnectBaseURL == "" {
+        return "", fmt.Errorf("kafka connect base URL not configured")
+    }
+    if connectorName == "" {
+        return "", fmt.Errorf("connector name required")
+    }
+    url := strings.TrimRight(d.cfg.KafkaConnectBaseURL, "/") + "/connectors/" + connectorName + "/config"
+    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
+    if err != nil {
+        return "", fmt.Errorf("build config request: %w", err)
+    }
+    resp, err := d.httpClient.Do(req)
+    if err != nil {
+        return "", fmt.Errorf("fetch connector config: %w", err)
+    }
+    defer resp.Body.Close()
+    if resp.StatusCode != http.StatusOK {
+        return "", fmt.Errorf("connector config HTTP %d", resp.StatusCode)
+    }
+    var cfg map[string]string
+    if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
+        return "", fmt.Errorf("parse connector config: %w", err)
+    }
+    prefix := strings.TrimSpace(cfg["topic.prefix"])
+    if prefix == "" {
+        return "", fmt.Errorf("connector %s has empty topic.prefix", connectorName)
+    }
+    return prefix, nil
+}

 func (d *DebeziumSignalClient) TriggerIncrementalSnapshot(
     ctx context.Context,
-    engine, database, collection, filter string,
+    connectorName, engine, database, collection, filter string,
 ) (string, error) {
     if !d.IsConfigured() {
         return "", fmt.Errorf("debezium signal: kafka writer not configured")
     }
     if database == "" || collection == "" {
         return "", fmt.Errorf("debezium signal: database and collection required")
     }
+    if connectorName == "" {
+        return "", fmt.Errorf("debezium signal: connector name required (needed to resolve topic.prefix for signal key)")
+    }

+    topicPrefix, err := d.ResolveTopicPrefix(ctx, connectorName)
+    if err != nil {
+        return "", fmt.Errorf("resolve topic.prefix for %s: %w", connectorName, err)
+    }

     qualified := database + "." + collection
     // ...payload build unchanged...

-    // Key = qualified collection name. Debezium does not use the key for
-    // routing in Kafka signal channel; it is set for partition affinity +
-    // debugging only. Every connector reads every message and self-filters
-    // by matching the `data-collections` payload field.
+    // Key MUST equal connector's topic.prefix. Debezium 2.5+
+    // KafkaSignalChannel silently drops messages where key != topic.prefix
+    // (see io.debezium.pipeline.signal.channels.KafkaSignalChannel#process).
+    // Mismatched key = no snapshot, no log entry, no error — only symptom
+    // is "snapshot signal published, no rows produced".
     msg := kafka.Message{
-        Key:   []byte(qualified),
+        Key:   []byte(topicPrefix),
         Value: body,
         Time:  time.Now(),
     }
     // ...
+    d.logger.Info("debezium signal published",
+        zap.String("topic", d.cfg.SignalKafkaTopic),
+        zap.String("connector", connectorName),
+        zap.String("topic_prefix", topicPrefix),
+        zap.String("engine", engine), ...)
 }
```

## T4 — Update callers

**File**: `centralized-data-service/internal/handler/recon_handler.go:~344`

```diff
- engine := service.ResolveEngineTypeBySource(ctx, h.db, db, collection)
- signalID, err := h.signal.TriggerIncrementalSnapshot(ctx, engine, db, collection, payload.Filter)
+ connectorName, _ := service.ResolveConnectorNameBySource(ctx, h.db, db, collection)
+ engine := service.ResolveEngineTypeBySource(ctx, h.db, db, collection)
+ signalID, err := h.signal.TriggerIncrementalSnapshot(ctx, connectorName, engine, db, collection, payload.Filter)
```

**File**: `centralized-data-service/internal/service/recon_heal.go:~680` — tương tự.

## T5 — Migrate 2 connector existing

```bash
# goopay-local
curl -s http://127.0.0.1:18083/connectors/goopay-local/config | \
  jq '. + {"signal.kafka.topic": "cdc.signal.commands"}' | \
  curl -X PUT -H 'Content-Type: application/json' \
    --data-binary @- \
    http://127.0.0.1:18083/connectors/goopay-local/config

# goopay-dev (same pattern)
```

Verify post-PUT:
```bash
curl -s http://127.0.0.1:18083/connectors/goopay-{local,dev}/config | \
  jq '.["signal.kafka.topic"]'
# Expect: "cdc.signal.commands" on both
```

## T7 — End-to-end verify
```bash
# Before trigger
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT COUNT(*) FROM shadow_goopay.export_jobs"

# Source count
docker exec gpay-mongo mongosh \
  "mongodb://localhost:27017/centralized-export-service?replicaSet=rs0" \
  --eval 'db["export-jobs"].countDocuments({})'

# Trigger snapshot via UI

# Wait + check
docker logs gpay-kafka-connect --since 1m | grep -iE \
  "Requested snapshot|Snapshot ended|incremental"

# After
docker exec gpay-postgres-cdc psql -U gpay_admin -d cdc_dw -c \
  "SELECT COUNT(*) FROM shadow_goopay.export_jobs"
# Must equal source count
```
