# Solution: Phase 2 — Dynamic Mapper Full

> Date: 2026-04-14
> Status: T1.1-T1.5 DONE, T1.6+ cần session mới

## Đã implement (T1.1-T1.5)

File: `centralized-data-service/internal/service/dynamic_mapper.go`

### T1.1 LoadRules() ✅
- Delegate to `RegistryService.ReloadAll()` — single source of truth, không duplicate cache

### T1.2 MapData() ✅
- Input: targetTable + rawData (map[string]interface{} — full document từ source)
- Output: MappedData { Columns (typed), EnrichedData (phase 2+), RawJSON (full JSON cho _raw_data) }
- Hỗ trợ nested fields (dot notation: "info.fee")
- Enriched fields routing (IsEnriched flag)

### T1.3 BuildUpsertQuery() ✅
- Dynamic INSERT...ON CONFLICT với parameterized values
- Include: pk + mapped columns + _raw_data + CDC metadata
- Hash comparison (WHERE _hash IS DISTINCT FROM EXCLUDED._hash)

### T1.4 convertType() ✅
- INT/BIGINT, NUMERIC/FLOAT, BOOLEAN, TIMESTAMP, JSONB, TEXT/VARCHAR
- MongoDB $date format support
- Unix timestamp (seconds + milliseconds)
- Graceful fallback: conversion fail → use raw value

### T1.5 StartConfigReloadListener() 
- Delegate to RegistryService (đã có NATS subscription trong worker_server.go)

## Chưa implement

### T1.6: Replace static mapping trong event_handler.go
- File: `internal/handler/event_handler.go`
- Hiện tại: `h.registrySvc.GetMappingRules(targetTable)` + manual mapping loop
- Cần: Thay bằng `dynamicMapper.MapData(ctx, targetTable, data)` + `BuildUpsertQuery()`
- DynamicMapper cần được inject vào EventHandler

### T1.7: Unit tests
- Test MapData() với các data types
- Test convertType() edge cases
- Test BuildUpsertQuery() output SQL

### T1.8: Build + verify runtime

## Architecture note

DynamicMapper dùng RegistryService (đã có RWMutex cache + hot reload). Không tạo cache riêng.

```
NATS event → EventHandler → DynamicMapper.MapData() → MappedData
                          → DynamicMapper.BuildUpsertQuery() → SQL
                          → BatchBuffer.Add(SQL, args)
```

Khi Debezium active (Step 3): CDC events chứa `fullDocument` JSON gốc → MapData nhận JSON gốc → `_raw_data` = toàn bộ data source → 100% không miss field mới.
