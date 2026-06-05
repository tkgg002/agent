# 01 Requirements — Disable Debezium Write to Source DB

> **Created**: 2026-05-21
> **Workspace**: `DebeziumSignalKafkaMigration` (extends existing)
> **Phase prefix**: `_disable_source_write`

## Statement (verbatim từ Boss)
> "debezium_signals tại sao vẫn tạo cái table này vào db source. mẹ mày. đã chuyển sang cái signal debezium rồi còn tạo cái này rồi snapshot-window-open, snapshot-window-close."

## Definition of Done
- Source MongoDB (10.200.187.11/12/13 replicaSet=goopay) **KHÔNG** còn collection `debezium_signals` mới được Debezium tạo.
- **KHÔNG** còn document type `snapshot-window-open` / `snapshot-window-close` ghi mới vào source DB sau khi connector restart.
- Live evidence (mongosh count BEFORE vs AFTER) phải đính kèm trong report.
- Path đề xuất phải **deploy-được trên prod** (source DB có read-only credential — không thể ghi).
- Connector vẫn `RUNNING/RUNNING`, change-stream (live CDC) vẫn produce events.
- FE Snapshot button vẫn cho user thấy trạng thái rõ ràng (KHÔNG silent fail) — nếu mất feature snapshot thì FE phải disable nút hoặc báo "not available".

## Out of scope
- Bump Debezium 2.7 → 3.x (đã thử, không fix root cause — xem `report_2026-05-20_debezium-bump-27-manual.md`).
- Implement custom snapshot worker — chỉ DOCUMENT là next-phase nếu user muốn, KHÔNG làm trong scope này.

## Constraints (per Boss governance + lessons)
1. **Source DB read-only**: không tạo collection/table trên source dù chỉ là "empty placeholder" (lesson 2026-05-20 line 3466).
2. **Không cheat config bằng tay**: KHÔNG PUT `signal.data.collection: null` qua REST một mình. Sửa code path injection.
3. **Plan trước, code sau**: viết solution doc + chờ Boss verb.
4. **Minimal impact**: chỉ sửa BE `injectDebeziumSignalDefaults` + FE `SourceConnectors.tsx`.
5. **No data destruction**: không drop collection `debezium_signals` hiện hữu trên source (Mongo prod cluster) — chỉ stop ghi mới.

## Evidence (captured 2026-05-21)
- 3 connector trên dev cluster `10.200.186.203:8083` đều có `signal.enabled.channels=source,kafka` + `signal.data.collection=<db>.debezium_signals`:
  - `goopay-ps` → `payment-service.debezium_signals` count=**38**
  - `goopay-pbs` → `payment-bill-service.debezium_signals` count=**42**
  - `goopay-ces` → `centrallized-export-service.debezium_signals` count=**62**
- Tổng 142 documents Debezium đã ghi mới vào source DB (snapshot-window-open / -close pairs).
- Connector `demo` typo `bank-service.debezium_signal` (singular) → không tạo được, snapshot silent-fail.
