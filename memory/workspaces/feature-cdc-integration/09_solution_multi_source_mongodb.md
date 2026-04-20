# Solution: Multi-Source MongoDB — Registry-driven connection

> Date: 2026-04-17
> Problem: Worker dùng 1 mongoClient cho tất cả tables, nhưng source_db nằm trên các MongoDB instances khác nhau

## 1. Root Cause

| source_db | Thực tế nằm ở | Worker connect |
|-----------|---------------|----------------|
| payment-bill-service | some-mongo:27017 (1M+ records) | gpay-mongo:17017 (2 records) |
| centralized-export-service | gpay-mongo:17017 (117 records) | gpay-mongo:17017 (đúng) |

ReconCore chỉ có 1 `mongoClient` → query sai MongoDB → source count sai.

## 2. Solution: source_url trong registry

Thêm field `source_url` vào `cdc_table_registry` → mỗi source_db map tới đúng MongoDB URL.

ReconCore + ReconSourceAgent: cache mongoClient per source_url, không dùng single client.

## 3. Implementation
- Model: thêm `source_url` field
- Seed: update registry records với đúng source_url
- ReconSourceAgent: nhận source_url per call, cache clients
- ReconCore: pass source_url từ registry entry

## 4. Definition of Done
- [ ] Registry có source_url per table
- [ ] ReconSourceAgent connect đúng MongoDB per source_url
- [ ] payment-bill-service source count = 1M+ (không phải 2)
