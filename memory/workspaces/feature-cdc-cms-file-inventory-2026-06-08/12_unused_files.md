# 12_unused_files — File CHƯA DÙNG trong `cdc-cms-service`

> Bằng chứng 3 tầng: **reachability** (`go list -deps` từ 2 binary) · **deadcode** (75 hàm unreachable) · **grep reference** (symbol 0 ref ngoài file, đã trừ wiring router/bus). `go build ./...` = PASS.

## 🔴 A. DEAD HẲN — xoá an toàn sau review (8 file · 575 LOC)

| File | LOC | Bằng chứng |
|---|---|---|
| `internal/infra/cache/doc.go` | 6 | **Dead package**: 0 import non-test, unreachable từ binary; `pkgs/rediscache` đã thay thế. Chỉ là placeholder. |
| `internal/infra/persistence/master_swap.go` | 192 | **Cả 6 hàm dead** (deadcode). `master.swap` đi qua NATS (`RegisterSubject`) → worker ngoài xử lý; impl này bị bỏ lại từ kế hoạch cũ. |
| `internal/bootstrap/registry_mirror.go` | 260 | Hàm chính `SyncLegacyToV2Bootstrap` **comment-out 100%**, 5 helper (`slugify`, `splitHostPort`...) orphan (deadcode). |
| `internal/model/cdc_event.go` | 28 | `CDCEvent/CDCEventData/UpsertRecord` 0 ref non-test — artifact từ `sync_v2` cũ chưa dọn. |
| `internal/app/ports/query_bus.go` | 16 | Interface `QueryBus`/`Query` không có impl/`.Ask()` call site — queries gọi concrete handler trực tiếp. |
| `internal/app/ports/publisher.go` | 10 | Interface `Publisher` không impl/caller — code dùng `NatsClient.PublishReload` trực tiếp. |
| `pkgs/utils/type_inference.go` | 50 | `InferDataType` 0 caller production. |
| `pkgs/utils/hash.go` | 13 | `CalculateHash` 0 caller production. |

## 🟠 B. NGHI THỪA / one-shot — cần User xác nhận

| File | LOC | Ghi chú |
|---|---|---|
| `cmd/sync_v2/main.go` | 40 | One-shot migration tool, DSN hardcode `localhost:5433`, không AppConfig/OTel, không Makefile/Docker target. Đã hoàn thành nhiệm vụ → archive hoặc xoá khỏi build. |
| `internal/api/registry_handler_read.go` | 58 | `List` + `GetStats` **không mount** trong `router.go` (chỉ `SyncHealth` được wire) → 2/3 hàm dead. Gỡ 2 endpoint hoặc mount lại. |

## 🟡 C. PARTIAL — hàm chết trong file SỐNG (cleanup, không xoá file)

**Pattern 1 — `*ForTest` shims (~15+ hàm, nhiều file)**: do cây test **không co-located** (`test/internal/...` tách rời) nên phải export hàm private cho test → đẻ ra `NewXForTest`, `MapErrForTest`, `BuildRowForTest`... deadcode coi là chết vì không reachable từ binary. **Không phải dead logic** — nhưng là **smell kiến trúc**. Fix gốc: đưa `_test.go` về cạnh file (co-locate) → xoá sạch toàn bộ `*ForTest`.

**Pattern 2 — `*Query.Type()` (17 file `queries/`)**: CQRS query-bus **chưa wire** (handler gọi trực tiếp, không qua bus) → mọi method `.Type()` chết. Hoặc xoá `.Type()` + marker, hoặc hoàn thiện query bus.

**Lẻ tẻ**: `middleware/audit.go` `Stop()`/`DroppedCount()` (no-op) · `pkgs/observability/otel.go` `Tracer()`/`StartSpan()` · `persistence/provisioning_state_machine.go` `ProvisioningIsPending/IsTerminal` · `api/utils.go` `normalizeShadowIdent` · `api/master_registry_handler_resolve.go` `trimString`.

## 🔵 D. TRÙNG LẶP concept `model/` (V1 GORM) ↔ `domain/` (V2 clean) — 4 cặp

Không "dead" hẳn nhưng gây nhầm lẫn, V1 đang bị drain:

| V1 (model/, GORM) | V2 (domain/, clean) | Bảng |
|---|---|---|
| `model/mapping_rule.go` | `domain/mapping/rule.go` | `cdc_mapping_rules` (V1) vs `mapping_rule_v2` |
| `model/reconciliation_report.go` | `domain/reconciliation/report.go` | `cdc_reconciliation_report` |
| `model/failed_sync_log.go` | `domain/reconciliation/failed_log.go` | `failed_sync_logs` |
| `model/table_registry.go` | `domain/source/object.go` | `cdc_table_registry` (V1) vs `source_object_registry` (V2) |

→ Khi V1 deprecated xong, 4 file `model/*` này thành dead. Hiện `model.MappingRule` chỉ còn ~2 caller.

## ⚪ E. Ghi chú liên quan (không phải dead, nhưng nên xử khi reorg)
- **18/32 command dùng raw `*gorm.DB`** (bypass port) — pain point chính của refactor v2.
- `persistence/provisioning_state_machine.go`: comment `// Package service` nhưng `package persistence` (copy-paste từ centralized-data-service chưa sửa header).
- Domain entity **anemic** (struct thuần, logic nằm ở commands/persistence).
- `docs/docs.go` (2280 LOC) = **swaggo generated** — giữ, không đụng.

## Tóm tắt số
- **Dead hẳn (A)**: 8 file / **575 LOC** → xoá sau review.
- **Nghi thừa (B)**: 2 file / 98 LOC → User xác nhận.
- **Partial cleanup (C)**: ~30+ hàm chết (chủ yếu `*ForTest` + `.Type()`) → cleanup khi co-locate test / wire bus.
- **Trùng V1/V2 (D)**: 4 file model → dead khi V1 EOL.
