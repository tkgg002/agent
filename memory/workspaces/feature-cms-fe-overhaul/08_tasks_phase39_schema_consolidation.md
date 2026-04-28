# Phase 39 — Task list

| # | Subject | Owner | Status |
|---:|---|---|---|
| T-39.1 | Viết workspace docs Phase 39 (01/02/03/06/08/09 prefix) | Brain | ⏳ |
| T-39.2 | Soạn 3 migration drafts (040/041/042) + rewrite cdc-auth-service/migrations/001 trong `09_tasks_solution_*.md` | Brain | ⏳ |
| T-39.3 | Soạn 3 code patches Go trong `09_tasks_solution_*.md` (user.go, alert.go, audit.go) | Brain | ⏳ |
| T-39.4 | Update wipe script `wipe_cdc_runtime_v2.sql` để DROP SCHEMA public CASCADE + recreate rỗng | Brain | ⏳ |
| T-39.5 | User approve toàn bộ scope (irreversible wipe) | User | ⏳ |
| T-39.6 | Stop 4 services (auth/cms/worker/transmute) | Muscle | ⏳ |
| T-39.7 | Backup `pg_dump` schema-only + cdc_system + auth_users ra /tmp | Muscle | ⏳ |
| T-39.8 | Apply migration 040/041/042 vào `centralized-data-service/migrations/` | Muscle | ⏳ |
| T-39.9 | Rewrite `cdc-auth-service/migrations/001_auth_users.sql` | Muscle | ⏳ |
| T-39.10 | Apply 3 code patches Go | Muscle | ⏳ |
| T-39.11 | Update `wipe_cdc_runtime_v2.sql` | Muscle | ⏳ |
| T-39.12 | Run wipe script | Muscle | ⏳ |
| T-39.13 | Run `make migrate` (centralized-data-service) | Muscle | ⏳ |
| T-39.14 | Run cdc-auth-service migration 001 | Muscle | ⏳ |
| T-39.15 | Run `make migrate-bootstrap-local` | Muscle | ⏳ |
| T-39.16 | `go build ./...` cho 4 service | Muscle | ⏳ |
| T-39.17 | Restart 4 services theo thứ tự auth → cms → worker → transmute | Muscle | ⏳ |
| T-39.18a | Verify Group 1 — 3 auth endpoints (login/register conflict/refresh) | Muscle | ⏳ |
| T-39.18b | Verify Group 2 — 5 alert endpoints (active/silenced/history/ack/silence) + background writer log clean ≥60s | Muscle | ⏳ |
| T-39.18c | Verify Group 3 — Audit smoke (reconciliation/check + mapping-rules/reload) → `cdc_system.admin_actions` count ≥2 | Muscle | ⏳ |
| T-39.18d | Verify Group 4 — 11 operator endpoints (Phase 38 baseline) | Muscle | ⏳ |
| T-39.18e | Verify Group 5 — `grep` audit: 0 raw SQL chưa qualify cho `auth_users\|admin_actions\|cdc_alerts` | Muscle | ⏳ |
| T-39.18f | Auto-flow probe: connector RUNNING + ≥4 topics + worker log clean | Muscle | ⏳ |
| T-39.19 | Document file orphan `cdc-cms-service/migrations/{003,004,005,013}.sql` để xoá | Brain | ⏳ |
| T-39.20 | Append `05_progress.md` Phase 39 closure + lesson nếu có | Brain | ⏳ |

## Mapping requirements → tasks

| Requirement | Task(s) |
|---|---|
| Move admin_actions vào cdc_system | T-39.8 (migration 040) + T-39.10 (audit.go patch) |
| Move cdc_alerts vào cdc_system | T-39.8 (migration 041) + T-39.10 (alert.go patch) |
| Schema cdc_auth_service + auth_users | T-39.9 + T-39.10 (user.go patch) |
| Drop public residue | T-39.11 + T-39.12 (wipe script) |
| Drop cdc_internal | T-39.12 (wipe script) |
| search_path bao gồm cdc_auth_service | T-39.8 (migration 042) |
| Build pass 4 service | T-39.16 |
| Auth endpoints (3) | T-39.18a |
| Alert endpoints (5) + background writer | T-39.18b |
| Audit-wrapped mutation endpoints (~30, smoke ≥2) | T-39.18c |
| 11 operator read endpoints (Phase 38) | T-39.18d |
| Raw SQL qualify audit | T-39.18e |
| Auto-flow probe | T-39.18f |

## Open follow-ups (debt → phase mới)

- `D-39.A` — Squash 39 migration files của centralized-data-service thành 1 baseline + sequence sau (tách thành phase riêng vì cần ngừng dev hoàn toàn)
- `D-39.B` — Cleanup data linkage gap (Debezium include list ↔ source_object_registry) — kế thừa từ Phase 38 D-38.C/D
- `D-39.C` — Move shadow data tables lừ dở (nếu Phase 38 đã có public.payments etc. là CDC ingest output) thật ra do worker tạo nhầm vị trí — cần audit shadow_automator để chắc lần sau bind tới `shadow_<src>` schema
