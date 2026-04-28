# Requirements — Phase 9 Master Binding Contract

## Bối cảnh

- `MasterRegistry` là page trọng yếu của `cms-fe operator-flow`.
- Sau Phase 8, `worker-schedule` đã có contract giàu context từ metadata V2.
- API `v1/masters` vẫn là điểm lệch lớn:
  - backend đọc/ghi `cdc_internal.master_table_registry`
  - FE vẫn bắt operator nhập `source_shadow` kiểu legacy

## Yêu cầu

1. `v1/masters` phải chuyển sang `cdc_system.master_binding`.
2. Contract phải đủ để operator thao tác theo namespace thật:
   - `master_schema`
   - `shadow_schema`
   - `shadow_table`
   - source context nếu có
3. Vẫn giữ compatibility tối thiểu với `source_shadow` để không gãy flow cũ ngay lập tức.
4. Khi đổi API phải cập nhật swagger/comment đồng thời.
