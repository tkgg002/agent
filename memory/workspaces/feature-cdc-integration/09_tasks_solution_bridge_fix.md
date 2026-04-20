# Solution: Bridge Fix

> Date: 2026-04-14
> Phase: bridge_fix
> Status: Tài liệu xong, chờ user duyệt approach trước khi Muscle code

## Approach đã chọn

**Approach B**: Thêm CDC columns vào bảng Airbyte (không tạo table riêng).

## Files cần sửa

| File | Thay đổi | Task |
|:-----|:---------|:-----|
| `centralized-data-service/internal/handler/command_handler.go` | Thêm `ensureCDCColumns()`, fix `HandleAirbyteBridge`, fix `bridgeInPlace`, fix `HandleBatchTransform`, fix `HandlePeriodicScan` | T1-T5 |

## SQL sẽ chạy tự động (bởi ensureCDCColumns)

```sql
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _raw_data JSONB;
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _source VARCHAR(20) DEFAULT 'airbyte';
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _synced_at TIMESTAMP DEFAULT NOW();
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _version BIGINT DEFAULT 1;
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _hash VARCHAR(64);
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE "{table}" ADD COLUMN IF NOT EXISTS _updated_at TIMESTAMP DEFAULT NOW();
```

## Rủi ro

1. **Airbyte overwrite**: Nếu Airbyte destination mode = `overwrite` + DROP TABLE → CDC columns bị mất. Cần monitor.
   - Mitigation: `ensureCDCColumns` chạy mỗi bridge cycle → tự heal
2. **Storage**: `_raw_data` JSONB duplicate data đã có trong typed columns
   - Mitigation: Sau transform hoàn tất → có thể SET `_raw_data = NULL` để tiết kiệm
