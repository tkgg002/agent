# 00_context.md — Kafka Connect local → prod connection

## Trigger

User cung cấp 1 đoạn `Dockerfile` build image Kafka Connect cho prod (image này deploy lên k8s cluster prod, dùng Strimzi base):

```dockerfile
ENV DEBEZIUM_JDBC_VERSION=3.5.0.Final
ENV CONFLUENT_VERSION=7.6.0

# Plugin 1: MongoDB Official Kafka Connect (Fat JAR)
# Plugin 2: Debezium JDBC Connector
# Plugin 3: Debezium MongoDB Connector
# Plugin 4: Confluent Avro Converter (bundle via Maven)

FROM quay.io/strimzi/kafka:0.51.0-kafka-4.2.0
USER root:root
COPY --from=builder --chown=1001:0 /plugins /opt/kafka/plugins/
USER 1001
```

Quote nguyên văn: "đây là con kafka trên prod, tao đang muốn kết nối local của tao để chạy trên nó. mày lên plan xem sao."

## Hiểu yêu cầu (3 cách interpret — cần user xác nhận)

| # | Interpret | Mô tả | Khả thi |
|---|---|---|---|
| **H1** | Local Kafka stack → Prod Kafka | Local `centralized-data-service` (cdc-worker Go binary + 3 source DB local) trỏ `KAFKA_BROKERS` về broker prod thay vì `kafka:9092` local. | Khả thi nếu prod broker có external listener + auth credentials cho user. |
| **H2** | Local script POST connector → Prod Kafka Connect REST | Từ máy local user `curl` file JSON connector tới REST API của Connect prod (via `kubectl port-forward` hoặc Ingress). | Khả thi cao, ít rủi ro, không đụng data prod. |
| **H3** | Local source DB → Prod Connect đọc CDC | Connect prod đọc CDC từ PG/Mongo/MariaDB local của user. | KHÔNG khả thi an toàn — local DB không reach từ pod prod (firewall/network), cần expose DB local ra internet. |

## Stack landscape

### Local hiện tại (`data-hub/centralized-data-service/docker-compose.yml`)

10 container trên network `cdc-bridge`:
- `gpay-kafka` (KRaft, port 19092/19093 host, advertise `kafka:9092` internal)
- `gpay-schema-registry` (port 18081 host, advertise `schema-registry:8081` internal)
- `gpay-kafka-connect` (port 18083 host, advertise `kafka-connect:8083` internal)
- `gpay-cdc-worker` (Go binary, reads `KAFKA_BROKERS=kafka:9092`, `KAFKA_SCHEMA_REGISTRY_URL=http://schema-registry:8081`)
- `gpay-nats`, `gpay-postgres-cdc`, `gpay-redis`, `gpay-kafka-exporter`, `gpay-redpanda-console`, `gpay-otel-collector`

Connector JSON hardcode:
- `pg-source-connector.json`: `gpay-schema-registry:8081`, `postgres-source` DSN, `src_pass`
- `mongodb-connector.json`, `cdc-mariadb-source.json`, `connector-mongodb.json` (tương tự)
- Script `register_pg_source.sh` POST `localhost:18083/connectors`

### Prod (theo Dockerfile user paste)

Image Kafka Connect được build từ Strimzi base `quay.io/strimzi/kafka:0.51.0-kafka-4.2.0` — chứng tỏ cluster prod dùng **Strimzi Operator** (KafkaConnect CRD + KafkaConnector CRD).

Plugin có sẵn trong image prod:
1. **mongo-kafka-connect** (Sink, Official MongoDB driver)
2. **debezium-connector-jdbc** 3.5.0.Final (Sink JDBC, write change events vào RDBMS)
3. **debezium-connector-mongodb** (Source CDC từ Mongo)
4. **kafka-connect-avro-converter** (Confluent CP 7.6.0)

→ Plugin nguồn (Source) hiện chỉ có Debezium MongoDB. **Thiếu Debezium PostgreSQL + MySQL** so với local docker-compose.

Cluster prod cần info user clarify:
- Bootstrap brokers external URL (vd `bootstrap.kafka-prod.example.com:9094`)
- Schema Registry URL (Strimzi không gói SR — chắc deploy Apicurio hoặc Confluent CP-SR riêng)
- Connect REST URL (qua Ingress hoặc port-forward)
- Auth method (SASL/SCRAM, mTLS, OAuth)
- Topic naming convention prod (có namespace riêng tránh collide local dev?)

## Constraints

| Constraint | Source |
|---|---|
| Plan only, NO execute trước user approve | User: "lên plan xem sao" |
| Plan rõ ràng, có code demo tới chi tiết | User rule |
| Theo core /agent, GEMINI.md | User rule + CLAUDE.md §7 |
| Không cheat DB / bypass config | User rule + CLAUDE.md §6 |
| Verify thực tế trước báo done | User rule + CLAUDE.md §3 |
| Có file `report_*.md` | User rule + CLAUDE.md §7 |
| Workspace mới với prefix `feature-...` | CLAUDE.md §7 |

## Out of scope (sẽ explicit)

- KHÔNG đụng `src/` của cdc-worker (Go) trừ khi user explicit
- KHÔNG deploy/đụng cluster prod (chỉ propose, user/operator apply)
- KHÔNG bake credentials prod vào git (lesson L-1934)
- KHÔNG đề xuất full migration sang Strimzi cho local (out-of-scope, chỉ propose hybrid)
- KHÔNG sửa Dockerfile prod (file user paste — không nói rõ đường dẫn)

## Risks identified upfront

| Risk | Severity | Mitigation upfront |
|---|---|---|
| Local script POST connector lên prod accidentally → bẩn data prod | HIGH | Topic namespace `cdc.dev-<user>.*`; CONNECT_GROUP_ID prefix `dev-` |
| Network: local máy không reach prod broker (firewall/VPN) | HIGH | Đề xuất `kubectl port-forward` làm fallback |
| Auth: SASL/SCRAM credentials user không có | HIGH | Plan có step "yêu cầu credentials từ operator" |
| Plugin Postgres/MySQL thiếu trong image prod | MEDIUM | Document gap; user quyết extend Dockerfile hay không |
| Schema Registry compat: Apicurio vs Confluent CP-SR khác API endpoint | MEDIUM | Test smoke decode 1 message trước khi đăng ký connector |
| Local cdc-worker version mismatch broker prod | LOW | KRaft 4.2 broker tương thích Kafka client 3.6+ (Go sarama tested) |
