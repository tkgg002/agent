# Project Context

> **Last Updated**: 2026-05-04
> **Maintained by**: Brain (Antigravity) qua workspace `feature-system-refactor-2026-05`

## Overview

**cdc-system** — Hệ thống Change Data Capture (CDC) đồng bộ dữ liệu thay đổi từ
nguồn vận hành (MongoDB / PostgreSQL / MariaDB) sang Data Warehouse PostgreSQL,
qua 2 tầng `shadow` → `master`, có schema evolution kiểm soát, reconciliation,
DLQ retry, và operator UI.

- **Scale**: monorepo 4 service (3 Go + 1 TS), ~250 .go file + 22 .tsx (~7600 LOC FE).
- **Users**: operator/admin nội bộ qua CMS UI; developer Devops ops backend.
- **Stage**: Development / Local smoke (chưa staging hoặc production deploy).
- **Repo**: 1 git monorepo, branch `main`, ngưỡng 6 commit (lịch sử ngắn).

---

## Domain Knowledge

### Terminologies

- **Source Object Registry** (`cdc_system.source_object_registry`): bảng metadata khai báo collection/table cần CDC. Có UNIQUE `normalized_source_key`.
- **Shadow Layer** (`cdc_internal.<table>` hoặc `shadow_<conn>_<engine>.<table>`): tầng đầu tiên — Debezium event raw + system cols `_gpay_source_id`, `_raw_data`, `_source_ts`, `_synced_at`, `_version`, `_hash`, `_gpay_deleted`. V1 dùng `_id` PK + `_raw_data`; V2 thêm anchor `_gpay_source_id` UNIQUE.
- **Master Layer** (`public.<name>` hoặc DW schema): tầng typed, applied mapping rules, có RLS.
- **Schema Proposal** (`cdc_internal.schema_proposal`): drift detect → financial whitelist → admin approve → ALTER TABLE shadow + INSERT mapping rule.
- **Mapping Rule** (`cdc_mapping_rules`): (source_table, target_column, data_type, jsonpath, transform_fn, status).
- **Transmute**: pipeline shadow → master qua `gjson.GetBytes(_raw_data, jsonpath)` + transform_fn + OCC upsert.
- **TransmuteScheduler**: cron poll 60s, fencing, 3 mode (cron / immediate / post_ingest).
- **JobMonitor**: subscribe `cdc.evt.transmute.completed` → close-loop `transmute_schedule.last_status`.
- **Recon (Reconciliation)**: 3 tier hash window source vs dest → detect missing → heal qua OCC upsert.
- **DLQ** (`failed_sync_logs` + state machine): write-before-publish, sanitized payload, non-blocking retry.
- **Fencing**: machine_id + token để tránh 2 instance scheduler cùng tick 1 schedule.

### Business Rules

- `master_binding.is_active` chỉ bật được khi `schema_status='approved'` (CHECK constraint).
- Transmute apply rule chỉ khi cả gate chain pass: master active+approved, shadow active+profile_active, có ≥1 approved rule.
- OCC theo `_source_ts older` → tránh overwrite dữ liệu mới hơn.
- Mongo healing read MUST ép `primary` (tránh replication lag).
- DLQ payload phải mask PII trước persist hoặc publish.
- KHÔNG phát tán sample PII thô qua NATS alert subjects.
- Admin-API V2 register: token bắt buộc strong khi `ADMIN_API_DEV != true` (Phase F1 boot fail-fast).

---

## Architecture Overview

### Service Groups

| Group | Thành phần | Đặc điểm |
|---|---|---|
| **Worker plane** | `centralized-data-service` (4 binary: `worker`, `admin-api`, `sinkworker`, `profile_table`) | Go 1.26.1, Gin, GORM, golang.org/x/time/rate. 144 .go file, 39 test. Heaviest service. |
| **Control plane** | `cdc-cms-service` | Go 1.26.1, Fiber + NATS + Redis. 76 .go file, 10 test. Live local `/tmp/cms-server` PID 13653 (5d). |
| **Auth plane** | `cdc-auth-service` | Go 1.26.1, Fiber + golang-jwt. Tiny — 9 .go file, 0 test. CHƯA chạy local. |
| **Operations UI** | `cdc-cms-web` | TypeScript + Vite + React. 22 .tsx, 7634 LOC. CHƯA chạy local. |

### Data Stores

- 4 PG container (5432 main / 5433 cdc-metadata / 5434 dest-DW / 5435 source).
- Mongo (17017), MariaDB (13307) làm source.
- Redis (16379), Kafka (19092/19093), Schema Registry (18081), NATS (14222).
- Debezium qua Kafka Connect (18083), OTel Collector (14317/14318).

---

## Current State (2026-05-04)

- **Phase**: System Refactor 2026-05 (workspace `feature-system-refactor-2026-05`), bucket B1 + B2.
- **Active Focus**:
  - B1: hygiene (commit pending Phase F1+F3 ✅ DONE 2026-05-04 17:11, doc drift Airbyte ✅ DONE).
  - B2: start auth + FE local + smoke E2E operator path (in progress).
- **Recent Done**:
  - Phase F1 admin-api hardening (5 issue) — commit `92d78d3`.
  - Phase F3 Mongo collection fallback fix (3 vị trí helpers.go) — cùng commit.
  - JobMonitor close-loop hoạt động — verify cron tick 09:51:13 UTC `success` cho 6/6 schedule.
- **Known Issues**:
  - `cdc-auth-service` 0 test, không chạy local (B1=HIGH gap).
  - `cdc-cms-web` không chạy dev server.
  - Architectural debts: shadow auto-create, prune V1 legacy seed (10 row), master cascade từ admin-api — defer Phase B3.

## Key Dependencies & Risk Areas

- **Mongo + Debezium + Kafka Connect** fail → CDC pipeline đứt → mọi shadow stale.
- **PostgreSQL CDC metadata DB (5433)** fail → registry/binding/schedule không còn → control plane chết.
- **NATS** fail → command bus đứt → admin-api signal worker không nhận, JobMonitor không close loop.
- **`/tmp/cms-server`** uptime 5d → có thể leak memory hoặc cache stale → restart định kỳ.
- **Schema Registry** drift → Debezium fail decode mới → ingest fail (xem lesson L-debezium-schema-evolution-compat).
