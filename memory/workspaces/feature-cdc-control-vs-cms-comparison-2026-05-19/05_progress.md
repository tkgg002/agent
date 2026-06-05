# 05_progress.md — Audit log (APPEND-ONLY)

## 2026-05-19 — Khởi tạo workspace + deliver gap analysis

- **14:30** — User yêu cầu: "xem qua cái cdc-control rồi so sánh với cdc hiện tại, làm tài liệu xem có gì khác, gì chi tiết hơn. ko thực hiện bất cứ dòng code nào".
- **14:31** — Tạo workspace `feature-cdc-control-vs-cms-comparison-2026-05-19/`.
- **14:32** — Viết `00_context.md` (bối cảnh + 3 repo + mục tiêu + non-goals).
- **14:33** — Viết `02_plan.md` (3 phase + 14 section cho file gap analysis + DoD).
- **14:34** — Spawn **2 Explore agent song song**:
  - Agent A: map cdc-control toàn bộ (Python/FastAPI/MySQL/Jinja2).
  - Agent B: map cdc-cms-service + cdc-cms-web (Go/Fiber/Postgres/React).
- **14:50** — Agent A trả về report inline (~12K tokens). Agent B output 52.9KB persisted tại `tool-results/toolu_01X1t2ZpnbLLTyLd9eAavr2W.json`.
- **14:51** — Extract Agent B output qua `jq` vào `/tmp/agent_b_output.md` (910 lines).
- **14:52** — Đọc Agent B output theo chunks 300 lines.
- **15:00** — Viết `10_gap_analysis.md` — file chính, gồm:
  - 14 section comprehensive.
  - 5 bảng so sánh chính: entity, route, connector lifecycle, shadow management, schema sync.
  - Bảng "Gap matrix" 50 feature × repo với legend ✅⚠❌.
  - Section 13: 7 mảng cdc-control chi tiết hơn.
  - Section 14: 12 mảng cdc-cms chi tiết hơn.
  - Tổng kết 1 câu.

## Verification (DoD check)

| Item | Status |
|------|:------:|
| `00_context.md` tạo | ✅ |
| `02_plan.md` tạo | ✅ |
| `10_gap_analysis.md` chứa ≥ 14 section | ✅ (14 section) |
| `10_gap_analysis.md` chứa ≥ 5 bảng so sánh | ✅ (>10 bảng) |
| `05_progress.md` append-only | ✅ (file này) |
| `07_status.md` tạo | ✅ (kế tiếp) |
| KHÔNG sửa source code 3 repo | ✅ (chỉ Read + workspace Write) |

## Skills sử dụng trong session này

- **Workspace governance** (CLAUDE.md §7) — khởi tạo full doc set.
- **Bash mkdir + ls** — chuẩn bị workspace directory.
- **Agent (Explore × 2 parallel)** — gather ground truth comprehensive cho 3 repo.
- **Bash jq** — extract persisted output JSON → markdown.
- **Read chunked** — đọc large output theo offset/limit.
- **Write** — tạo các file markdown trong workspace.
- **Plan & Verify** (CLAUDE.md §3) — plan 3-phase + verify DoD.
- **Brain Code Prohibition** (CLAUDE.md §12) — KHÔNG đụng source code.

## 2026-05-19 (cont) — Update README centralized-data-service với ENV vars docs

- **User yêu cầu**: cập nhật README.md với thông tin về 4 nhóm ENV variables của `config.go` file 2 (`AutomaticEnv` + `applyEnvOverrides`).
- **Verify location**: grep `applyEnvOverrides` → `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/config/config.go` (KHÔNG phải `cdc-cms-service`).
- **Read config.go** (313 dòng): confirm structure user mô tả là chính xác. Bổ sung detail user chưa list:
  - `OTEL_LOGS_SAMPLEBYSEVERITY_*` (DEBUG/INFO/WARN/ERROR/FATAL nested)
  - `OTEL_LOGS_MEMORYLIMITMIB`, `OTEL_LOGS_FALLBACK_*` (nested)
  - `mergeTopicPrefixAlias` — alias `kafka.topicPrefixes` plural → `cfg.Kafka.TopicPrefix`
  - `validateConfig` — 5 rules (server.port, systemDb, masterDb, jwt.secret + production placeholder check)
  - Underscore subtle differences trong overrides: `KAFKA_CONNECT_URL` vs auto `DEBEZIUM_KAFKACONNECTURL`, `DEBEZIUM_CONNECTOR_NAME` vs auto `DEBEZIUM_CONNECTORNAME`, `KAFKA_SCHEMA_REGISTRY_URL` vs auto `KAFKA_SCHEMAREGISTRYURL`
- **Edit README**: insert section "### Environment variables" vào sau YAML example trong "## Configuration", trước "## Surfaces". 7 sub-section: cfgPath/CFG_PATH, AutomaticEnv map, applyEnvOverrides, Dynamic CONNECTION_OVERRIDE_*, Alias hỗ trợ, Validation, So sánh cũ vs mới.
- **Verify**: README từ 187 dòng → 326 dòng. Grep confirm 5 anchor key có mặt.
- **KHÔNG sửa source code Go**: chỉ edit README.md.
