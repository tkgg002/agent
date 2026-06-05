# 02 Plan — Debezium 2.7.4.Final manual install

## docker-compose command diff
```diff
    command:
      - bash
      - -c
      - |
-       confluent-hub install --no-prompt debezium/debezium-connector-mongodb:2.5.4
-       confluent-hub install --no-prompt debezium/debezium-connector-postgresql:2.5.4
-       confluent-hub install --no-prompt debezium/debezium-connector-mysql:2.5.4
+       set -euo pipefail
+       PLUGIN_DIR=/usr/share/confluent-hub-components
+       VERSION=2.7.4.Final
+       BASE=https://repo1.maven.org/maven2/io/debezium
+       mkdir -p "$$PLUGIN_DIR"
+       for c in mongodb postgres mysql; do
+         dst="$$PLUGIN_DIR/debezium-connector-$$c"
+         if [ -d "$$dst" ] && ls "$$dst" | grep -q "$$VERSION"; then
+           echo "[plugin] $$c $$VERSION already present, skip"; continue
+         fi
+         rm -rf "$$dst"
+         tarball="debezium-connector-$$c-$$VERSION-plugin.tar.gz"
+         url="$$BASE/debezium-connector-$$c/$$VERSION/$$tarball"
+         echo "[plugin] download $$url"
+         curl -fsSL "$$url" -o "/tmp/$$tarball"
+         tar -xzf "/tmp/$$tarball" -C "$$PLUGIN_DIR"
+         rm -f "/tmp/$$tarball"
+       done
+       echo "[plugin] installed, listing:"
+       ls "$$PLUGIN_DIR"
        /etc/confluent/docker/run
```

Notes:
- `$$` cần thiết trong docker-compose YAML để escape `$` (compose-only, không phải shell-only).
- Artifact name cho Postgres trên Maven = `postgres` (không `postgresql`); thư mục extract sẽ là `debezium-connector-postgres`. Connect tự discovery plugin class → KHÔNG cần đổi `plugin.class` config (connector tự dùng `io.debezium.connector.postgresql.PostgresConnector`).
- Mongo connector class `io.debezium.connector.mongodb.MongoDbConnector` không đổi giữa 2.5 → 2.7.
- Existing connector config trong Kafka `_connect-configs` reload tự động.

## Steps
1. Edit `docker-compose.yml` command theo diff trên.
2. `docker compose rm -sfv kafka-connect`.
3. `docker compose up -d kafka-connect` — wait container Up.
4. Monitor log container: `[plugin] download ...` cho cả 3 → `[plugin] installed`.
5. Poll REST `/connector-plugins` until 3 plugin có version `2.7.4.Final`.
6. Poll `/connectors/goopay-local/status` → RUNNING/RUNNING.
7. Capture shadow PG row count BEFORE.
8. NATS publish trigger snapshot (cmd cũ — worker code Bug A đã đúng).
9. Wait 60s, capture Connect log + shadow count AFTER.
10. Report.

## Risk + mitigation
| Risk | Mitigation |
|---|---|
| Connector 2.7.4 yêu cầu config key mới (deprecation) | Capture log validation, sửa config nếu cần |
| 2.7.4 vẫn NPE / cần signal.data.collection | Báo user kết quả thật, không cheat. Đây là điểm phải confront cho prod read-only. |
| Network fail khi tải Maven | Retry 3 lần, fallback domain `maven.aliyun.com/repository/public` (mirror) nếu cần |
| Tarball extract conflict tên thư mục cũ 2.5.4 | `rm -rf "$$dst"` trước extract đảm bảo clean |

## Rollback
Nếu 2.7.4 break tệ hơn 2.5.4:
- Edit compose về 2.5.4.Final URL Maven (tarball có sẵn) — vẫn dùng manual install, KHÔNG quay lại confluent-hub install.
- Recreate.
