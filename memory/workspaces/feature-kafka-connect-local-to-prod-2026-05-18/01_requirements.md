# 01_requirements.md — Functional + Non-functional + DoD

## Functional Requirements

### Common (apply mọi interpret H1/H2)

| R# | Requirement | Acceptance |
|---|---|---|
| R1 | Có 1 file `.env.local` (gitignored) override `KAFKA_BROKERS`, `KAFKA_SCHEMA_REGISTRY_URL`, `KAFKA_CONNECT_URL` về prod URLs | `docker compose config` cho thấy ENV resolved về URL prod |
| R2 | Connector JSON KHÔNG hardcode service-local URL → dùng `${env:VAR}` qua `EnvVarConfigProvider` | grep file JSON không còn literal `gpay-schema-registry:8081` |
| R3 | Local script `register_*.sh` accept `CONNECT_URL` ENV override → POST tới prod | `CONNECT_URL=<prod> bash register_pg_source.sh` đăng ký được connector |
| R4 | Topic naming có namespace dev (`cdc.dev-<user>.*`) tránh collide topic prod | `kafka-topics --list` thấy topic `cdc.dev-traingn.*` tách biệt |
| R5 | Có doc 1 trang ghi rõ: URL prod nào trỏ đâu, cert path, auth credentials lookup | `report_kafka_connect_local2prod_2026-05-18.md` ở root repo |

### H1-specific (Local stack → Prod Kafka)

| R# | Requirement |
|---|---|
| R1.1 | `cdc-worker` local consume được 1 topic test trên Kafka broker prod |
| R1.2 | `cdc-worker` local publish được 1 message thử lên topic `cdc.dev-<user>.test` |
| R1.3 | Auth SASL/SCRAM hoặc mTLS được config qua ENV (không hardcode trong code) |
| R1.4 | Cert CA (nếu TLS) load từ file path bên ngoài git (vd `~/.config/cdc/ca.crt`) |
| R1.5 | Toggle dễ giữa local-only mode và hybrid mode bằng 1 ENV flag (`CDC_TARGET=local|prod`) |

### H2-specific (Local POST connector → Prod Connect REST)

| R# | Requirement |
|---|---|
| R2.1 | Có cách reach Connect REST prod từ local (`kubectl port-forward` hoặc Ingress URL) |
| R2.2 | Script `register_pg_source.sh` tham số hoá `CONNECT_URL` qua ENV, default local |
| R2.3 | Connector JSON dùng `${env:VAR}` resolve qua `EnvVarConfigProvider` của Kafka Connect worker (cần worker prod config provider) |
| R2.4 | Đăng ký connector ở prod KHÔNG break connector đang RUNNING (dùng tên unique `cdc-pg-source-dev-<user>`) |
| R2.5 | Có lệnh teardown rollback (`curl DELETE`) khi xong test |

## Non-functional Requirements

| N# | Requirement |
|---|---|
| N1 | Không leak credentials vào git (lesson L-1934). `.env.local` + cert path qua gitignore |
| N2 | Không bake URL prod vào image build (đã apply ở refactor cdc-cms-web Phase 3) |
| N3 | Minimal impact: chỉ đụng file dev config / docker-compose, KHÔNG đụng `src/` cdc-worker |
| N4 | Reversible: 1 lệnh chuyển về local-only mode (unset `.env.local` hoặc `CDC_TARGET=local`) |
| N5 | Observability: log rõ ràng worker đang nối broker nào (đã có `Kafka brokers=...` log trong cdc-worker) |
| N6 | Topic isolation: prefix `cdc.dev-<user>.*` tránh đụng topic prod live |
| N7 | Audit: APPEND vào `05_progress.md` mọi command chạy trên prod (CLAUDE.md §11) |

## Definition of Done

### H1 DoD
- [ ] `.env.local` template tạo (gitignored), document trong README
- [ ] `cdc-worker` local restart với ENV prod → log `Kafka brokers connected` với URL prod
- [ ] Smoke: produce 1 message test → consume back đúng nội dung → metrics emit
- [ ] Rollback: `unset CDC_TARGET; docker compose up cdc-worker` về local Kafka OK
- [ ] Report `report_*.md` ghi rõ verify command + output

### H2 DoD
- [ ] Script `register_pg_source.sh` accept `CONNECT_URL` ENV (default `localhost:18083`)
- [ ] Connector JSON 4 file refactor sang `${env:VAR}` syntax
- [ ] Smoke: `CONNECT_URL=<prod-via-port-forward> bash register_pg_source.sh` → connector state RUNNING trên prod
- [ ] Smoke: `curl GET <prod>/connectors/cdc-pg-source-dev-<user>/status` về RUNNING
- [ ] Teardown: `curl DELETE` xoá connector sau test, no orphan
- [ ] Report `report_*.md` đầy đủ verify evidence

### Bắt buộc trước khi user approve plan
- [ ] User xác nhận H1 / H2 / cả hai
- [ ] User cung cấp: bootstrap broker prod URL, Schema Registry URL, Connect REST URL, auth method
- [ ] User confirm có quyền chạy `kubectl` tới cluster prod (nếu H2 dùng port-forward)
- [ ] User confirm topic namespace dev OK (vd `cdc.dev-traingn.*`)
