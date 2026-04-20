# Requirements: Bridge Fix — Airbyte tables thiếu CDC columns

> Date: 2026-04-14
> Phase: bridge_fix
> Triggered by: Runtime errors khi bridge/transform trên Airbyte tables

## Bối cảnh

Airbyte tạo tables trực tiếp trong Postgres (VD: `refund_requests`, `payment_bill_histories`) với:
- Typed columns từ source (id, fee, note, createdAt...)
- Airbyte internal columns (`_airbyte_raw_id`, `_airbyte_extracted_at`, `_airbyte_meta`, `_airbyte_generation_id`)
- **KHÔNG có CDC columns**: `_raw_data`, `_source`, `_synced_at`, `_version`, `_hash`, `_deleted`, `_created_at`, `_updated_at`

CDC system (bridge + transform + scan) cần `_raw_data` để hoạt động → tất cả operations fail.

## Yêu cầu

### R1: Bridge phải tự thêm CDC columns vào bảng Airbyte trước khi bridge
- ALTER TABLE ADD COLUMN IF NOT EXISTS cho 8 CDC columns
- Chạy tự động, không cần user click

### R2: Bridge `bridgeInPlace` phải hoạt động trên bảng Airbyte 
- Pack typed columns → `_raw_data` JSONB
- Chỉ update rows chưa có `_raw_data` hoặc hash thay đổi

### R3: Transform phải skip tables chưa sẵn sàng
- Nếu table chưa có `_raw_data` column → skip, không error
- Nếu `_raw_data` toàn NULL → skip

### R4: Mapping rules lookup phải xử lý source_table (dash) vs target_table (underscore)
- Registry: `source_table = "payment-bill-histories"`, `target_table = "payment_bill_histories"`
- Mapping rules: `source_table = "payment-bill-histories"` (dash)
- Transform lookup bằng source_table → phải match

### R5: Tables chưa tồn tại (VD: `payment_bills`, `refund_requests_histories`)
- Nếu Airbyte chưa sync → table chưa có trong DB
- Bridge/Transform phải check table exists trước, skip nếu chưa có
