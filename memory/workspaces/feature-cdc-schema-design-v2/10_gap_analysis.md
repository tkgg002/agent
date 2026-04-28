# Gap Analysis

## Current V1 Gaps

1. `cdc_table_registry` đang gộp:
   - source identity
   - destination identity
   - runtime state
   - recon state
2. `target_table` đang bị dùng như canonical key.
3. Runtime chỉ có một system/sink DB pool.
4. `master_ddl_generator` và `transmuter` hardcode schema vật lý.
5. `mapping_rules` chưa bind đúng vào từng master binding.

## V2 Coverage

1. Tách control plane khỏi data plane.
2. Chuẩn hóa identity của source object.
3. Tách shadow/master routing thành binding.
4. Thêm multi-connection runtime manager.
5. Chuẩn bị phase migration an toàn.

## Remaining Work After This Design

1. Viết migration SQL thật.
2. Viết backfill script.
3. Refactor cache/repository/service.
4. Refactor event ingest / transmute / DDL / recon.
5. Viết integration test cho:
   - postgres source -> postgres shadow A -> postgres master B
   - mongodb source -> postgres shadow C -> postgres master D
