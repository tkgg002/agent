# 05_progress.md — Kafka Connect local → prod — Audit Log (APPEND-only)

> CLAUDE.md §11: TUYỆT ĐỐI CẤM overwrite. Chỉ APPEND vào cuối file.

---

## [2026-05-18 17:15] [Muscle:claude-opus-4-7] Workspace khởi tạo

User trigger: cung cấp Dockerfile prod (Strimzi base + 4 plugin) + quote "đây là con kafka trên prod, tao đang muốn kết nối local của tao để chạy trên nó. mày lên plan xem sao."

**Lessons + context đã đọc (CLAUDE.md §7)**:
- `agent/memory/global/project_context.md` (cdc-system overview, 4 service stack, Kafka 19092, SR 18081, Connect 18083)
- `agent/memory/global/tech_stack.md` (Go 1.26.1, Sarama, GORM, Strimzi prod inferred)
- `agent/memory/global/active_plans.md` (workspace `feature-system-refactor-2026-05` Active; recent: dynamic source DSN fix 2026-05-18)
- `agent/GEMINI.md` (full 14 rules)
- `lessons.md` grep keyword `kafka.connect`, `strimzi`, `EnvVarConfigProvider`: phát hiện:
  - L1154: Pattern "FE không gọi thẳng infra REST" → applicable cho dev workflow (local script gọi prod Connect REST OK miễn là dev đã có RBAC, không phải user-facing UI)
  - L1688: Multi-tier filter Debezium — applicable cho Plan A1 (topic.include vs db.include consistency)
  - L781: Redpanda Console v2.8.1+ regression — không apply trực tiếp ở plan này

**Files tạo MỚI**:
1. `00_context.md` (workspace) — scope + 3 interpret H1/H2/H3 + landscape + constraints + risks upfront
2. `01_requirements.md` (workspace) — R1-R5 common + R1.1-1.5 H1-specific + R2.1-2.5 H2-specific + N1-N7 non-functional + DoD đầy đủ
3. `02_plan.md` (workspace) — Plan A (H2) recommend ưu tiên với 4 phase (A0-A3) + code demo chi tiết từng file; Plan B (H1) skeleton; Plan C deferred options; Risk Register R-A đến R-F; 4 approval gates
4. `05_progress.md` (file này)

**Files KHÔNG đụng** (compliance):
- KHÔNG đụng `src/` cdc-worker
- KHÔNG đụng Dockerfile prod (user paste, không nói repo path)
- KHÔNG sửa connector JSON / docker-compose / script (chỉ plan, chưa execute)
- KHÔNG đụng cluster prod
- KHÔNG tạo file report ở root repo (chưa execute)

**Status**: 📋 PLAN ONLY — chờ user xác nhận H1/H2/cả hai + cung cấp credentials/URLs prod (Phase A0 pre-flight).

**Decision pending từ user (4 input cần)**:
1. Confirm interpret: H1 (local stack ↔ prod Kafka broker) / H2 (local script POST connector → prod Connect REST) / cả hai
2. Bootstrap broker prod URL + Schema Registry URL + Connect REST URL
3. Auth method (SASL/SCRAM, mTLS, OAuth) + credentials lookup (k8s Secret name / Vault path / file)
4. Topic naming convention dev (`cdc.dev-traingn.*` OK chưa, có namespace khác không)

---

## [2026-05-19 ~09:50] [Muscle:claude-opus-4-7] Extract hardcoded SR + Kafka bootstrap khỏi cdc-cms-web SourceConnectors.tsx

User directive nguyên văn: *"'/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx' mang cái key.converter.schema.registry.url này ra env. tìm luôn cái gì cần mang ra ví dụ gpay-kafka:9092 cũng đang set cứng phải không"*

**Bối cảnh**: User đang frustrated với hardcoded infra URL/host trong code FE. Đây là task riêng — KHÔNG phải Phase A1 của workspace plan (đó là refactor connector JSON ở `centralized-data-service/deployments/debezium/`). Task này scope tách: extract config ra ENV trong 1 file FE cdc-cms-web duy nhất.

**Pre-edit survey** (`src/pages/SourceConnectors.tsx`):
- 6 hardcode `'http://gpay-schema-registry:8081'` ở `buildConnectorConfig()` (mongo/mysql/postgres × key+value converter)
- 3 hardcode `'gpay-kafka:9092'` ở `schema.history.internal.kafka.bootstrap.servers` (mongo/mysql/postgres)
- 1 placeholder `'mongodb://localhost:27017/?replicaSet=rs0'` line 401 — SKIP (Form default, user nhập đè runtime)
- 2 fallback `host: 'localhost'` line 244/428 — SKIP (parse seed fallback, không phải config thực)
- Port defaults 3306/5432 — SKIP

**Files MODIFIED**:
- `cdc-cms-web/src/pages/SourceConnectors.tsx`:
  - Thêm 2 const sau line 36 (sau `const { Title, Text } = Typography;`):
    ```ts
    const SCHEMA_REGISTRY_URL =
      import.meta.env.VITE_SCHEMA_REGISTRY_URL || 'http://gpay-schema-registry:8081';
    const KAFKA_BOOTSTRAP_SERVERS =
      import.meta.env.VITE_KAFKA_BOOTSTRAP_SERVERS || 'gpay-kafka:9092';
    ```
  - Replace 6 chỗ `'http://gpay-schema-registry:8081'` → `SCHEMA_REGISTRY_URL`
  - Replace 3 chỗ `'gpay-kafka:9092'` → `KAFKA_BOOTSTRAP_SERVERS`

**Bug khám phá + fix** (CLAUDE.md §3 Verification Before Done):
- Lần đầu dùng `replace_all=true` cho cả 2 string → đụng cả fallback literal trong const declaration → tạo self-reference `SCHEMA_REGISTRY_URL || SCHEMA_REGISTRY_URL` (recursive ReferenceError runtime). Phát hiện ngay sau verify grep. Fix bằng 1 Edit không-replace_all với context cụ thể của const declaration.

**Verification**:
- `grep -nE '(gpay-|:8081|:9092|http://)' src/pages/SourceConnectors.tsx` → 0 hit trong `buildConnectorConfig` (chỉ còn fallback strings ở const declaration)
- `npx tsc --noEmit -p .` → silent (TS compile pass)
- `npm run build:prod` → built in 544ms, `dist/assets/SourceConnectors-yHZrUhCp.js` 20.76 kB
- Built bundle grep: `gpay-schema-registry:8081` + `gpay-kafka:9092` đã được Vite inline thành literal (fallback baked vì `.env.production` chưa có ENV mới)

**Behavior expected post-edit**:
| Build mode | ENV present | Bundle output |
|---|---|---|
| Local `npm run build:prod` (mặc định) | KHÔNG có VITE_SCHEMA_REGISTRY_URL | Bundle chứa `http://gpay-schema-registry:8081` (Docker DNS — work cho local stack) |
| Prod build với `VITE_SCHEMA_REGISTRY_URL=http://kafka-sr-prod-svc.data-hub:8081` | CÓ | Bundle chứa URL prod (override fallback) |

**Files KHÔNG đụng**:
- KHÔNG sửa `.env.production` (vì user chưa cho URL prod cluster Kafka SR — không assume value)
- KHÔNG sửa Dockerfile / docker-entrypoint.sh
- KHÔNG đụng `centralized-data-service/` (workspace task chính — vẫn đang chờ approval)
- KHÔNG đụng `cdc-cms-service/` backend

**Status**: ✅ DONE task riêng. Build PASS, verify PASS.

**Next user decisions còn pending**:
1. (Task riêng) Có cần thêm 2 var `VITE_SCHEMA_REGISTRY_URL` + `VITE_KAFKA_BOOTSTRAP_SERVERS` vào `.env.production` với placeholder/value cụ thể không?
2. (Workspace task chính) 4 input pre-flight ở entry trên (interpret H1/H2 + URLs prod + auth + topic namespace) — vẫn chờ.

---
