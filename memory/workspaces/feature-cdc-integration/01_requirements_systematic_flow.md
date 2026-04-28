# Requirements — Systematic Connect→Master Flow

> Stage 4 · Phase: `systematic_flow` · Muscle: claude-opus-4-7-1m · 2026-04-24
> Boss Directive: conservative defaults (Q1-Q4) + Option A wizard + Fallback Sonyflake Trigger + Atomic Swap transaction.

## 1. Scope

Xây automation flow từ **Connect Source → Master có data** với 1 click (button "Automate Everything"). Không thay schema 8-cols của `create_cdc_table()`. Thêm:

1. Persist Connection Fingerprint vào `cdc_internal.sources`.
2. Synchronous Shadow DDL từ Register (với Sonyflake fallback trigger).
3. Stateful Wizard dựa trên bảng `cdc_internal.cdc_wizard_sessions`.
4. Atomic Master Swap (BEGIN…COMMIT + RENAME TABLE).

## 2. Functional Requirements (F)

### F-1. Source Registry
- **F1.1**: Khi `POST /api/v1/system/connectors` thành công trên Kafka Connect → Insert/Upsert vào `cdc_internal.sources` với fields: `connector_name` (UNIQUE), `source_type`, `topic_prefix`, `server_address`, `database_include_list`, `collection_include_list`, `connector_class`, `raw_config_sanitized JSONB`, `status` (`created`, `running`, `failed`, `paused`, `deleted`), `created_by`, `created_at`, `updated_at`.
- **F1.2**: `GET /api/v1/sources` — list active sources (dropdown ready).
- **F1.3**: `GET /api/v1/sources/:id` — detail + collections list (parsed từ `collection.include.list`).
- **F1.4**: Khi `DELETE /api/v1/system/connectors/:name` thành công → mark `cdc_internal.sources.status='deleted'` (soft delete để giữ audit trail).

### F-2. Shadow Automator (sync)
- **F2.1**: Register Table flow gọi `service.ShadowAutomator.EnsureShadowTable(ctx, registry)` **synchronous** trước khi return 202. Nếu fail → rollback registry insert, return 500.
- **F2.2**: Idempotent: `CREATE TABLE IF NOT EXISTS cdc_internal.<target>` + `UNIQUE(source_id)`.
- **F2.3**: Schema giữ nguyên 8-cols như `create_cdc_table()` trong 003 (không downgrade).
- **F2.4**: Sau khi table created → **attach Sonyflake fallback trigger** (nếu chưa). Trigger gọi `cdc_internal.gen_sonyflake_id()` (new function in migration 028) khi `NEW.id IS NULL`.
- **F2.5**: Nếu `cdc_internal.gen_sonyflake_id()` function chưa tồn tại khi Go process khởi động → EnsureShadowTable phải deploy nó lần đầu (bootstrap DDL) rồi mới tạo table. "Deploy SQL Function gen Sonyflake nếu chưa có" (Boss directive).
- **F2.6**: UPDATE `cdc_table_registry.is_table_created=true` + `cdc_internal.table_registry` (nếu Sprint 5 row đã có) = synchronous.

### F-3. Wizard State Machine
- **F3.1**: `POST /api/v1/wizard/sessions` — create draft, body `{source_name, created_by}`, return `{session_id}`.
- **F3.2**: `GET /api/v1/wizard/sessions/:id` — load current state (step N, step_payload, status).
- **F3.3**: `PATCH /api/v1/wizard/sessions/:id` — update step or payload (optimistic: only last_step+1 hoặc re-submit current).
- **F3.4**: `POST /api/v1/wizard/sessions/:id/execute` — "Automate Everything". Chạy pipeline: create connector → persist source → register table → trigger snapshot → poll shadow row count → emit completion event. Return 202 + SSE stream hoặc poll endpoint.
- **F3.5**: `GET /api/v1/wizard/sessions/:id/progress` — { step, status, log: [...] } cho FE progress bar.

### F-4. Atomic Master Swap
- **F4.1**: `POST /api/v1/masters/:name/swap` (destructive) — body `{new_table_name, reason}`. Thực thi trong 1 transaction:
  ```sql
  BEGIN;
  ALTER TABLE public.<master> RENAME TO <master>_old_<timestamp>;
  ALTER TABLE public.<new_table_name> RENAME TO <master>;
  COMMIT;
  ```
- **F4.2**: Lock timeout: `SET lock_timeout = '3s'` trong TX để tránh treo API.
- **F4.3**: Log swap event vào `cdc_activity_log` (Audit).
- **F4.4**: Nếu fail trước COMMIT → DB tự rollback. Handler trả 409 + reason.

### F-5. Frontend (Option A: rewrite wizard)
- **F5.1**: `SourceToMasterWizard.tsx` refactor — hook `useWizardSession(sessionId)` load+persist state qua API.
- **F5.2**: URL param `?session_id=X` để resume (F5 không mất).
- **F5.3**: Active Step: disable Next button nếu step hiện tại chưa done (check progress endpoint).
- **F5.4**: Nút **"🚀 Automate Everything"** on step 1 — gọi F3.4.
- **F5.5**: `TableRegistry.tsx` register modal — `Select` source_db từ dropdown sources (F1.2), nếu user chọn source → auto-fill `source_db`, `source_type`, và list available collections từ F1.3.

## 3. Non-Functional Requirements (NF)

- **NF1 — Idempotent**: F1.1 + F2.2 + F4.1 must be safe to retry.
- **NF2 — Latency**: EnsureShadowTable < 2s (bao gồm DDL + trigger attach).
- **NF3 — Transactional**: F4.1 MUST be single Postgres TX. No "step-done-but-partial-state".
- **NF4 — Observability**: Mọi action log vào `cdc_activity_log`. Wizard progress log trong `cdc_wizard_sessions.progress_log JSONB`.
- **NF5 — Backward compat**: Không break flow cũ (NATS `cdc.cmd.create-default-columns` vẫn chạy — coi như fallback path).
- **NF6 — Security**: All write endpoints qua `destructiveChain` (JWT → RequireOpsAdmin → Idempotency → Audit).

## 4. Out of Scope

- Không rewrite `create_cdc_table()` SQL function (giữ nguyên cho legacy registry).
- Không đổi Master DDL Generator (chỉ thêm Swap endpoint).
- Không thay SinkWorker v1.25 upsert logic.
- Không Trigger Kafka Connect retry — chỉ surface state.
- Không đổi Machine ID allocation (`claim_machine_id()` vẫn do Go Worker chạy).

## 5. Acceptance Criteria

1. **AC1**: Admin tạo connector MongoDB mới từ `/sources` → GET `/api/v1/sources` trả row tương ứng.
2. **AC2**: Admin mở `/registry` → modal Register có dropdown source list, chọn 1 source → auto-fill fields → Submit → response 202 với `is_table_created=true` (synchronous DDL done).
3. **AC3**: `\d cdc_internal.<target>` hiện đủ 8 system cols + trigger `trg_<target>_sonyflake_fallback`.
4. **AC4**: `INSERT INTO cdc_internal.<target> (source_id, _raw_data, _source) VALUES (...)` không cung cấp `id` → row được tạo với `id = gen_sonyflake_id()` (BIGINT increasing).
5. **AC5**: Admin click "Automate Everything" trên Wizard → tất cả 11 step chạy tự động hoặc lỗi rõ ràng ở step nào. Session_id lưu trong URL, F5 resume được.
6. **AC6**: `POST /api/v1/masters/public_user/swap` với `new_table_name=public_user_v2` → sau khi execute, `public.public_user_v2` không còn, `public.public_user` là bảng mới, `public.public_user_old_<ts>` là bảng cũ. Tất cả trong 1 TX.

## 6. Dependencies

- Migration 027, 028 (new) chạy trước backend deploy.
- `goopay_dw` DB user có `CREATE FUNCTION` + `CREATE TRIGGER` + `ALTER TABLE` privileges (verify production role).
- Kafka Connect REST live trên `cfg.System.KafkaConnectURL`.
