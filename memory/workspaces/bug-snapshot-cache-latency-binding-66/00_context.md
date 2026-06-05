# Workspace: bug-snapshot-cache-latency-binding-66

## Context & Objectives
Mục tiêu là giải quyết vấn đề trễ cache (stale cache) trong `snapshot.v2` khiến các cấu hình quy tắc mapping và masking của `shadow_binding_id` 66 bị bỏ qua (sử dụng cache cũ hoặc không nhận cấu hình mới).

### Vấn đề hiện tại:
1. Có sự lệch pha (race condition) giữa NATS schema reload signals và `SnapshotRunner` khi chạy snapshot ngay lập tức, dẫn tới `MetadataRegistryService` cache bị stale hoặc không đồng bộ kịp.
2. `MaskingService` lưu cấu hình sensitive fields trong `sync.Map` (`sensitiveFields`), nhưng khi registry reload (`ReloadAll`), `MaskingService` không tự giải phóng/invalidation cache này, dẫn tới `resolveMaskMap` trả về cấu hình stale cho binding 66.
3. Cần kiểm tra xem các masking rules được duyệt (`approved`) trong `mapping_rule_v2` có được `MaskingService` áp dụng đúng ở runtime không.
4. Cần củng cố tính nhất quán pre-flight của `SnapshotRunner` để ngăn chặn việc skip dữ liệu âm thầm (silent skip) và đảm bảo `DynamicMapper` luôn nhìn thấy các mapping rules mới nhất.

### Tác động:
- Khi snapshot, mặc dù field đã được approve (status = approved), dữ liệu tương ứng vẫn không được mapping qua hoặc bị masking sai cách do cache cũ.
