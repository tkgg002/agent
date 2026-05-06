# 02_plan — Multi-Engine Unified Pipeline

> Đối ứng: `00_context_multi_engine_unified.md`, `01_requirements_multi_engine_unified.md`.
> Workflow tham chiếu: `agent/workflows/feature-dev.md` (Discovery → Exploration → Design → Implementation → QA).

## P1. Layer breakdown (3 layer + infra + e2e)

```
┌────────────────────────────────────────────────────────────────────┐
│ L4 infra: docker-compose gpay-mariadb + connector spec MariaDB     │
└────────────────────────────────────────────────────────────────────┘
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ L1 cdc-worker (Go):                                                │
│  • KafkaConfig.TopicPrefix string → []string (alias topicPrefixes)│
│  • kafka_consumer.discoverTopics: union các prefix                 │
│  • RegistryService: GetDebeziumNamespaces() multi-engine aware     │
│  • SmokeTest: load 3 prefix + 3 engine binding                     │
└────────────────────────────────────────────────────────────────────┘
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ L2 cms-api (Go):                                                   │
│  • GET /api/v1/cms/sources expose provisioning_mode/state/engine   │
│  • POST /api/v1/cms/sources/:id/provisioning/mode (đã có) verify  │
│  • Idempotency-Key middleware on /mode                            │
└────────────────────────────────────────────────────────────────────┘
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ L3 cms-fe (React+Vite):                                            │
│  • Type SourceObjectRow + 3 field mới                              │
│  • TableRegistry.tsx: column Engine + Mode + State                 │
│  • Hook useProvisioningMode (mutation + invalidate)                │
│  • Confirm dialog khi flip lúc *_pending                           │
└────────────────────────────────────────────────────────────────────┘
                                ▼
┌────────────────────────────────────────────────────────────────────┐
│ L5 E2E smoke 3 engines:                                            │
│  • PG (id=26) Manual→Auto kick                                     │
│  • Mongo (new row payment-bills) Auto end-to-end                   │
│  • MariaDB (new row legacy_orders) Auto end-to-end                 │
└────────────────────────────────────────────────────────────────────┘
```

## P2. Thứ tự thi công + lý do

| Step | Layer | Lý do thứ tự |
|------|-------|--------------|
| 1 | L1 cdc-worker (multi-prefix + multi-engine filter) | Smallest blast radius (1 file config + 2 service file). Build + unit test isolated. |
| 2 | L4 infra (MariaDB) | Cần MariaDB để smoke test L1 với engine thứ 3. Chỉ docker-compose + connector JSON; không deploy connector vội. |
| 3 | L2 cms-api (expose fields) | Đa số đã có; chỉ verify response shape + idempotency middleware. |
| 4 | L3 cms-fe (Toggle UI) | Phụ thuộc L2 response shape; phụ thuộc L1 worker behavior để E2E. |
| 5 | L5 E2E 3 engines | Cuối cùng — verify tổng. |

## P3. File touch matrix (predict)

| File | Change kind | Layer |
|------|-------------|-------|
| `centralized-data-service/config/config.go` | Edit (TopicPrefix []string + dual-decode) | L1 |
| `centralized-data-service/config/config-local.yml` | Edit (topicPrefixes list) | L1 |
| `centralized-data-service/internal/handler/kafka_consumer.go` | Edit (loop prefix discovery + namespace filter) | L1 |
| `centralized-data-service/internal/service/registry_service.go` | Edit (GetDebeziumNamespaces) | L1 |
| `centralized-data-service/internal/service/registry_service_test.go` | Edit/new (unit test) | L1 |
| `centralized-data-service/internal/handler/kafka_consumer_test.go` | Edit/new (multi-prefix test) | L1 |
| `centralized-data-service/docker-compose.yml` | Edit (add gpay-mariadb) | L4 |
| `centralized-data-service/deployments/connectors/cdc-mariadb-source.json` | New (Debezium MySQL connector cho MariaDB) | L4 |
| `centralized-data-service/migrations/cdc/049_mariadb_seed_legacy_orders.sql` | New (registry row + binding seed cho MariaDB) | L4 |
| `cdc-cms-service/internal/api/source_handler.go` (or current list handler) | Edit (response include 3 field) | L2 |
| `cdc-cms-service/internal/middleware/idempotency.go` | New (nếu chưa có) | L2 |
| `cdc-cms-service/internal/router/router.go` | Edit (mount Idempotency middleware on /mode) | L2 |
| `cdc-cms-web/src/types/index.ts` (hoặc tương đương) | Edit (SourceObjectRow + 3 field) | L3 |
| `cdc-cms-web/src/hooks/useProvisioningMode.ts` | New | L3 |
| `cdc-cms-web/src/pages/TableRegistry.tsx` | Edit (3 column + Switch + filter dropdown) | L3 |
| `cdc-cms-web/src/api/cmsApi.ts` (nếu cần) | Edit | L3 |

## P4. Risk + mitigation

| Risk | Mitigation |
|------|-----------|
| YAML scalar `topicPrefix:` trong env cũ → unmarshal lỗi sang `[]string` | Custom UnmarshalYAML/UnmarshalJSON: scalar → singleton list. Unit test 2 form. |
| Worker đang chạy production sẽ break khi reload | Worker hiện KHÔNG chạy (đã verify ps). Reload sau khi merge. |
| Mongo collection "orders" trùng tên PG table "orders" → registry filter collision | Bổ sung `GetDebeziumNamespaces()` trả full tuple `(engine, db, namespace, object)`. Filter chính xác. |
| MariaDB Debezium connector spec sai → snapshot stuck | Chỉ commit JSON spec; deploy thủ công sau code review. Document command deploy trong `09_tasks_solution`. |
| FE Toggle flip lúc orchestrator đang advance → race | API đã có CAS WHERE current_mode → 409 Conflict → FE show error + refresh. |
| `RequireOpsAdmin` chặn anonymous → smoke test cần token | Document curl với JWT từ `cdc-auth-service` trong `09_tasks_solution`. |

## P5. Definition of Done

1. `make build` (cms-service + centralized-data-service) PASS.
2. Unit test mới (worker + cms) PASS.
3. Worker boot với `config-local.yml` mới — log `kafka consumer started` thấy 3 prefix + ≥4 topic discovered.
4. `curl POST /api/v1/cms/sources/26/provisioning/mode -d {"mode":"auto"}` → 200, DB row updated.
5. Mở `cdc-cms-web` page TableRegistry → thấy column Engine/Mode/State, click Switch → API gọi đúng + state mode đảo.
6. E2E 3 engines (PG/Mongo/MariaDB): mỗi engine có 1 row Auto kick xuyên 4 step → state `running`.
7. APPEND `05_progress_multi_engine_unified.md` với log từng layer + commit hash.

## P6. Câu hỏi mở (đề xuất, sẽ tự quyết nếu user không phản hồi trong 1 turn)

1. **Q1**: MariaDB topic prefix nên là `cdc.mariadb` hay `cdc.gpay_maria` hay `cdc.maria.gpay`? **Đề xuất**: `cdc.mariadb` — đối xứng `cdc.goopay` (Mongo) + `cdc.gpay` (PG).
2. **Q2**: FE column "State" nên Tag color hay Steps component? **Đề xuất**: Tag — đỡ chiếm chiều rộng; Steps để khi click row mở drawer.
3. **Q3**: Toggle Switch flip có cần confirm dialog ở mọi state, hay chỉ `*_pending`? **Đề xuất**: chỉ `*_pending` (in-flight cmd) + `failed` (clear last_step_error).
