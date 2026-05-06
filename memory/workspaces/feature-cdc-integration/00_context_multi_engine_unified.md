# 00_context — Multi-Engine Unified Pipeline (Phase suffix: `multi_engine_unified`)

**Workspace**: `feature-cdc-integration` (existing).
**Lý do tạo phase suffix thay vì workspace mới**: tuân `agent/memory/global/lessons.md:1347` — "Phase mới ≠ Workspace mới". Đây là phase con của product CDC đã có workspace.

## 1. Bối cảnh kỹ thuật trước khi vào việc

### 1.1 Capture layer (Mongo→Kafka) — đã LIVE
- Container `gpay-mongo` (replica set `rs0`), Debezium connector `goopay-mongodb-cdc` RUNNING.
- 4 topic `cdc.goopay.*`: `payment-bills` (3), `refund-requests` (5158), `export-jobs` (238), `debezium_signal`.
- Live test (xem turn trước): insert vào Mongo → offset Kafka +1 trong 3s → message khớp `_id`.

### 1.2 Capture layer (PG→Kafka) — đã LIVE (E2E rig)
- Connector `cdc-pg-source` RUNNING, topics `cdc.gpay.public.{orders,payments,users}`.

### 1.3 Capture layer (MariaDB→Kafka) — CHƯA TỒN TẠI
- Không có container `gpay-mariadb`. Phải bổ sung trong phase này.

### 1.4 Consumer layer (Worker→PG) — ĐỨT cho path Mongo
Nguyên nhân (đã verify):
- `centralized-data-service/config/config-local.yml:70` — `topicPrefix: cdc.gpay` (string đơn) → chỉ subscribe 1 prefix.
- `cdc_system.source_object_registry`: 0 row active có `(source_engine_type=mongodb AND sync_engine=debezium)`. Row 1–8 mongo legacy đều `sync_engine='airbyte'`, `is_active=false`. Không có row MariaDB nào.
- `shadow_binding`: 4 active đều PG path. 0 cho Mongo/MariaDB.
- `master_binding`: 4 active đều PG path.

### 1.5 Provisioning Mode (auto/manual) — backend đã ship, FE thiếu
- Migration `047_source_provisioning_state.sql` thêm `provisioning_mode VARCHAR(20)` + state machine 9 trạng thái.
- API `cdc-cms-service`: `POST /api/v1/cms/sources/:id/provisioning/mode` body `{"mode":"auto|manual"}` (file `internal/api/provisioning_handler.go:190`).
- Service `provisioning_orchestrator.go:529 SetMode` với CAS UPDATE + D1 fan-out (manual→auto kicks `Advance` ngay).
- FE `cdc-cms-web/src/pages/TableRegistry.tsx`: KHÔNG có column/Switch nào cho `provisioning_mode`. **Đây là gap chính của FE.**

## 2. Yêu cầu user (turn này)

> "làm topicPrefix thành list/multi-prefix giữ cả PG, mongo tao còn thêm mariadb nữa. ... rồi làm thì phải làm cả cdc-worker, cả cms-api, cả cms-fe, mày làm 1 cái tế cha mày à, hay muốn làm cái vỏ rỗng."

→ Đa engine (PG/Mongo/MariaDB) + Toggle Auto/Manual full-stack 3 layer (worker + cms-api + cms-fe). Cấm "vỏ rỗng" 1 layer.

## 3. Workspace artefact đã tham chiếu

- `00_context_provisioning_mode.md` — bối cảnh provisioning_mode gốc (V1).
- `01_requirements_provisioning_mode.md` — R1..R5, state machine, API endpoints.
- `04_decisions_provisioning_mode.md` — D1..D8 (chưa đọc lại lần này, sẽ đọc khi vào implementation).
- `09_tasks_solution_track_d_hardening.md` — placeholder Track E Mongo Debezium connector (out-of-scope plan cũ; phase này tiếp nối nhưng KHÔNG dùng plan `curried-waddling-spindle.md` đã bị user reject).

## 4. Bài học áp dụng (từ `agent/memory/global/lessons.md`)

- L#1347 — Phase mới = doc suffix trong existing workspace. ✅ Tuân.
- L Cascade Liability — pipeline event-driven cần close-loop từng step. ✅ Tuân.
- L Session Handoff Liability — phải có session report cuối phiên. ✅ Sẽ append `05_progress` khi xong từng step.
- L Phase mới phải plan trước — KHÔNG đụng code khi chưa có doc set. ✅ Đây là file đầu của doc set.
