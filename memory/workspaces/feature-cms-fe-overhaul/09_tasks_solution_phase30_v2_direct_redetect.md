# Phase 30 Solution — V2 Direct Re-detect

## Giải pháp chốt
- Không cố kéo toàn bộ operator actions khỏi bridge cùng lúc.
- Chỉ tách `detect-timestamp-field` trước vì đây là action có contract worker đủ mềm:
  - CMS chỉ cần resolve `target_table`
  - worker đã support lookup theo `target_table` fallback

## Tác động
- Row `V2 Ready` trong `DataIntegrity` không còn bị chặn bởi `registry_id` chỉ để re-detect timestamp.
- Operator-flow thực chiến hơn: FE bớt “No Bridge = bó tay”.
- Bridge vẫn giữ cho các action chưa đủ điều kiện direct V2 như `create-default-columns`.

## Chưa làm trong phase này
- Không direct-V2 hóa `create-default-columns`
- Không direct-V2 hóa `scan-fields`
- Không thay worker payload sâu hơn mức cần thiết
