# 09_tasks_solution_current.md — Brain Analysis: Bridge Architecture Decision

> Date: 2026-04-14 10:00
> Role: Brain
> Status: DECISION NEEDED trước khi Muscle code

## Vấn đề

Airbyte tạo tables trực tiếp trong Postgres (VD: `refund_requests`) với typed columns + `_airbyte_*` columns. **KHÔNG có `_raw_data`, `_source`, `_version`, `_hash`**.

Bridge hiện tại cố:
- `bridgeInPlace`: UPDATE `_raw_data` trên bảng Airbyte → FAIL (`column _raw_data does not exist`)
- `bridgeAirbyte`: INSERT vào `target_table` = bảng Airbyte → FAIL (same reason hoặc type mismatch)

## 2 Approach

### Approach A: Tạo CDC table riêng (prefix `cdc_`)
```
Airbyte table: refund_requests (typed columns, Airbyte quản lý)
CDC table:     cdc_refund_requests (_raw_data, _source, _hash, typed columns)
Bridge:        INSERT INTO cdc_refund_requests SELECT to_jsonb(*) FROM refund_requests
```
- Ưu: Tách biệt hoàn toàn, Airbyte không bị ảnh hưởng
- Nhược: Duplicate data (Airbyte table + CDC table)

### Approach B: Thêm CDC columns vào bảng Airbyte (ALTER TABLE)
```
Airbyte table: refund_requests (typed columns + _raw_data + _source + _hash + _version)
Bridge:        UPDATE refund_requests SET _raw_data = to_jsonb(*)
```
- Ưu: Không duplicate, 1 table duy nhất
- Nhược: Airbyte có thể overwrite/drop columns khi sync lại. Coupling cao.

## Khuyến nghị: Approach B (thêm columns vào bảng Airbyte)

**Lý do**: 
- Approach A tạo 2x storage cho mỗi table
- Airbyte destination mode = `overwrite` nhưng chỉ overwrite data, không DROP + CREATE table (verified từ log: table vẫn còn sau sync)
- CDC columns (`_raw_data`, `_source`, `_hash`) sẽ tồn tại song song với Airbyte columns
- Bridge `bridgeInPlace` logic đã đúng — chỉ cần ALTER TABLE thêm columns trước

## Fix Plan

### Task 1: Bridge phải ALTER TABLE thêm CDC columns trước khi bridge
File: `command_handler.go` — `HandleAirbyteBridge` + `bridgeInPlace`

Trước khi INSERT/UPDATE, check + add missing columns:
```sql
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _raw_data JSONB;
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _source VARCHAR(20) DEFAULT 'airbyte';
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _synced_at TIMESTAMP DEFAULT NOW();
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _version BIGINT DEFAULT 1;
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _hash VARCHAR(64);
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE "refund_requests" ADD COLUMN IF NOT EXISTS _updated_at TIMESTAMP DEFAULT NOW();
```

### Task 2: Reconciliation — target_table = Airbyte alias (đã fix)
Giữ `target_table = rawTable` (alias). Không cần `cdc_` prefix.

### Task 3: Mapping rules — source_table dùng stream name (dash), target_table dùng alias (underscore)
Transform handler lookup rules bằng source_table (dash). Registry lưu cả 2.

### Task 4: Transform phải skip tables chưa có _raw_data populated
Thêm check: nếu `_raw_data` column chưa có hoặc toàn NULL → skip, không error.
