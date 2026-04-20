# Plan: Bridge Fix

> Date: 2026-04-14
> Phase: bridge_fix
> Approach: B — Thêm CDC columns vào bảng Airbyte (không tạo table riêng)

## Quyết định kiến trúc

**Approach B**: ALTER TABLE thêm CDC columns vào bảng do Airbyte tạo.

Lý do:
- Không duplicate data (Approach A tạo 2 tables/stream)
- Airbyte destination mode `overwrite` chỉ overwrite data, không DROP table
- CDC columns (`_raw_data`, `_source`, `_hash`...) tồn tại song song với Airbyte columns

## Tasks

### Task 1: Helper function `ensureCDCColumns(tableName)`
- Check table exists → nếu không → return (skip)
- ALTER TABLE ADD COLUMN IF NOT EXISTS cho 8 CDC columns
- Gọi trước mỗi bridge operation

### Task 2: Fix `HandleAirbyteBridge` 
- Gọi `ensureCDCColumns` trước khi bridge
- `bridgeInPlace`: chỉ chạy sau khi ensure columns thành công

### Task 3: Fix `HandleBatchTransform`
- Check `_raw_data` column exists trước khi transform
- Check table exists trước khi query
- Skip gracefully nếu chưa sẵn sàng

### Task 4: Fix `HandlePeriodicScan`
- Check table exists + `_raw_data` column exists
- Skip tables chưa sẵn sàng

### Task 5: Fix mapping rules lookup
- Transform dùng `source_table` (dash) để lookup rules
- Verify `GetByTable` query đúng

## Definition of Done
- [ ] Bridge chạy không error trên Airbyte tables
- [ ] `_raw_data` populated cho tables có data
- [ ] Transform skip tables chưa sẵn sàng (không error)
- [ ] Periodic scan skip tables chưa có `_raw_data`
- [ ] Activity Log ghi đúng cho mỗi operation
