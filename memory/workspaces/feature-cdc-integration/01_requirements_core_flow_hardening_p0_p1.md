# 01 — Requirements: Core-Flow Hardening Phase P0+P1

**Phase code**: `core_flow_hardening_p0_p1`
**Created**: 2026-05-04 13:55 (+07)
**Trigger**: User chốt scope sau audit `report_core_flow_audit_20260504_1340.md`.
**Source of truth**: User message ngày 2026-05-04 (sau audit P0 Q&A).

---

## Tasks chốt scope

| ID | Gap | Tên |
|----|-----|-----|
| P0.1 | G1 | Refactor `kafka_consumer.go` — Reader manager nhận tín hiệu NATS để refresh topics (regex API không khả thi với kafka-go consumer group mode) |
| P0.2 | G6 | Phác thảo cấu trúc `cdc-admin-api` cho transactional registration (3 bước) |
| P1.1 | G3 | `handleDelete` đổi UPSERT thay UPDATE cho cột `_deleted` |

Out-of-scope phase này: G2 (parse-by-engine), G4 (`_id`→`id` cleanup), G5 (multi-master fan-out). Sẽ phase sau khi P0/P1 land.

---

## Functional requirements

### FR-1 (P0.1) — Dynamic topic subscription

- **FR-1.1**: Khi admin PUT Debezium include list để add topic mới, worker không phải restart vẫn pick up được topic mới trong vòng ≤ 60s.
- **FR-1.2**: Khi admin gửi NATS message `cdc.cmd.kafka.refresh-topics`, worker phải re-discover + recreate Kafka reader nếu topic set thay đổi, trong vòng ≤ 5s từ lúc nhận message.
- **FR-1.3**: Trong lúc recreate reader, không được drop in-flight messages của topic cũ — phải flush batch + commit offset trước khi close reader cũ.
- **FR-1.4**: Idempotent: nếu topic set không đổi, không recreate (tiết kiệm rebalance churn).
- **FR-1.5**: Background safety-net ticker 60s vẫn auto-call refresh ngay cả khi không có signal NATS.

### FR-2 (P0.2) — Transactional source registration

- **FR-2.1**: Endpoint mới `POST /v2/sources/register` thực thi cả 3 bước trong 1 request:
  1. INSERT registry rows (source_object_registry + shadow_binding + master_binding + transmute_schedule).
  2. PUT Debezium connector include list (extend, không replace).
  3. Pre-empt Schema Registry per-subject compat = NONE cho `<topic>-value`.
- **FR-2.2**: Bước 1 chạy trong DB transaction; bước 2-3 sau commit DB nhưng có rollback compensation nếu fail.
- **FR-2.3**: Response trả về `provisioning_state` cuối cùng + `last_step_error` nếu có.
- **FR-2.4**: Idempotent theo `object_code` — gọi lại với cùng object_code không tạo duplicate.
- **FR-2.5**: Sau bước 3 thành công, publish NATS `cdc.cmd.kafka.refresh-topics` để worker reload (FR-1.2).
- **FR-2.6**: Optional verify-loop: poll Kafka topic offset trong N giây, nếu > 0 thì update `provisioning_state='active'`, ngược lại `provisioning_state='pending_data'`.

### FR-3 (P1.1) — Delete tombstone-first

- **FR-3.1**: `handleDelete` ở `event_handler.go` thay UPDATE bằng INSERT…ON CONFLICT…DO UPDATE.
- **FR-3.2**: INSERT branch phải set: PK column, `_gpay_source_id` (B11 anchor), `_deleted=TRUE`, `_created_at=NOW()`, `_updated_at=NOW()`, `_source='debezium'`.
- **FR-3.3**: ON CONFLICT branch giữ existing data, chỉ update `_deleted=TRUE`, `_updated_at=NOW()`.
- **FR-3.4**: Behavior khi PK column khác nhau giữa source và shadow (Mongo `_id` vs PG `id`): phải tôn trọng registry config (G4 fix sau, phase này không touch hard-rename).

---

## Non-functional requirements

- **NFR-1**: Build pass (`go build ./...`).
- **NFR-2**: Existing tests pass (`go test ./internal/handler/... ./internal/service/...`).
- **NFR-3**: New unit tests cho Reader manager + transactional registration handler.
- **NFR-4**: Admin API có security gate cơ bản (token auth từ env, không expose ra public).
- **NFR-5**: Tuân thủ CLAUDE.md §6 (Simplicity First — không over-engineer; minimal impact).

---

## Acceptance criteria

| Task | Criterion |
|------|-----------|
| P0.1 | Smoke: chạy worker → admin PUT include list thêm collection mới → publish `cdc.cmd.kafka.refresh-topics` → INSERT 1 doc source → trong 30s shadow row landed (KHÔNG restart worker). |
| P0.2 | Smoke: `curl -X POST /v2/sources/register -d {...}` → kiểm tra `cdc_system.source_object_registry` có row mới + Debezium connector include list được extend + Schema Registry compat=NONE + INSERT source xong landed shadow trong 30s. |
| P1.1 | Smoke: source PG DELETE row chưa từng INSERT shadow → shadow phải có row với `_deleted=TRUE` + `_gpay_source_id` đầy đủ. |

---

## Dependencies & assumptions

- **D-1**: kafka-go v0.4.50 không support regex topic + GroupID combo → MUST dùng Reader manager (không phải GroupTopics regex).
- **D-2**: Schema Registry HTTP API endpoint khả dụng từ worker network namespace (đã verified ở B10).
- **D-3**: Debezium Connect API endpoint khả dụng (đã verified ở multi A series).
- **A-1**: Admin API có thể chia sẻ DB connection pool với worker (cùng `cdc_dw`) hoặc tự khởi tạo riêng — design ưu tiên riêng để cô lập.
- **A-2**: NATS subject naming convention `cdc.cmd.<verb-noun>` đã chuẩn (existing: `cdc.cmd.transmute`, `cdc.cmd.restart-debezium`, etc.) → `cdc.cmd.kafka.refresh-topics` follow chuẩn.

---

## Risks

| Risk | Mitigation |
|------|-----------|
| Reader recreate gây offset reset | Phải `Close()` reader cũ sau khi commit offset (kafka-go ConsumerGroup persist offsets ở broker — recreate cùng GroupID sẽ resume). |
| Admin API vụ rollback registry partial | Bước 1 trong DB tx; nếu bước 2/3 fail sau commit → endpoint trả 207 Multi-Status với partial state, để operator/dashboard quyết retry hoặc manual rollback. KHÔNG auto-DELETE registry rows (phá flow audit). |
| handleDelete INSERT tombstone-first có thể vi phạm FK nếu shadow row tham chiếu master | Shadow tables không có FK ra master (verified). An toàn. |

---

## Skills sẽ dùng

- Read, Grep, Bash (psql, docker, curl), Write, Edit (Brain → Muscle execute).
- `/muscle-execute` workflow cho phase implementation.
- `/security-agent` mandatory sau khi P0.2 land (admin API có ext network surface).
- `/qa-agent` cho integration test scenario.
