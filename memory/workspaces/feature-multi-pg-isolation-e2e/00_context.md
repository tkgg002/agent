# 00 — Context (feature-multi-pg-isolation-e2e)

## Bối cảnh
Sau khi hoàn tất Phase 39 (cdc_internal triệt tiêu, schema split logic-level: cdc_system / cdc_auth_service / shadow_<src> / dw_<binding>), toàn bộ control-plane + auth + shadow + master/DW vẫn nằm chung 1 PostgreSQL container `gpay-postgres` (db `goopay_dw`).

User yêu cầu nâng cấp lên **physical isolation** giữa 3 vai trò DB + thêm 1 SOURCE DB riêng để chạy E2E auto-pipeline test:

| Container hiện tại | Container đề xuất | Vai trò | Schema |
|---|---|---|---|
| gpay-postgres | `gpay-postgres` (giữ nguyên tên) | Auth-only | `cdc_auth_service` |
| (chung) | `gpay-postgres-cdc` (mới) | CDC control plane + shadow | `cdc_system` + `shadow_<src>` |
| (chung) | `gpay-postgres-dest` (mới) | Destination/DW | master tables + `dw_<binding>` |
| — | `gpay-postgres-source` (mới) | Test source DB | `public` (sample tables: orders, users, …) |

## Yêu cầu chính (user)
1. Cung cấp SOURCE DB mới (test data sẵn).
2. Tách 3 PG containers: `gpay-postgres` (auth), `gpay-postgres-cdc`, `gpay-postgres-dest`.
3. Khi user cung cấp đủ thông tin (source connection string, table list) → luồng auto chạy hoàn chỉnh: register source → tạo shadow → Debezium capture → sinkworker ingest → transmute → master/DW.

## Liên kết workspace cũ
- `feature-cms-fe-overhaul` (Phase 39 vừa done) — cùng repo, kế thừa schema isolation rules.
- `feature-cdc-integration` (Active từ 2026-04-06) — kiến trúc Hybrid Debezium + Airbyte.
- `feature-cdc-schema-design-v2` — V2 binding model đang được dùng.

## Constraints
- KHÔNG được phá Phase 39 invariant (cdc_internal=0, public empty).
- Auth service đang chạy, phải không downtime cho login flow trong khi tách (test local thì có thể stop hết, restart hết).
- Phải reuse được Sonyflake foundation (machine_id_seq + fencing_token_seq) — sequence này hiện ở trong PG đơn nên cần xử lý cross-DB.
- Phải hoạt động được trong môi trường Docker Compose hiện tại (network bridge `cdc_default`).

## Out of scope
- Production HA / replication setup.
- Security hardening (TLS, secrets manager).
- Cross-DB foreign keys (không khả thi với postgres native; dùng app-level integrity).
