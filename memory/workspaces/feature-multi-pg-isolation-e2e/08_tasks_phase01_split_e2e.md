# 08 — Tasks — Phase 01 Split E2E

| ID | Task | Owner | Status | Depends |
|----|------|-------|--------|---------|
| T-A1 | Add 3 PG services + healthcheck vào docker-compose.yml | Muscle | ⏳ | - |
| T-A2 | Add `wal_level=logical` cho postgres-source | Muscle | ⏳ | T-A1 |
| T-A3 | Bring up 4 containers + verify healthy | Muscle | ⏳ | T-A1, T-A2 |
| T-B1 | Tách wipe scripts → wipe_auth.sql / wipe_cdc.sql / wipe_dest.sql | Muscle | ⏳ | - |
| T-B2 | Move migrations vào subfolder `migrations/cdc/` + tạo `migrations/dest/` | Muscle | ⏳ | T-A3 |
| T-B3 | Patch migration 010 (partition logic) → schema-qualified `cdc_system.<partition>` | Muscle | ⏳ | T-B2 |
| T-B4 | Tạo migration `dest/001_master_dw_init.sql` cho master tables foundation | Muscle | ⏳ | T-B2 |
| T-B5 | Update Makefile `migrate-cdc` + `migrate-dest` + `migrate-auth` targets | Muscle | ⏳ | T-B2 |
| T-B6 | Tách bootstrap → bootstrap_cdc_local.sql + bootstrap_dest_local.sql | Muscle | ⏳ | T-B2 |
| T-B7 | Tạo cdc-source-test/sql/init_source_local.sql (3 tables × 10 rows) | Muscle | ⏳ | T-A3 |
| T-C1 | cdc-auth-service config-local.yml: chỉ trỏ gpay-postgres | Muscle | ⏳ | T-A3 |
| T-C2 | cdc-cms-service config-local.yml: 2 block control_plane + destination | Muscle | ⏳ | T-A3 |
| T-C3 | centralized-data-service config-local.yml: 2 block | Muscle | ⏳ | T-A3 |
| T-C4 | Patch `pkgs/database` (3 services) hỗ trợ named connections | Muscle | ⏳ | T-C2, T-C3 |
| T-C5 | Audit grep raw SQL — đảm bảo schema-qualified hoặc dùng đúng connection | Muscle | ⏳ | T-C4 |
| T-C6 | Build 4 binaries (auth, cms, worker, sinkworker) | Muscle | ⏳ | T-C4, T-C5 |
| T-D1 | Insert connection_registry row trỏ postgres-source | Muscle | ⏳ | T-B6 |
| T-D2 | Viết deployments/connect/register_pg_source.sh — Debezium connector REST register | Muscle | ⏳ | T-A3 |
| T-D3 | Verify topic naming match (Connect prefix vs sinkworker pattern) | Muscle | ⏳ | T-D2 |
| T-D4 | Wizard endpoint test: register source object → shadow tạo trên -cdc | Muscle | ⏳ | T-C6 |
| T-E1 | Viết scripts/e2e_test_split.sh tự động chạy DoD D1-D10 | Muscle | ⏳ | T-D4 |
| T-E2 | Run E2E test → ghi report 10-point PASS | Muscle | ⏳ | T-E1 |
| T-E3 | Append 05_progress.md (workspace) DONE entry | Muscle | ⏳ | T-E2 |

## Critical path
T-A1 → T-A3 → T-B2 → T-B5 → T-B6 → T-C4 → T-C6 → T-D2 → T-D4 → T-E1 → T-E2

## Parallelizable
- T-B7 (source seed) và T-C1/C2/C3 (config) có thể song song sau khi T-A3 done.
- T-D2 (Debezium register script) độc lập với phase B/C.
